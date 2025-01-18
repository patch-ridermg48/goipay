package processor

import (
	"context"
	"errors"
	"time"

	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/dto"
	"github.com/chekist32/goipay/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const (
	persist_cache_timeout time.Duration = 1 * time.Minute
)

var (
	unimplementedError error = errors.New("coin is either unimplemented or not set up")
)

type PaymentProcessor struct {
	dbConnPool *pgxpool.Pool

	ctx context.Context
	log *zerolog.Logger

	invoiceCn      chan db.Invoice
	newInvoicesCns *util.SyncMapTypeSafe[string, chan db.Invoice]

	cryptoProcessors map[db.CoinType]cryptoProcessor
}

func (p *PaymentProcessor) loadPersistedPendingInvoices() error {
	q, tx, err := util.InitDbQueriesWithTx(p.ctx, p.dbConnPool)
	if err != nil {
		p.log.Err(err).Msg(util.DefaultFailedSqlTxInitMsg)
		return err
	}

	invoices, err := q.ShiftExpiresAtForNonConfirmedInvoices(p.ctx)
	if err != nil {
		tx.Rollback(p.ctx)
		p.log.Err(err).Str("queryName", "ShiftExpiresAtForNonConfirmedInvoices").Msg(util.DefaultFailedSqlQueryMsg)
		return err
	}

	tx.Commit(p.ctx)

	// TODO: Add impelmentation for LTC
	// TODO: Add impelmentation for ETH
	// TODO: Add impelmentation for TON
	for i := 0; i < len(invoices); i++ {
		if cp, ok := p.cryptoProcessors[invoices[i].Coin]; ok {
			cp.handleInvoice(p.ctx, invoices[i])
		}
	}

	return nil
}

func (p *PaymentProcessor) load() error {
	go func() {
		for {
			select {
			case tx := <-p.invoiceCn:
				p.newInvoicesCns.Range(func(key string, cn chan db.Invoice) bool {
					go func() {
						select {
						case cn <- tx:
							return
						case <-time.After(util.SEND_TIMEOUT):
							p.newInvoicesCns.Delete(key)
							return
						case <-p.ctx.Done():
							return
						}
					}()

					return true
				})

				p.log.Info().Msgf("Transaction %v changed status to %v", util.PgUUIDToString(tx.ID), tx.Status)
			case <-p.ctx.Done():
				return
			}
		}
	}()

	if err := p.loadPersistedPendingInvoices(); err != nil {
		return err
	}

	for _, v := range p.cryptoProcessors {
		if err := v.load(p.ctx); err != nil {
			return err
		}
	}

	return nil
}

func (p *PaymentProcessor) HandleNewInvoice(req *dto.NewInvoiceRequest) (*db.Invoice, error) {
	// TODO: Add impelmentation for TON
	// TODO: Add impelmentation for ETH
	// TODO: Add impelmentation for LTC
	if cp, ok := p.cryptoProcessors[req.Coin]; ok {
		return cp.handleInvoicePbReq(p.ctx, req)
	}

	return nil, unimplementedError
}

func (p *PaymentProcessor) NewInvoicesChan() <-chan db.Invoice {
	cn := make(chan db.Invoice)
	p.newInvoicesCns.Store(uuid.NewString(), cn)
	return cn
}

func NewPaymentProcessor(ctx context.Context, dbConnPool *pgxpool.Pool, c *dto.DaemonsConfig, log *zerolog.Logger) (*PaymentProcessor, error) {
	invoiceCn := make(chan db.Invoice)
	cryptoProcessors := make(map[db.CoinType]cryptoProcessor, 0)

	if c.Xmr.Url != "" {
		xmr, err := newXmrProcessor(log, dbConnPool, invoiceCn, c)
		if err != nil {
			return nil, err
		}
		cryptoProcessors[xmr.coin] = xmr
	}
	if c.Btc.Url != "" {
		btc, err := newBtcProcessor(log, dbConnPool, invoiceCn, c)
		if err != nil {
			return nil, err
		}
		cryptoProcessors[btc.coin] = btc
	}

	pp := &PaymentProcessor{
		dbConnPool:       dbConnPool,
		invoiceCn:        invoiceCn,
		newInvoicesCns:   &util.SyncMapTypeSafe[string, chan db.Invoice]{},
		cryptoProcessors: cryptoProcessors,
		ctx:              ctx,
		log:              log,
	}
	if err := pp.load(); err != nil {
		return nil, err
	}

	return pp, nil
}
