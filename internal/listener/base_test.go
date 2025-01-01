package listener

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chekist32/goipay/internal/util"
	"github.com/chekist32/goipay/test"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testTx struct {
	TxId string
}

func (b testTx) getTxId() string {
	return b.TxId
}

type testBlock struct {
	Height uint64
}

type mockSharedDaemonRpcClient struct {
	mock.Mock
}

func (m *mockSharedDaemonRpcClient) getLastBlockHeight() (uint64, error) {
	args := m.Called()
	return args.Get(0).(uint64), args.Error(1)
}
func (m *mockSharedDaemonRpcClient) getBlockByHeight(height uint64) (testBlock, error) {
	args := m.Called(height)
	return args.Get(0).(testBlock), args.Error(1)
}
func (m *mockSharedDaemonRpcClient) getTransactionPool() ([]testTx, error) {
	args := m.Called()
	return args.Get(0).([]testTx), args.Error(1)
}

func TestBlockChan(t *testing.T) {
	t.Parallel()

	t.Run("Check NewBlockChan Func", func(t *testing.T) {
		bdrce := newBaseDaemonRpcClientExecutor[testTx, testBlock](&zerolog.Logger{}, &mockSharedDaemonRpcClient{})

		expectedBlock := testBlock{
			Height: rand.Uint64(),
		}

		blockCnAmount := atomic.Int32{}
		blockCnAmount.Store(0)

		blockCn := bdrce.NewBlockChan()
		bdrce.newBlockChns.Range(func(key string, value chan testBlock) bool {
			go func() {
				blockCnAmount.Add(1)
				value <- expectedBlock
			}()

			return true
		})

		actualBlock := test.GetValueFromCnOrLogFatalWithTimeout[testBlock](blockCn, util.MIN_SYNC_TIMEOUT, "Timeout has been expired")

		assert.Equal(t, int32(1), blockCnAmount.Load())
		assert.Equal(t, expectedBlock, actualBlock)
	})
}

func TestTxPoolChan(t *testing.T) {
	t.Parallel()

	t.Run("Check NewTxPoolChan Func", func(t *testing.T) {
		bdrce := newBaseDaemonRpcClientExecutor[testTx, testBlock](&zerolog.Logger{}, &mockSharedDaemonRpcClient{})

		expectedTx := testTx{
			TxId: uuid.NewString(),
		}

		txPoolCnAmount := atomic.Int32{}
		txPoolCnAmount.Store(0)

		txPoolCn := bdrce.NewTxPoolChan()
		bdrce.txPoolChns.Range(func(key string, value chan testTx) bool {
			go func() {
				txPoolCnAmount.Add(1)
				value <- expectedTx
			}()

			return true
		})

		actualTx := test.GetValueFromCnOrLogFatalWithTimeout[testTx](txPoolCn, util.MIN_SYNC_TIMEOUT, "Timeout has been expired")

		assert.Equal(t, int32(1), txPoolCnAmount.Load())
		assert.Equal(t, expectedTx, actualTx)
	})
}

func TestStartStop(t *testing.T) {
	t.Parallel()

	tbcrce := newBaseDaemonRpcClientExecutor[testTx, testBlock](&zerolog.Logger{}, &mockSharedDaemonRpcClient{})

	tbcrce.Start(0)
	assert.NoError(t, tbcrce.ctx.Err())

	tbcrce.Start(0)
	assert.NoError(t, tbcrce.ctx.Err())

	tbcrce.Stop()
	assert.Error(t, tbcrce.ctx.Err())
}

func TestSyncBlock(t *testing.T) {
	t.Parallel()

	t.Run("Succesfull syncBlock", func(t *testing.T) {
		lastBlockHeight := rand.Uint64()
		d := new(mockSharedDaemonRpcClient)
		d.On("getLastBlockHeight").Return(
			lastBlockHeight,
			error(nil),
		)

		expectedBlock := testBlock{
			Height: lastBlockHeight - 1,
		}
		d.On("getBlockByHeight", uint64(lastBlockHeight-1)).Return(
			expectedBlock,
			error(nil),
		)

		bdrce := newBaseDaemonRpcClientExecutor(&zerolog.Logger{}, d)
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
		d := new(mockSharedDaemonRpcClient)
		d.On("getLastBlockHeight").Return(
			lastBlockHeight,
			error(nil),
		)

		expectedBlock := testBlock{
			Height: lastBlockHeight - 1,
		}
		d.On("getBlockByHeight", uint64(lastBlockHeight-1)).Return(
			expectedBlock,
			error(nil),
		)

		bdrce := newBaseDaemonRpcClientExecutor(&zerolog.Logger{}, d)
		bdrce.blockSync.lastBlockHeight.Store(lastBlockHeight - 1)
		_ = bdrce.NewBlockChan()

		bdrce.ctx = context.Background()
		bdrce.syncBlock()

		<-time.After(util.MIN_SYNC_TIMEOUT + 1*time.Second)

		cnCount := 0
		bdrce.newBlockChns.Range(func(key string, value chan testBlock) bool {
			cnCount++
			return true
		})

		assert.Equal(t, 0, cnCount)
	})

}

