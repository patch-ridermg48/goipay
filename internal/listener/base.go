package listener

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/chekist32/goipay/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type transactionPoolSync struct {
	txs map[string]bool
}

type blockSync struct {
	lastBlockHeight atomic.Uint64
}

type DaemonRpcClientExecutor[T, B any] interface {
	syncBlock()
	syncTransactionPool()

	Start(startBlock uint64)
	Stop()
	NewBlockChan() <-chan B
	NewTxPoolChan() <-chan T
	LastSyncedBlockHeight() uint64
}

type baseDaemonRpcClientExecutor[T, B any] struct {
	log *zerolog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	txPoolChns   *util.SyncMapTypeSafe[string, chan T]
	newBlockChns *util.SyncMapTypeSafe[string, chan B]

	blockSync           blockSync
	transactionPoolSync transactionPoolSync
}

func (d *baseDaemonRpcClientExecutor[T, B]) broadcastNewBlock(block *B) {
	d.newBlockChns.Range(func(key string, cn chan B) bool {
		go func() {
			select {
			case cn <- *block:
				return
			case <-time.After(util.MIN_SYNC_TIMEOUT):
				d.newBlockChns.Delete(key)
				return
			}
		}()
		return true
	})
}

func (d *baseDaemonRpcClientExecutor[T, B]) broadcastNewTx(tx *T) {
	d.txPoolChns.Range(func(key string, cn chan T) bool {
		go func() {
			select {
			case cn <- *tx:
				return
			case <-time.After(util.MIN_SYNC_TIMEOUT):
				d.txPoolChns.Delete(key)
				return
			}
		}()
		return true
	})
}

func (d *baseDaemonRpcClientExecutor[T, B]) sync(drce DaemonRpcClientExecutor[T, B], blockTimeout time.Duration, txPoolTimeout time.Duration) {
	go func() {
		t := time.NewTicker(blockTimeout)
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-t.C:
				drce.syncBlock()
			}
		}
	}()

	go func() {
		t := time.NewTicker(txPoolTimeout)
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-t.C:
				drce.syncTransactionPool()
			}
		}
	}()
}

func (d *baseDaemonRpcClientExecutor[T, B]) start(drce DaemonRpcClientExecutor[T, B], startBlock uint64) {
	if d.ctx.Err() == nil {
		return
	}
	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.blockSync.lastBlockHeight.Store(startBlock)

	d.sync(drce, util.MIN_SYNC_TIMEOUT, util.MIN_SYNC_TIMEOUT/2)
}

func (d *baseDaemonRpcClientExecutor[T, B]) Stop() {
	d.cancel()
}

func (d *baseDaemonRpcClientExecutor[T, B]) NewBlockChan() <-chan B {
	cn := make(chan B)
	d.newBlockChns.Store(uuid.NewString(), cn)
	return cn
}

func (d *baseDaemonRpcClientExecutor[T, B]) NewTxPoolChan() <-chan T {
	cn := make(chan T)
	d.txPoolChns.Store(uuid.NewString(), cn)
	return cn
}

func (d *baseDaemonRpcClientExecutor[T, B]) LastSyncedBlockHeight() uint64 {
	return d.blockSync.lastBlockHeight.Load()
}

func newBaseDaemonRpcClientExecutor[T, B any](log *zerolog.Logger) baseDaemonRpcClientExecutor[T, B] {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return baseDaemonRpcClientExecutor[T, B]{
		log:                 log,
		ctx:                 ctx,
		cancel:              cancel,
		transactionPoolSync: transactionPoolSync{txs: make(map[string]bool)},
		txPoolChns:          &util.SyncMapTypeSafe[string, chan T]{},
		newBlockChns:        &util.SyncMapTypeSafe[string, chan B]{},
	}
}
