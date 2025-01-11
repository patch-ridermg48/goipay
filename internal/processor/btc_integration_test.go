package processor

import (
	"context"
	"fmt"
	"log"
	"math"
	"testing"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/listener"
	"github.com/chekist32/goipay/test"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
)

func createUserWithBtcData(ctx context.Context, q *db.Queries) (pgtype.UUID, db.CryptoDatum, db.BtcCryptoDatum) {
	userId, err := q.CreateUser(ctx)
	if err != nil {
		log.Fatal(err)
	}
	btcData, err := q.CreateBTCCryptoData(ctx, "tpubDCUURn3yPT4P3SkrUq9rG1RyJK6BGhmrovvSAF61LHLCZhNUMRw7FANPmhGuDWXo3GMkc6C4ZFGBuPMrovjdnXhtJfQE3uK3s6QzFuiQaz9")
	if err != nil {
		log.Fatal(err)
	}
	cd, err := q.CreateCryptoData(ctx, db.CreateCryptoDataParams{BtcID: btcData.ID, UserID: userId})
	if err != nil {
		log.Fatal(err)
	}

	return userId, cd, btcData
}

func createNewTestBtcDaemon() *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         "rpc.ankr.com/btc_signet/abe7e45a955c46d234a46d1f4534dfeff3a7bdeaf2cee815afd69c0e3949df32",
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   false,
	}
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func TestGenerateNextBtcAddressHandler(t *testing.T) {
	t.Parallel()

	data := []struct {
		prevMajorIndex int32
		prevMinorIndex int32
		expectedAddr   string
	}{
		{
			prevMajorIndex: 0,
			prevMinorIndex: 0,
			expectedAddr:   "tb1qqdcfs9s5gjsnmazcsqfe2h6gwzwdu2eufesk8h",
		},
		{
			prevMajorIndex: 0,
			prevMinorIndex: 124,
			expectedAddr:   "tb1qg3hmjm7waaj3ejem5qt9kzcm40qppmq49lgpnk",
		},
		{
			prevMajorIndex: 0,
			prevMinorIndex: math.MaxInt32,
			expectedAddr:   "tb1q4lnztm5gjh3jqeahl00gk85aprqcm9vdl3gzr8",
		},
		{
			prevMajorIndex: 1,
			prevMinorIndex: 2,
			expectedAddr:   "tb1qwfu6lpvaxekpz7qjdlfvwrs44yaq02gj509pxn",
		},
	}

	ctx := context.Background()

	dbConn, _, close := getPostgresWithDbConn()
	defer close(ctx)

	for _, d := range data {
		t.Run(fmt.Sprintf("Should Return Valid Address Ma %v Mi %v", d.prevMajorIndex, d.prevMinorIndex), func(t *testing.T) {
			test.RunInTransaction(t, dbConn, func(t *testing.T, tx pgx.Tx) {
				// Given
				q := db.New(dbConn).WithTx(tx)
				userId, cd, _ := createUserWithBtcData(ctx, q)

				_, err := q.UpdateIndicesBTCCryptoDataById(ctx, db.UpdateIndicesBTCCryptoDataByIdParams{ID: cd.BtcID, LastMajorIndex: d.prevMajorIndex, LastMinorIndex: d.prevMinorIndex})
				if err != nil {
					log.Fatal(err)
				}

				// When
				addr, err := generateNextBTCAddressHandler(ctx, q, &generateNextAddressHandlerData{userId: userId, network: listener.SignetBTC})

				// Assert
				assert.NoError(t, err)
				assert.Equal(t, d.expectedAddr, addr.Address)
			})
		})
	}

}

func TestVerifyBTCTxHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	dbConn, _, close := getPostgresWithDbConn()
	defer close(ctx)

	daemon := listener.NewSharedBTCDaemonRpcClient(createNewTestBtcDaemon())

	t.Run("Should Return Right Amount (Valid Tx)", func(t *testing.T) {
		test.RunInTransaction(t, dbConn, func(t *testing.T, tx pgx.Tx) {
			// Given
			q := db.New(dbConn).WithTx(tx)
			userId, _, _ := createUserWithXmrData(ctx, q)

			expectedTxId := "ea217d3fd466c86d4fea6759b6b12cc62ec9a13b30042941e03ae84e6d748e9f"
			expectedInvoice, err := q.CreateInvoice(ctx, db.CreateInvoiceParams{
				UserID:                userId,
				Coin:                  db.CoinTypeBTC,
				CryptoAddress:         "tb1qpmtec0cq470g9uwsjdhkjvzzczusynsz4ltejd",
				RequiredAmount:        0.00019522,
				ExpiresAt:             pgtype.Timestamptz{Time: time.Now().Add(time.Minute), Valid: true},
				ConfirmationsRequired: 0,
			})
			if err != nil {
				log.Fatal(err)
			}

			txs, err := daemon.GetTransactions([]string{expectedTxId})
			if err != nil {
				log.Fatal(err)
			} else if len(txs) < 1 {
				log.Fatalf("Invalid Tx Id: %v", expectedTxId)
			}

			// When
			amount, err := verifyBTCTxHandler(ctx, q, &verifyTxHandlerData[listener.BTCTx]{invoice: expectedInvoice, tx: txs[0]})

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, expectedInvoice.RequiredAmount, amount)
		})
	})

	t.Run("Should Return 0 Amount (Invalid Tx)", func(t *testing.T) {
		test.RunInTransaction(t, dbConn, func(t *testing.T, tx pgx.Tx) {
			// Given
			q := db.New(dbConn).WithTx(tx)
			userId, _, _ := createUserWithBtcData(ctx, q)

			expectedTxId := "b6cf23d11dfd4551107b8022a25fafaf0f421c569765a7347c59cedfa6b3007c"
			expectedInvoice, err := q.CreateInvoice(ctx, db.CreateInvoiceParams{
				UserID:                userId,
				Coin:                  db.CoinTypeBTC,
				CryptoAddress:         "tb1qpmtec0cq470g9uwsjdhkjvzzczusynsz4ltejd",
				RequiredAmount:        0.00019522,
				ExpiresAt:             pgtype.Timestamptz{Time: time.Now().Add(time.Minute), Valid: true},
				ConfirmationsRequired: 0,
			})
			if err != nil {
				log.Fatal(err)
			}

			txs, err := daemon.GetTransactions([]string{expectedTxId})
			if err != nil {
				log.Fatal(err)
			} else if len(txs) < 1 {
				log.Fatalf("Invalid Tx Id: %v", expectedTxId)
			}

			// When
			amount, err := verifyBTCTxHandler(ctx, q, &verifyTxHandlerData[listener.BTCTx]{invoice: expectedInvoice, tx: txs[0]})

			// Assert
			assert.NoError(t, err)
			assert.Equal(t, float64(0), amount)
		})
	})
}
