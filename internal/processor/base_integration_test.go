package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/dto"
	"github.com/chekist32/goipay/internal/listener"
	"github.com/chekist32/goipay/internal/util"
	"github.com/chekist32/goipay/test"
	db_test "github.com/chekist32/goipay/test/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
)

type TestTx struct {
	TxId          string
	Confirmations uint64
}

func (t TestTx) GetTxId() string {
	return t.TxId
}
func (t TestTx) GetConfirmations() uint64 {
	return t.Confirmations
}
func (t TestTx) IsDoubleSpendSeen() bool {
	return false
}

type TestBlock struct {
	Height uint64
}

func (b TestBlock) GetTxHashes() []string {
	return nil
}

func getInvoiceOrFatal(ctx context.Context, q *db_test.Queries, id *pgtype.UUID) db.Invoice {
	invoice, err := q.FindInvoiceById(ctx, *id)
	if err != nil {
		log.Fatal(err)
	}

	return db.Invoice{
		ID:                    invoice.ID,
		CryptoAddress:         invoice.CryptoAddress,
		Coin:                  db.CoinType(invoice.Coin),
		RequiredAmount:        invoice.RequiredAmount,
		ActualAmount:          invoice.ActualAmount,
		ConfirmationsRequired: invoice.ConfirmationsRequired,
		CreatedAt:             invoice.CreatedAt,
		ConfirmedAt:           invoice.ConfirmedAt,
		Status:                db.InvoiceStatusType(invoice.Status),
		ExpiresAt:             invoice.ExpiresAt,
		TxID:                  invoice.TxID,
		UserID:                invoice.UserID,
	}
}

func createNewTestBaseCryptoProcessor[T listener.SharedTx, B listener.SharedBlock](
	daemon listener.SharedDaemonRpcClient[T, B],
	verifyTxHandler func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[T]) (float64, error),
	generateNextAddressHandler func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error),
) (chan db.Invoice, *baseCryptoProcessor[T, B], testcontainers.Container, func(ctx context.Context)) {
	invoiceCn := make(chan db.Invoice)
	dbConn, postgres, close := test.SpinUpPostgresContainerAndGetPgxpool(fmt.Sprintf("%v/../../sql/migrations", os.Getenv("PWD")))

	base, err := newBaseCryptoProcessor(
		&zerolog.Logger{},
		dbConn,
		invoiceCn,
		daemon,
		verifyTxHandler,
		generateNextAddressHandler,
	)
	if err != nil {
		log.Fatal(err)
	}

	return invoiceCn, base, postgres, close
}

