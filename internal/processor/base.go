package processor

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/dto"
	"github.com/chekist32/goipay/internal/listener"
	"github.com/chekist32/goipay/internal/util"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var (
	unsupportedCoin error = errors.New("coin is unsupported by crypto processor")
)

type pendingInvoice struct {
	invoice           *atomic.Pointer[db.Invoice]
	cancelTimeoutFunc context.CancelFunc
}

type verifyTxHandlerData[T listener.SharedTx] struct {
	invoice db.Invoice
	tx      T
}

type generateNextAddressHandlerData struct {
	userId  pgtype.UUID
	network listener.NetworkType
}

type cryptoProcessor interface {
	load(ctx context.Context) error
	handleInvoicePbReq(ctx context.Context, req *dto.NewInvoiceRequest) (*db.Invoice, error)
	handleInvoice(ctx context.Context, invoice db.Invoice)
	supportsCoin(coin db.CoinType) bool
}

type baseCryptoProcessor[T listener.SharedTx, B listener.SharedBlock] struct {
	log *zerolog.Logger

	dbConnPool *pgxpool.Pool

	daemon   listener.SharedDaemonRpcClient[T, B]
	daemonEx listener.DaemonRpcClientExecutor[T, B]
	network  listener.NetworkType

	coin            db.CoinType
	supportedTokens map[db.CoinType]bool

	invoiceCn       chan<- db.Invoice
	pendingInvoices *util.SyncMapTypeSafe[string, pendingInvoice]

	verifyTxHandler            func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[T]) (float64, error)
	generateNextAddressHandler func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error)
}

func (b *baseCryptoProcessor[T, B]) verifyTxOnMempool(ctx context.Context, cryptoTx T) {
	if cryptoTx.IsDoubleSpendSeen() {
		return
	}

	b.pendingInvoices.Range(func(key string, value pendingInvoice) bool {
		go func() {
			q, tx, err := util.InitDbQueriesWithTx(ctx, b.dbConnPool)
			if err != nil {
				b.log.Err(err).Str("coin", string(b.coin)).Msg(util.DefaultFailedSqlTxInitMsg)
				return
			}
			defer tx.Rollback(ctx)

			invoice := value.invoice.Load()

			amount, err := b.verifyTxHandler(ctx, q, &verifyTxHandlerData[T]{invoice: *invoice, tx: cryptoTx})
			if err != nil {
				b.log.Err(err).Str("coin", string(b.coin)).Msg("An error occurred while verifying the tx output.")
				return
			}

			if invoice.RequiredAmount <= amount && invoice.Status == db.InvoiceStatusTypePENDING {
				b.confirmPENDING_MEMPOOL(ctx, q, cryptoTx, amount, value)
				b.confirmCONFIRMED(ctx, q, value)
			}

			tx.Commit(ctx)
		}()

		return true
	})
}

func (b *baseCryptoProcessor[T, B]) confirmPENDING_MEMPOOL(ctx context.Context, q *db.Queries, cryptoTx T, am float64, value pendingInvoice) {
	var txId pgtype.Text
	if err := txId.Scan(cryptoTx.GetTxId()); err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("fieldName", "txId").Msg(util.DefaultFailedScanningToPostgresqlDataTypeMsg)
		return
	}

	var amount pgtype.Float8
	if err := amount.Scan(am); err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("fieldName", "amount").Msg(util.DefaultFailedScanningToPostgresqlDataTypeMsg)
		return
	}

	invoice, err := q.ConfirmInvoiceStatusMempoolById(ctx, db.ConfirmInvoiceStatusMempoolByIdParams{ID: value.invoice.Load().ID, ActualAmount: amount, TxID: txId})
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("queryName", "ConfirmInvoiceStatusMempoolById").Msg(util.DefaultFailedSqlQueryMsg)
		return
	}

	value.invoice.Store(&invoice)
	b.broadcastUpdatedInvoice(ctx, &invoice)
}

