package listener

import (
	"math/rand"
	"sync/atomic"
	"testing"

	"github.com/chekist32/goipay/internal/util"
	"github.com/chekist32/goipay/test"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type testTx struct {
	TxId string
}

type testBlock struct {
	Height uint64
}

type testDaemonRpcClientExecutor struct {
	baseDaemonRpcClientExecutor[testTx, testBlock]
}

func (t *testDaemonRpcClientExecutor) syncBlock() {
}
func (t *testDaemonRpcClientExecutor) syncTransactionPool() {
}
func (t *testDaemonRpcClientExecutor) Start(startBlock uint64) {
	t.baseDaemonRpcClientExecutor.start(t, startBlock)
}

func TestBlockChan(t *testing.T) {
	t.Parallel()

	t.Run("Check NewBlockChan Func", func(t *testing.T) {
		bdrce := newBaseDaemonRpcClientExecutor[testTx, testBlock](&zerolog.Logger{})

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
		bdrce := newBaseDaemonRpcClientExecutor[testTx, testBlock](&zerolog.Logger{})

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

	tbcrce := &testDaemonRpcClientExecutor{
		baseDaemonRpcClientExecutor: newBaseDaemonRpcClientExecutor[testTx, testBlock](&zerolog.Logger{}),
	}

	tbcrce.Start(0)
	assert.NoError(t, tbcrce.ctx.Err())

	tbcrce.start(tbcrce, 0)
	assert.NoError(t, tbcrce.ctx.Err())

	tbcrce.Stop()
	assert.Error(t, tbcrce.ctx.Err())
}
