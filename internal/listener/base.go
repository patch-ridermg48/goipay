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

type sharedTx interface {
	getTxId() string
}

type sharedDaemonRpcClient[T sharedTx, B any] interface {
	getLastBlockHeight() (uint64, error)
	getBlockByHeight(height uint64) (B, error)
	getTransactionPool() ([]T, error)
}

type DaemonRpcClientExecutor[T, B any] interface {
	Start(startBlock uint64)
	Stop()
	NewBlockChan() <-chan B
	NewTxPoolChan() <-chan T
	LastSyncedBlockHeight() uint64
}

type baseDaemonRpcClientExecutor[T sharedTx, B any] struct {
	log *zerolog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	txPoolChns   *util.SyncMapTypeSafe[string, chan T]
	newBlockChns *util.SyncMapTypeSafe[string, chan B]

	blockSync           blockSync
	transactionPoolSync transactionPoolSync

	client sharedDaemonRpcClient[T, B]
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

func (d *baseDaemonRpcClientExecutor[T, B]) syncBlock() {
	height, err := d.client.getLastBlockHeight()
	if err != nil {
		d.log.Err(err).Str("method", "getLastBlockHeight").Msg(util.DefaultFailedFetchingDaemonMsg)
		return
	}

	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			if height <= d.blockSync.lastBlockHeight.Load() {
				return
			}

			block, err := d.client.getBlockByHeight(d.blockSync.lastBlockHeight.Load())
			if err != nil {
				d.log.Err(err).Str("method", "getBlockByHeight").Msg(util.DefaultFailedFetchingDaemonMsg)
				return
			}
			d.log.Info().Msgf("Synced blockheight: %v", height)

			d.broadcastNewBlock(&block)

			d.blockSync.lastBlockHeight.Add(1)
		}
	}
}

func (d *baseDaemonRpcClientExecutor[T, B]) syncTransactionPool() {
	txs, err := d.client.getTransactionPool()
	if err != nil {
		d.log.Err(err).Str("method", "getTransactionPool").Msg(util.DefaultFailedFetchingDaemonMsg)
		return
	}

	prevTxs := d.transactionPoolSync.txs
	newTxs := make(map[string]bool)

	for i := 0; i < len(txs); i++ {
		newTxs[txs[i].getTxId()] = true

		if prevTxs[txs[i].getTxId()] {
			continue
		}

		d.broadcastNewTx(&txs[i])
	}

	d.transactionPoolSync.txs = newTxs
}

func (d *baseDaemonRpcClientExecutor[T, B]) sync(blockTimeout time.Duration, txPoolTimeout time.Duration) {
	go func() {
		t := time.NewTicker(blockTimeout)
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-t.C:
				d.syncBlock()
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
				d.syncTransactionPool()
			}
		}
	}()
}

func (d *baseDaemonRpcClientExecutor[T, B]) Start(startBlock uint64) {
	if d.ctx.Err() == nil {
		return
	}
	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.blockSync.lastBlockHeight.Store(startBlock)

	d.sync(util.MIN_SYNC_TIMEOUT, util.MIN_SYNC_TIMEOUT/2)
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

func newBaseDaemonRpcClientExecutor[T sharedTx, B any](log *zerolog.Logger, client sharedDaemonRpcClient[T, B]) *baseDaemonRpcClientExecutor[T, B] {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return &baseDaemonRpcClientExecutor[T, B]{
		log:                 log,
		ctx:                 ctx,
		cancel:              cancel,
		client:              client,
		transactionPoolSync: transactionPoolSync{txs: make(map[string]bool)},
		txPoolChns:          &util.SyncMapTypeSafe[string, chan T]{},
		newBlockChns:        &util.SyncMapTypeSafe[string, chan B]{},
	}
}