func (b *baseCryptoProcessor[T, B]) confirmCONFIRMED(ctx context.Context, q *db.Queries, value pendingInvoice) {
	invoice := value.invoice.Load()
	if !invoice.TxID.Valid {
		return
	}

	txs, err := b.daemon.GetTransactions([]string{invoice.TxID.String})
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("method", "get_transactions").Msg(util.DefaultFailedFetchingDaemonMsg)
		return
	}
	if len(txs) < 1 || txs[0].IsDoubleSpendSeen() {
		b.log.Info().Str("coin", string(b.coin)).Msgf("Tx %v was rejected by blockchain", invoice.TxID.String)
		b.expireInvoice(ctx, invoice)
		return
	}

	if uint64(invoice.ConfirmationsRequired) > txs[0].GetConfirmations() {
		return
	}
	if _, loaded := b.pendingInvoices.LoadAndDelete(invoice.CryptoAddress); !loaded {
		return
	}
	value.cancelTimeoutFunc()

	confirmedInvoice, err := q.ConfirmInvoiceById(ctx, invoice.ID)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("queryName", "ConfirmInvoiceById").Msg(util.DefaultFailedSqlQueryMsg)
		return
	}

	go b.releaseAddressHelper(ctx, invoice)
	b.broadcastUpdatedInvoice(ctx, &confirmedInvoice)
}

func (b *baseCryptoProcessor[T, B]) verifyTxOnNewBlock(ctx context.Context) {
	b.pendingInvoices.Range(func(key string, value pendingInvoice) bool {
		go func() {
			q, tx, err := util.InitDbQueriesWithTx(ctx, b.dbConnPool)
			if err != nil {
				b.log.Err(err).Str("coin", string(b.coin)).Msg(util.DefaultFailedSqlTxInitMsg)
				return
			}
			b.confirmCONFIRMED(ctx, q, value)
			tx.Commit(ctx)
		}()
		return true
	})
}

func (b *baseCryptoProcessor[T, B]) persistCryptoCache(ctx context.Context) {
	q, tx, err := util.InitDbQueriesWithTx(ctx, b.dbConnPool)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Msg(util.DefaultFailedSqlTxInitMsg)
		return
	}
	defer tx.Rollback(ctx)

	var height pgtype.Int8
	if err := height.Scan(int64(b.daemonEx.LastSyncedBlockHeight())); err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("fieldName", "height").Msg(util.DefaultFailedScanningToPostgresqlDataTypeMsg)
		return
	}

	if _, err := q.UpdateCryptoCacheByCoin(ctx, db.UpdateCryptoCacheByCoinParams{Coin: b.coin, LastSyncedBlockHeight: height}); err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("queryName", "UpdateCryptoCacheByCoin").Msg(util.DefaultFailedSqlQueryMsg)
		return
	}

	tx.Commit(ctx)
}