func TestBroadcastInvoice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Should Receive The Invoice", func(t *testing.T) {
		// Given
		d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
		d.On("GetCoinType").Return(db.CoinTypeXMR)

		invoiceCn, p, _, close := createNewTestBaseCryptoProcessor(
			d,
			func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
				return 0, nil
			},
			func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
				return db.CryptoAddress{}, nil
			},
		)
		defer close(ctx)

		q := db.New(p.dbConnPool)
		userId, err := q.CreateUser(ctx)
		if err != nil {
			log.Fatal(err)
		}
		expectedAddr, err := q.CreateCryptoAddress(ctx, db.CreateCryptoAddressParams{
			Address:    uuid.NewString(),
			Coin:       db.CoinTypeXMR,
			IsOccupied: true,
			UserID:     userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		var expiresAt pgtype.Timestamptz
		if err := expiresAt.Scan(time.Now().UTC()); err != nil {
			log.Fatal(err)
		}
		invoice, err := q.CreateInvoice(ctx, db.CreateInvoiceParams{
			CryptoAddress:         expectedAddr.Address,
			Coin:                  expectedAddr.Coin,
			RequiredAmount:        1.0,
			ConfirmationsRequired: 0,
			ExpiresAt:             expiresAt,
			UserID:                userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		// When
		p.broadcastUpdatedInvoice(ctx, &invoice)
		receivedInvoice := test.GetValueFromCnOrLogFatalWithTimeout(invoiceCn, util.MIN_SYNC_TIMEOUT, "Timeout expired")

		// Assert
		assert.Equal(t, invoice, receivedInvoice)
	})
}

func TestReleaseAddressHelper(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Should Properly Release The Address", func(t *testing.T) {
		// Given
		d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
		d.On("GetCoinType").Return(db.CoinTypeXMR)

		_, p, _, close := createNewTestBaseCryptoProcessor(
			d,
			func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
				return 0, nil
			},
			func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
				return db.CryptoAddress{}, nil
			},
		)
		defer close(ctx)

		q := db.New(p.dbConnPool)
		userId, err := q.CreateUser(ctx)
		if err != nil {
			log.Fatal(err)
		}
		expectedAddr, err := q.CreateCryptoAddress(ctx, db.CreateCryptoAddressParams{
			Address:    uuid.NewString(),
			Coin:       db.CoinTypeXMR,
			IsOccupied: true,
			UserID:     userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		var expiresAt pgtype.Timestamptz
		if err := expiresAt.Scan(time.Now().UTC()); err != nil {
			log.Fatal(err)
		}
		invoice, err := q.CreateInvoice(ctx, db.CreateInvoiceParams{
			CryptoAddress:         expectedAddr.Address,
			Coin:                  expectedAddr.Coin,
			RequiredAmount:        1.0,
			ConfirmationsRequired: 0,
			ExpiresAt:             expiresAt,
			UserID:                userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		// When/Assert
		_, err = q.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoin(ctx, db.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoinParams{UserID: invoice.UserID, Coin: invoice.Coin})
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		p.releaseAddressHelper(ctx, &invoice)

		addr, err := q.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoin(ctx, db.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoinParams{UserID: invoice.UserID, Coin: invoice.Coin})
		assert.NoError(t, err)
		assert.Equal(t, expectedAddr, addr)
	})
}

func TestExpireInvoice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Should Properly Expire The Invoice And Release The Address", func(t *testing.T) {
		// Given
		d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
		d.On("GetCoinType").Return(db.CoinTypeXMR)
		invoiceCn, p, _, close := createNewTestBaseCryptoProcessor(
			d,
			func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
				return 0, nil
			},
			func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
				return db.CryptoAddress{}, nil
			},
		)
		defer close(ctx)

		q := db.New(p.dbConnPool)
		qTest := db_test.New(p.dbConnPool)
		userId, err := q.CreateUser(ctx)
		if err != nil {
			log.Fatal(err)
		}
		expectedAddr, err := q.CreateCryptoAddress(ctx, db.CreateCryptoAddressParams{
			Address:    uuid.NewString(),
			Coin:       db.CoinTypeXMR,
			IsOccupied: true,
			UserID:     userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		var expiresAt pgtype.Timestamptz
		if err := expiresAt.Scan(time.Now().UTC()); err != nil {
			log.Fatal(err)
		}
		expectedPendingInvoice, err := q.CreateInvoice(ctx, db.CreateInvoiceParams{
			CryptoAddress:         expectedAddr.Address,
			Coin:                  expectedAddr.Coin,
			RequiredAmount:        1.0,
			ConfirmationsRequired: 0,
			ExpiresAt:             expiresAt,
			UserID:                userId,
		})
		if err != nil {
			log.Fatal(err)
		}
		invoicePtr := &atomic.Pointer[db.Invoice]{}
		invoicePtr.Store(&expectedPendingInvoice)
		_, cancel := context.WithCancel(ctx)
		p.pendingInvoices.Store(expectedPendingInvoice.CryptoAddress, pendingInvoice{invoice: invoicePtr, cancelTimeoutFunc: cancel})

		// When/Assert
		_, err = q.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoin(ctx, db.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoinParams{UserID: expectedPendingInvoice.UserID, Coin: expectedPendingInvoice.Coin})
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		p.expireInvoice(ctx, &expectedPendingInvoice)

		expiredInvoice := test.GetValueFromCnOrLogFatalWithTimeout(invoiceCn, util.MIN_SYNC_TIMEOUT, "Timeout expired")
		assert.Equal(t, expectedPendingInvoice.ID, expiredInvoice.ID)
		assert.Equal(t, db.InvoiceStatusTypeEXPIRED, expiredInvoice.Status)

		// wait for releaseAddressHelper
		<-time.After(300 * time.Millisecond)

		addr, err := q.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoin(ctx, db.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoinParams{UserID: expectedPendingInvoice.UserID, Coin: expectedPendingInvoice.Coin})
		assert.NoError(t, err)
		assert.Equal(t, expectedAddr, addr)

		expiredInvoice = getInvoiceOrFatal(ctx, qTest, &expectedPendingInvoice.ID)
		assert.Equal(t, expectedPendingInvoice.ID, expiredInvoice.ID)
		assert.Equal(t, db.InvoiceStatusTypeEXPIRED, expiredInvoice.Status)
		assert.False(t, expiredInvoice.ConfirmedAt.Valid)
	})
}