func TestSyncTransactionPool(t *testing.T) {
	t.Parallel()

	t.Run("Succesfull syncTransactionPool", func(t *testing.T) {
		// 1
		expectedTxs1Map := map[string]testTx{
			"tx1": {TxId: "tx1"},
			"tx2": {TxId: "tx2"},
			"tx3": {TxId: "tx3"},
			"tx4": {TxId: "tx4"},
			"tx5": {TxId: "tx5"},
		}
		expectedTxs1Slice := make([]testTx, 0)
		for _, tx := range expectedTxs1Map {
			expectedTxs1Slice = append(expectedTxs1Slice, tx)
		}

		d := new(mockSharedDaemonRpcClient)
		d.On("getTransactionPool").Once().Return(
			expectedTxs1Slice,
			error(nil),
		)

		bdrce := newBaseDaemonRpcClientExecutor(&zerolog.Logger{}, d)
		txPoolCn := bdrce.NewTxPoolChan()

		bdrce.syncTransactionPool()

		txs1 := make(map[string]testTx, 0)
		for i := 0; i < len(expectedTxs1Map); i++ {
			select {
			case tx := <-txPoolCn:
				txs1[tx.getTxId()] = tx
			case <-time.After(util.MIN_SYNC_TIMEOUT):
				log.Fatal(errors.New("Timeout has been expired"))
			}
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
		expectedTxs2Map := map[string]testTx{
			"tx1": {TxId: "tx1"},
			"tx3": {TxId: "tx3"},
			"tx5": {TxId: "tx5"},
			"tx7": {TxId: "tx7"},
			"tx6": {TxId: "tx6"},
		}
		expectedTxs2Slice := make([]testTx, 0)
		for _, tx := range expectedTxs2Map {
			expectedTxs2Slice = append(expectedTxs2Slice, tx)
		}
		d.On("getTransactionPool").Return(
			expectedTxs2Slice,
			error(nil),
		)

		bdrce.syncTransactionPool()

		txs2 := make(map[string]testTx, 0)
		for i := 0; i < 2; i++ {
			select {
			case tx := <-txPoolCn:
				txs2[tx.getTxId()] = tx
			case <-time.After(util.MIN_SYNC_TIMEOUT):
				log.Fatal(errors.New("Timeout has been expired"))
			}
		}

		assert.Equal(t, map[string]testTx{"tx7": {TxId: "tx7"}, "tx6": {TxId: "tx6"}}, txs2)
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
		expectedTxs1Map := map[string]testTx{
			"tx1": {TxId: "tx1"},
			"tx2": {TxId: "tx2"},
			"tx3": {TxId: "tx3"},
			"tx4": {TxId: "tx4"},
			"tx5": {TxId: "tx5"},
		}
		expectedTxs1Slice := make([]testTx, 0)
		for _, tx := range expectedTxs1Map {
			expectedTxs1Slice = append(expectedTxs1Slice, tx)
		}

		d := new(mockSharedDaemonRpcClient)
		d.On("getTransactionPool").Once().Return(
			expectedTxs1Slice,
			error(nil),
		)

		bdrce := newBaseDaemonRpcClientExecutor(&zerolog.Logger{}, d)
		_ = bdrce.NewTxPoolChan()

		bdrce.syncTransactionPool()
		<-time.After(util.MIN_SYNC_TIMEOUT + 1*time.Second)

		cnCount := 0
		bdrce.txPoolChns.Range(func(key string, value chan testTx) bool {
			cnCount++
			return true
		})

		assert.Equal(t, 0, cnCount)
	})
}