func (b *baseCryptoProcessor[T, B]) load(ctx context.Context) error {
	q, tx, err := util.InitDbQueriesWithTx(ctx, b.dbConnPool)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Msg(util.DefaultFailedSqlTxInitMsg)
		return err
	}
	defer tx.Rollback(ctx)

	cache, err := q.FindCryptoCacheByCoin(ctx, b.coin)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("queryName", "FindCryptoCacheByCoin").Msg(util.DefaultFailedSqlQueryMsg)
		return err
	}

	h, err := b.daemon.GetLastBlockHeight()
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("method", "GetLastBlockHeight").Msg(util.DefaultFailedFetchingDaemonMsg)
		return err
	}

	height := int64(h)
	if cache.LastSyncedBlockHeight.Valid {
		height = cache.LastSyncedBlockHeight.Int64
	}

	tx.Commit(ctx)

	go func() {
		blockCn := b.daemonEx.NewBlockChan()

		for {
			select {
			case block := <-blockCn:
				go func() {
					txs, err := b.daemon.GetTransactions(block.GetTxHashes())
					if err != nil {
						b.log.Err(err).Str("coin", string(b.coin)).Str("method", "GetTransactions").Msg(util.DefaultFailedFetchingDaemonMsg)
						return
					}

					for i := 0; i < len(txs); i++ {
						go b.verifyTxOnMempool(ctx, txs[i])
					}

				}()

				go b.verifyTxOnNewBlock(ctx)

			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		txPoolCn := b.daemonEx.NewTxPoolChan()

		for {
			select {
			case tx := <-txPoolCn:
				go b.verifyTxOnMempool(ctx, tx)
			case <-ctx.Done():
				return
			}
		}
	}()

	b.daemonEx.Start(uint64(height))

	go func() {
		b.persistCryptoCache(ctx)

		for {
			select {
			case <-time.After(persist_cache_timeout):
				go b.persistCryptoCache(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (b *baseCryptoProcessor[T, B]) createInvoice(ctx context.Context, req *dto.NewInvoiceRequest) (*db.Invoice, error) {
	q, tx, err := util.InitDbQueriesWithTx(ctx, b.dbConnPool)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Msg(util.DefaultFailedSqlTxInitMsg)
		return nil, err
	}
	defer tx.Rollback(ctx)

	var userId pgtype.UUID
	if err := userId.Scan(req.UserId); err != nil {
		return nil, err
	}

	coin := req.Coin

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout < util.MIN_SYNC_TIMEOUT {
		timeout = util.MIN_SYNC_TIMEOUT
	}

	var expiresAt pgtype.Timestamptz
	if err := expiresAt.Scan(time.Now().UTC().Add(timeout)); err != nil {
		return nil, err
	}

	addr, err := q.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoin(ctx, db.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoinParams{UserID: userId, Coin: coin})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		addr, err = b.generateNextAddressHandler(ctx, q, &generateNextAddressHandlerData{userId: userId, network: b.network})
		if err != nil {
			return nil, err
		}
	}

	invoice, err := q.CreateInvoice(
		ctx,
		db.CreateInvoiceParams{
			CryptoAddress:         addr.Address,
			Coin:                  coin,
			RequiredAmount:        req.Amount,
			ConfirmationsRequired: int16(req.Confirmations),
			ExpiresAt:             expiresAt,
			UserID:                userId,
		},
	)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("queryName", "CreateInvoice").Msg(util.DefaultFailedSqlQueryMsg)
		return nil, err
	}

	tx.Commit(ctx)

	return &invoice, nil
}

func (b *baseCryptoProcessor[T, B]) handleInvoicePbReq(ctx context.Context, req *dto.NewInvoiceRequest) (*db.Invoice, error) {
	if !b.supportsCoin(req.Coin) {
		return nil, unsupportedCoin
	}

	invoice, err := b.createInvoice(ctx, req)
	if err != nil {
		return nil, err
	}

	b.handleInvoice(ctx, *invoice)
	b.broadcastUpdatedInvoice(ctx, invoice)

	return invoice, nil
}

func (b *baseCryptoProcessor[T, B]) releaseAddressHelper(ctx context.Context, invoice *db.Invoice) {
	q, tx, err := util.InitDbQueriesWithTx(ctx, b.dbConnPool)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Msg(util.DefaultFailedSqlTxInitMsg)
		return
	}
	defer tx.Rollback(ctx)

	if _, err := q.UpdateIsOccupiedByCryptoAddress(ctx, db.UpdateIsOccupiedByCryptoAddressParams{IsOccupied: false, Address: invoice.CryptoAddress}); err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("queryName", "UpdateIsOccupiedByCryptoAddress").Msg(util.DefaultFailedSqlQueryMsg)
		return
	}

	tx.Commit(ctx)
}

func (b *baseCryptoProcessor[T, B]) broadcastUpdatedInvoice(ctx context.Context, invoice *db.Invoice) {
	go func() {
		timeoutCtx, cancel := context.WithTimeout(ctx, util.SEND_TIMEOUT)
		defer cancel()

		select {
		case b.invoiceCn <- *invoice:
			b.log.Debug().Str("coin", string(b.coin)).Str("invoiceId", util.PgUUIDToString(invoice.ID)).Msg("Invoice broadcasted")
			return
		case <-timeoutCtx.Done():
			b.log.Debug().Str("coin", string(b.coin)).Str("invoiceId", util.PgUUIDToString(invoice.ID)).Msg("Timeout expired")
			return
		}
	}()
}

func (b *baseCryptoProcessor[T, B]) expireInvoice(ctx context.Context, invoice *db.Invoice) {
	if _, loaded := b.pendingInvoices.LoadAndDelete(invoice.CryptoAddress); !loaded {
		return
	}

	q, tx, err := util.InitDbQueriesWithTx(ctx, b.dbConnPool)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Msg(util.DefaultFailedSqlTxInitMsg)
		return
	}
	defer tx.Rollback(ctx)

	expiredInvoice, err := q.ExpireInvoiceById(ctx, invoice.ID)
	if err != nil {
		b.log.Err(err).Str("coin", string(b.coin)).Str("queryName", "ExpireInvoiceById").Msg(util.DefaultFailedSqlQueryMsg)
		return
	}

	go b.releaseAddressHelper(ctx, invoice)
	b.broadcastUpdatedInvoice(ctx, &expiredInvoice)

	tx.Commit(ctx)
}