func TestPersistCryptoCache(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Should Properly Persist Crypto Cache", func(t *testing.T) {
		// Given
		expectedCoin := db.CoinTypeXMR
		expectedLastHeight := uint64(123)
		d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
		d.On("GetCoinType").Return(expectedCoin)
		d.On("GetLastBlockHeight").Return(expectedLastHeight, error(nil)).Maybe()
		d.On("GetTransactionPool").Return([]string{}, error(nil)).Maybe()

		_, p, _, close := createNewTestBaseCryptoProcessor(
			d,
			func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
				return 0, nil
			},
			func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
				return db.CryptoAddress{}, nil
			},
		)
		defer close(ctx)

		// When/Assert
		q := db.New(p.dbConnPool)

		cache, err := q.FindCryptoCacheByCoin(ctx, expectedCoin)
		if err != nil {
			log.Fatal(err)
		}
		assert.False(t, cache.LastSyncedBlockHeight.Valid)

		p.daemonEx.Start(expectedLastHeight)
		p.persistCryptoCache(ctx)

		cache, err = q.FindCryptoCacheByCoin(ctx, expectedCoin)
		assert.NoError(t, err)
		assert.True(t, cache.LastSyncedBlockHeight.Valid)
		assert.Equal(t, int64(expectedLastHeight), cache.LastSyncedBlockHeight.Int64)
	})
}

func TestHandleInvoice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	t.Run("Should Delete Expired Invoice From The Pending Invoices", func(t *testing.T) {
		// Given
		d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
		d.On("GetCoinType").Return(db.CoinTypeXMR)

		invoiceCn, p, _, close := createNewTestBaseCryptoProcessor(
			d,
			func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
				return 0, nil
			},
			func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
				return db.CryptoAddress{}, nil
			},
		)
		defer close(ctx)

		q := db.New(p.dbConnPool)
		qTest := db_test.New(p.dbConnPool)
		userId, err := q.CreateUser(ctx)
		if err != nil {
			log.Fatal(err)
		}
		expectedAddr, err := q.CreateCryptoAddress(ctx, db.CreateCryptoAddressParams{
			Address:    uuid.NewString(),
			Coin:       db.CoinTypeXMR,
			IsOccupied: true,
			UserID:     userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		var expiresAt pgtype.Timestamptz
		if err := expiresAt.Scan(time.Now().UTC()); err != nil {
			log.Fatal(err)
		}
		expectedPendingInvoice, err := q.CreateInvoice(ctx, db.CreateInvoiceParams{
			CryptoAddress:         expectedAddr.Address,
			Coin:                  expectedAddr.Coin,
			RequiredAmount:        1.0,
			ConfirmationsRequired: 0,
			ExpiresAt:             expiresAt,
			UserID:                userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		// When
		p.handleInvoice(ctx, expectedPendingInvoice)

		// Assert
		_ = test.GetValueFromCnOrLogFatalWithTimeout(invoiceCn, util.MIN_SYNC_TIMEOUT, "Timeout expired")
		<-time.After(300 * time.Millisecond)

		_, ok := p.pendingInvoices.Load(expectedPendingInvoice.CryptoAddress)
		assert.False(t, ok)

		addr, err := q.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoin(ctx, db.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoinParams{UserID: expectedPendingInvoice.UserID, Coin: expectedPendingInvoice.Coin})
		assert.NoError(t, err)
		assert.Equal(t, expectedAddr, addr)

		expiredInvoice := getInvoiceOrFatal(ctx, qTest, &expectedPendingInvoice.ID)
		assert.Equal(t, expectedPendingInvoice.ID, expiredInvoice.ID)
		assert.Equal(t, db.InvoiceStatusTypeEXPIRED, expiredInvoice.Status)
		assert.False(t, expiredInvoice.ConfirmedAt.Valid)
	})

	t.Run("Should Return From handleInvoiceHelper", func(t *testing.T) {
		// Given
		d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
		d.On("GetCoinType").Return(db.CoinTypeXMR)

		_, p, _, close := createNewTestBaseCryptoProcessor(
			d,
			func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
				return 0, nil
			},
			func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
				return db.CryptoAddress{}, nil
			},
		)
		defer close(ctx)

		q := db.New(p.dbConnPool)
		qTest := db_test.New(p.dbConnPool)
		userId, err := q.CreateUser(ctx)
		if err != nil {
			log.Fatal(err)
		}
		expectedAddr, err := q.CreateCryptoAddress(ctx, db.CreateCryptoAddressParams{
			Address:    uuid.NewString(),
			Coin:       db.CoinTypeXMR,
			IsOccupied: true,
			UserID:     userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		var expiresAt pgtype.Timestamptz
		if err := expiresAt.Scan(time.Now().UTC().Add(1 * time.Hour)); err != nil {
			log.Fatal(err)
		}
		expectedPendingInvoice, err := q.CreateInvoice(ctx, db.CreateInvoiceParams{
			CryptoAddress:         expectedAddr.Address,
			Coin:                  expectedAddr.Coin,
			RequiredAmount:        1.0,
			ConfirmationsRequired: 0,
			ExpiresAt:             expiresAt,
			UserID:                userId,
		})
		if err != nil {
			log.Fatal(err)
		}

		// When
		cancelCtx, cancel := context.WithCancel(ctx)

		p.handleInvoice(cancelCtx, expectedPendingInvoice)
		cancel()

		// Assert
		_, ok := p.pendingInvoices.Load(expectedPendingInvoice.CryptoAddress)
		assert.True(t, ok)

		_, err = q.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoin(ctx, db.FindNonOccupiedCryptoAddressAndLockByUserIdAndCoinParams{UserID: expectedPendingInvoice.UserID, Coin: expectedPendingInvoice.Coin})
		assert.ErrorIs(t, err, pgx.ErrNoRows)

		pendingInvoice := getInvoiceOrFatal(ctx, qTest, &expectedPendingInvoice.ID)
		assert.Equal(t, expectedPendingInvoice, pendingInvoice)
	})
}

func TestCreateInvoice(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Given
	expectedAddress := uuid.NewString()
	expectedUserId, err := util.StringToPgUUID(uuid.NewString())
	if err != nil {
		log.Fatal(err)
	}
	d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
	d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
	d.On("GetCoinType").Return(db.CoinTypeXMR)

	_, p, _, close := createNewTestBaseCryptoProcessor(
		d,
		func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
			return 0, nil
		},
		func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
			return q.CreateCryptoAddress(ctx, db.CreateCryptoAddressParams{
				Address:    expectedAddress,
				Coin:       db.CoinTypeXMR,
				IsOccupied: true,
				UserID:     data.userId,
			})
		},
	)
	defer close(ctx)

	q := db.New(p.dbConnPool)
	if _, err := q.CreateUserWithId(ctx, *expectedUserId); err != nil {
		log.Fatal(err)
	}

	// When
	createdInvoice, err := p.createInvoice(ctx, &dto.NewInvoiceRequest{
		UserId:        util.PgUUIDToString(*expectedUserId),
		Coin:          db.CoinTypeXMR,
		Amount:        123,
		Timeout:       600,
		Confirmations: 0,
	})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, db.CoinTypeXMR, createdInvoice.Coin)
	assert.EqualValues(t, 123, createdInvoice.RequiredAmount)
	assert.EqualValues(t, 0, createdInvoice.ConfirmationsRequired)
	assert.Equal(t, expectedAddress, createdInvoice.CryptoAddress)
}

