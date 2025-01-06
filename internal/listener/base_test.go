package listener

import (
	"context"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/util"
	"github.com/chekist32/goipay/test"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func TestBlockChan(t *testing.T) {
	t.Parallel()

	t.Run("Check NewBlockChan Func", func(t *testing.T) {
		mockClient := NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		mockClient.On("GetCoinType").Return(db.CoinTypeXMR)

		bdrce := NewBaseDaemonRpcClientExecutor(&zerolog.Logger{}, mockClient)

		expectedBlock := TestBlock{
			Height: rand.Uint64(),
		}

		blockCnAmount := atomic.Int32{}
		blockCnAmount.Store(0)

		blockCn := bdrce.NewBlockChan()
		bdrce.newBlockChns.Range(func(key string, value chan TestBlock) bool {
			go func() {
				blockCnAmount.Add(1)
				value <- expectedBlock
			}()

			return true
		})

		actualBlock := test.GetValueFromCnOrLogFatalWithTimeout(blockCn, util.MIN_SYNC_TIMEOUT, "Timeout has been expired")

		assert.Equal(t, int32(1), blockCnAmount.Load())
		assert.Equal(t, expectedBlock, actualBlock)
	})
}

func TestTxPoolChan(t *testing.T) {
	t.Parallel()

	t.Run("Check NewTxPoolChan Func", func(t *testing.T) {
		mockClient := NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		mockClient.On("GetCoinType").Return(db.CoinTypeXMR)

		bdrce := NewBaseDaemonRpcClientExecutor(&zerolog.Logger{}, mockClient)

		expectedTx := TestTx{
			TxId: uuid.NewString(),
		}

		txPoolCnAmount := atomic.Int32{}
		txPoolCnAmount.Store(0)

		txPoolCn := bdrce.NewTxPoolChan()
		bdrce.txPoolChns.Range(func(key string, value chan TestTx) bool {
			go func() {
				txPoolCnAmount.Add(1)
				value <- expectedTx
			}()

			return true
		})

		actualTx := test.GetValueFromCnOrLogFatalWithTimeout(txPoolCn, util.MIN_SYNC_TIMEOUT, "Timeout has been expired")

		assert.Equal(t, int32(1), txPoolCnAmount.Load())
		assert.Equal(t, expectedTx, actualTx)
	})
}

func TestStartStop(t *testing.T) {
	t.Parallel()

	mockClient := NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
	mockClient.On("GetCoinType").Return(db.CoinTypeXMR)

	tbcrce := NewBaseDaemonRpcClientExecutor(&zerolog.Logger{}, mockClient)

	tbcrce.Start(0)
	assert.NoError(t, tbcrce.ctx.Err())

	tbcrce.Start(0)
	assert.NoError(t, tbcrce.ctx.Err())

	tbcrce.Stop()
	assert.Error(t, tbcrce.ctx.Err())
}

func TestSyncBlock(t *testing.T) {
	t.Parallel()

	t.Run("Successful syncBlock", func(t *testing.T) {
		lastBlockHeight := rand.Uint64()
		d := NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetCoinType").Return(db.CoinTypeXMR)
		d.On("GetLastBlockHeight").Return(
			lastBlockHeight,
			error(nil),
		)

		expectedBlock := TestBlock{
			Height: lastBlockHeight - 1,
		}
		d.On("GetBlockByHeight", uint64(lastBlockHeight-1)).Return(
			expectedBlock,
			error(nil),
		)

		bdrce := NewBaseDaemonRpcClientExecutor(&zerolog.Logger{}, d)
		bdrce.blockSync.lastBlockHeight.Store(lastBlockHeight - 1)
		blockCn := bdrce.NewBlockChan()

		bdrce.ctx = context.Background()
		bdrce.syncBlock()

		actualBlock := test.GetValueFromCnOrLogFatalWithTimeout(blockCn, util.MIN_SYNC_TIMEOUT, "Timeout has been expired")

		assert.Equal(t, lastBlockHeight, bdrce.blockSync.lastBlockHeight.Load())
		assert.Equal(t, expectedBlock, actualBlock)
	})

	t.Run("MIN_SYNC_TIMEOUT exceeded", func(t *testing.T) {
		lastBlockHeight := rand.Uint64()
		d := NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetCoinType").Return(db.CoinTypeXMR)
		d.On("GetLastBlockHeight").Return(
			lastBlockHeight,
			error(nil),
		)

		expectedBlock := TestBlock{
			Height: lastBlockHeight - 1,
		}
		d.On("GetBlockByHeight", uint64(lastBlockHeight-1)).Return(
			expectedBlock,
			error(nil),
		)

		bdrce := NewBaseDaemonRpcClientExecutor(&zerolog.Logger{}, d)
		bdrce.blockSync.lastBlockHeight.Store(lastBlockHeight - 1)
		_ = bdrce.NewBlockChan()

		bdrce.ctx = context.Background()
		bdrce.syncBlock()

		<-time.After(util.MIN_SYNC_TIMEOUT + 1*time.Second)

		cnCount := 0
		bdrce.newBlockChns.Range(func(key string, value chan TestBlock) bool {
			cnCount++
			return true
		})

		assert.Equal(t, 0, cnCount)
	})

}

func TestSyncTransactionPool(t *testing.T) {
	t.Parallel()

	t.Run("Successful syncTransactionPool", func(t *testing.T) {
		// 1
		expectedTxs1Map := map[string]TestTx{
			"tx1": {TxId: "tx1"},
			"tx2": {TxId: "tx2"},
			"tx3": {TxId: "tx3"},
			"tx4": {TxId: "tx4"},
			"tx5": {TxId: "tx5"},
		}
		expectedTxHashes1Slice := make([]string, 0)
		for _, tx := range expectedTxs1Map {
			expectedTxHashes1Slice = append(expectedTxHashes1Slice, tx.GetTxId())
		}

		d := NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetCoinType").Return(db.CoinTypeXMR)
		d.On("GetTransactionPool").Once().Return(
			expectedTxHashes1Slice,
			error(nil),
		)
		d.On("GetTransactions", mock.Anything).Return(func(txHashes []string) ([]TestTx, error) {
			txs := make([]TestTx, 0, len(txHashes))
			for i := 0; i < len(txHashes); i++ {
				txs = append(txs, expectedTxs1Map[txHashes[i]])
			}
			return txs, nil
		}).Times(len(expectedTxHashes1Slice))

		bdrce := NewBaseDaemonRpcClientExecutor(&zerolog.Logger{}, d)
		txPoolCn := bdrce.NewTxPoolChan()

		bdrce.syncTransactionPool()

		txs1 := make(map[string]TestTx, 0)
		for i := 0; i < len(expectedTxs1Map); i++ {
			tx := test.GetValueFromCnOrLogFatalWithTimeout(txPoolCn, util.MIN_SYNC_TIMEOUT, "Timeout has been expired")
			txs1[tx.GetTxId()] = tx
		}

		assert.Equal(t, expectedTxs1Map, txs1)
		assert.Condition(t, func() (success bool) {
			for id := range expectedTxs1Map {
				if !bdrce.transactionPoolSync.txs[id] {
					return false
				}
			}

			return true
		})

		// 2
		expectedTxs2Map := map[string]TestTx{
			"tx1": {TxId: "tx1"},
			"tx3": {TxId: "tx3"},
			"tx5": {TxId: "tx5"},
			"tx7": {TxId: "tx7"},
			"tx6": {TxId: "tx6"},
		}
		expectedTxHashes2Slice := make([]string, 0)
		for _, tx := range expectedTxs2Map {
			expectedTxHashes2Slice = append(expectedTxHashes2Slice, tx.GetTxId())
		}
		d.On("GetTransactionPool").Return(
			expectedTxHashes2Slice,
			error(nil),
		)
		d.On("GetTransactions", mock.Anything).Return(func(txHashes []string) ([]TestTx, error) {
			txs := make([]TestTx, 0, len(txHashes))
			for i := 0; i < len(txHashes); i++ {
				txs = append(txs, expectedTxs2Map[txHashes[i]])
			}
			return txs, nil
		})

		bdrce.syncTransactionPool()

		txs2 := make(map[string]TestTx, 0)
		for i := 0; i < 2; i++ {
			tx := test.GetValueFromCnOrLogFatalWithTimeout(txPoolCn, util.MIN_SYNC_TIMEOUT, "Timeout has been expired")
			txs2[tx.GetTxId()] = tx
		}

		assert.Equal(t, map[string]TestTx{"tx7": {TxId: "tx7"}, "tx6": {TxId: "tx6"}}, txs2)
		assert.Condition(t, func() (success bool) {
			for id := range expectedTxs2Map {
				if !bdrce.transactionPoolSync.txs[id] {
					return false
				}
			}

			return true
		})
	})

	t.Run("MIN_SYNC_TIMEOUT exceeded", func(t *testing.T) {
		expectedTxs1Map := map[string]TestTx{
			"tx1": {TxId: "tx1"},
			"tx2": {TxId: "tx2"},
			"tx3": {TxId: "tx3"},
			"tx4": {TxId: "tx4"},
			"tx5": {TxId: "tx5"},
		}
		expectedTxHashes1Slice := make([]string, 0)
		for _, tx := range expectedTxs1Map {
			expectedTxHashes1Slice = append(expectedTxHashes1Slice, tx.GetTxId())
		}

		d := NewMockSharedDaemonRpcClient[TestTx, TestBlock](t)
		d.On("GetCoinType").Return(db.CoinTypeXMR)
		d.On("GetTransactionPool").Once().Return(
			expectedTxHashes1Slice,
			error(nil),
		)
		d.On("GetTransactions", mock.Anything).Return(func(txHashes []string) ([]TestTx, error) {
			txs := make([]TestTx, 0, len(txHashes))
			for i := 0; i < len(txHashes); i++ {
				txs = append(txs, expectedTxs1Map[txHashes[i]])
			}
			return txs, nil
		})

		bdrce := NewBaseDaemonRpcClientExecutor(&zerolog.Logger{}, d)
		_ = bdrce.NewTxPoolChan()

		bdrce.syncTransactionPool()
		<-time.After(util.MIN_SYNC_TIMEOUT + 1*time.Second)

		cnCount := 0
		bdrce.txPoolChns.Range(func(key string, value chan TestTx) bool {
			cnCount++
			return true
		})

		assert.Equal(t, 0, cnCount)
	})
}