func (b *baseCryptoProcessor[T, B]) handleInvoiceHelper(confirmedInvoiceCtx context.Context, invoice *db.Invoice) {
	select {
	case <-time.After(invoice.ExpiresAt.Time.Sub(time.Now().UTC())):
		b.expireInvoice(confirmedInvoiceCtx, invoice)
		return
	case <-confirmedInvoiceCtx.Done():
		return
	}
}
func (b *baseCryptoProcessor[T, B]) handleInvoice(ctx context.Context, invoice db.Invoice) {
	if _, ok := b.pendingInvoices.Load(invoice.CryptoAddress); ok {
		return
	}

	confirmedInvoiceCtx, cancel := context.WithCancel(ctx)

	invoicePtr := &atomic.Pointer[db.Invoice]{}
	invoicePtr.Store(&invoice)
	b.pendingInvoices.Store(invoice.CryptoAddress, pendingInvoice{invoice: invoicePtr, cancelTimeoutFunc: cancel})

	go b.handleInvoiceHelper(confirmedInvoiceCtx, &invoice)
}

func (b *baseCryptoProcessor[T, B]) supportsCoin(coin db.CoinType) bool {
	return b.coin == coin || b.supportedTokens[coin]
}

func newBaseCryptoProcessor[T listener.SharedTx, B listener.SharedBlock](
	log *zerolog.Logger,
	dbConnPool *pgxpool.Pool,
	invoiceCn chan<- db.Invoice,
	daemon listener.SharedDaemonRpcClient[T, B],
	verifyTxHandler func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[T]) (float64, error),
	generateNextAddressHandler func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error),
	supportedTokens []db.CoinType,
) (*baseCryptoProcessor[T, B], error) {
	net, err := daemon.GetNetworkType()
	if err != nil {
		return nil, err
	}

	return &baseCryptoProcessor[T, B]{
			log:                        log,
			dbConnPool:                 dbConnPool,
			invoiceCn:                  invoiceCn,
			network:                    net,
			daemon:                     daemon,
			daemonEx:                   listener.NewBaseDaemonRpcClientExecutor(log, daemon),
			coin:                       daemon.GetCoinType(),
			supportedTokens:            util.SliceToSet(supportedTokens),
			pendingInvoices:            new(util.SyncMapTypeSafe[string, pendingInvoice]),
			verifyTxHandler:            verifyTxHandler,
			generateNextAddressHandler: generateNextAddressHandler,
		},
		nil
}