func TestHandleInvoicePbReq(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Given
	d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
	d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
	d.On("GetCoinType").Return(db.CoinTypeXMR)
	invCn, p, _, close := createNewTestBaseCryptoProcessor(
		d,
		func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
			return 0, nil
		},
		func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
			return db.CryptoAddress{}, nil
		},
	)
	defer close(ctx)

	q := db.New(p.dbConnPool)
	qTest := db_test.New(p.dbConnPool)
	userId, _, _ := createUserWithXmrData(ctx, q)

	req := &dto.NewInvoiceRequest{
		UserId:        util.PgUUIDToString(userId),
		Coin:          db.CoinTypeXMR,
		Amount:        123,
		Timeout:       600,
		Confirmations: 0,
	}

	// When
	invoice, err := p.handleInvoicePbReq(ctx, req)

	// Assert
	assert.NoError(t, err)

	expectedInvoice := getInvoiceOrFatal(ctx, qTest, &invoice.ID)

	assert.Equal(t, expectedInvoice, *invoice)

	pendingInvoice, ok := p.pendingInvoices.Load(expectedInvoice.CryptoAddress)
	assert.True(t, ok)
	assert.Equal(t, expectedInvoice, *pendingInvoice.invoice.Load())

	invoiceFromCn := test.GetValueFromCnOrLogFatalWithTimeout(invCn, util.SEND_TIMEOUT, "Timeout expired")
	assert.Equal(t, expectedInvoice, invoiceFromCn)
}

func TestLoad(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Given
	expectedLastHeight := uint64(1124)
	d := listener.NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
	d.On("GetNetworkType").Return(listener.StagenetXMR, error(nil))
	d.On("GetCoinType").Return(db.CoinTypeXMR)
	d.On("GetLastBlockHeight").Return(expectedLastHeight, error(nil))
	d.On("GetTransactionPool").Return([]string{}, error(nil)).Maybe()
	_, p, _, close := createNewTestBaseCryptoProcessor(
		d,
		func(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[TestTx]) (float64, error) {
			return 0, nil
		},
		func(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
			return db.CryptoAddress{}, nil
		},
	)
	defer close(ctx)

	q := db.New(p.dbConnPool)

	// When
	err := p.load(ctx)

	// Assert
	assert.NoError(t, err)

	<-time.After(300 * time.Millisecond)
	cache, err := q.FindCryptoCacheByCoin(ctx, db.CoinTypeXMR)
	if err != nil {
		log.Fatal(err)
	}

	assert.True(t, cache.LastSyncedBlockHeight.Valid)
	assert.Equal(t, int64(expectedLastHeight), cache.LastSyncedBlockHeight.Int64)
}
