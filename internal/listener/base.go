package listener

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type NetworkType uint8

const (
	MainnetXMR NetworkType = iota
	StagenetXMR
	TestnetXMR

	MainnetBTC
	TestnetBTC
	RegtestBTC
	SignetBTC

	MainnetLTC
	TestnetLTC
	RegtestLTC
	SignetLTC
)

type transactionPoolSync struct {
	txs map[string]bool
}

type blockSync struct {
	lastBlockHeight atomic.Uint64
}

type SharedTx interface {
	GetTxId() string
	GetConfirmations() uint64
	IsDoubleSpendSeen() bool
}

type SharedBlock interface {
	GetTxHashes() []string
}

type SharedDaemonRpcClient[T SharedTx, B SharedBlock] interface {
	GetLastBlockHeight() (uint64, error)
	GetBlockByHeight(height uint64) (B, error)
	GetTransactionPool() ([]string, error)
	GetTransactions(txHashes []string) ([]T, error)
	GetNetworkType() (NetworkType, error)
	GetCoinType() db.CoinType
}

type DaemonRpcClientExecutor[T SharedTx, B SharedBlock] interface {
	Start(startBlock uint64)
	Stop()
	NewBlockChan() <-chan B
	NewTxPoolChan() <-chan T
	LastSyncedBlockHeight() uint64
}

type BaseDaemonRpcClientExecutor[T SharedTx, B SharedBlock] struct {
	log *zerolog.Logger

	ctx    context.Context
	cancel context.CancelFunc

	coin db.CoinType

	txPoolChns   *util.SyncMapTypeSafe[string, chan T]
	newBlockChns *util.SyncMapTypeSafe[string, chan B]

	blockSync           blockSync
	transactionPoolSync transactionPoolSync

	client SharedDaemonRpcClient[T, B]
}

func (d *BaseDaemonRpcClientExecutor[T, B]) broadcastNewBlock(block *B) {
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

func (d *BaseDaemonRpcClientExecutor[T, B]) broadcastNewTx(tx *T) {
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

func (d *BaseDaemonRpcClientExecutor[T, B]) syncBlock() {
	height, err := d.client.GetLastBlockHeight()
	if err != nil {
		d.log.Err(err).Str("method", "GetLastBlockHeight").Str("coin", string(d.coin)).Msg(util.DefaultFailedFetchingDaemonMsg)
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

			block, err := d.client.GetBlockByHeight(d.blockSync.lastBlockHeight.Load())
			if err != nil {
				d.log.Err(err).Str("method", "GetBlockByHeight").Str("coin", string(d.coin)).Msg(util.DefaultFailedFetchingDaemonMsg)
				return
			}
			d.log.Info().Str("coin", string(d.coin)).Msgf("Synced blockheight: %v", height)

			d.broadcastNewBlock(&block)

			d.blockSync.lastBlockHeight.Add(1)
		}
	}
}

func (d *BaseDaemonRpcClientExecutor[T, B]) syncTransactionPool() {
	txHashes, err := d.client.GetTransactionPool()
	if err != nil {
		d.log.Err(err).Str("method", "GetTransactionPool").Str("coin", string(d.coin)).Msg(util.DefaultFailedFetchingDaemonMsg)
		return
	}

	prevTxs := d.transactionPoolSync.txs
	newTxs := make(map[string]bool)

	for i := 0; i < len(txHashes); i++ {
		newTxs[txHashes[i]] = true

		if prevTxs[txHashes[i]] {
			continue
		}

		tx, err := d.client.GetTransactions([]string{txHashes[i]})
		if err != nil || len(tx) < 1 {
			d.log.Err(err).Str("method", "GetTransactions").Str("coin", string(d.coin)).Msg(util.DefaultFailedFetchingDaemonMsg)
			continue
		}

		d.broadcastNewTx(&tx[0])
	}

	d.transactionPoolSync.txs = newTxs

}

func (d *BaseDaemonRpcClientExecutor[T, B]) sync(blockTimeout time.Duration, txPoolTimeout time.Duration) {
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

func (d *BaseDaemonRpcClientExecutor[T, B]) Start(startBlock uint64) {
	if d.ctx.Err() == nil {
		return
	}
	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.blockSync.lastBlockHeight.Store(startBlock)

	d.sync(util.MIN_SYNC_TIMEOUT, util.MIN_SYNC_TIMEOUT/2)
}

func (d *BaseDaemonRpcClientExecutor[T, B]) Stop() {
	d.cancel()
}

func (d *BaseDaemonRpcClientExecutor[T, B]) NewBlockChan() <-chan B {
	cn := make(chan B)
	d.newBlockChns.Store(uuid.NewString(), cn)
	return cn
}

func (d *BaseDaemonRpcClientExecutor[T, B]) NewTxPoolChan() <-chan T {
	cn := make(chan T)
	d.txPoolChns.Store(uuid.NewString(), cn)
	return cn
}

func (d *BaseDaemonRpcClientExecutor[T, B]) LastSyncedBlockHeight() uint64 {
	return d.blockSync.lastBlockHeight.Load()
}

func NewBaseDaemonRpcClientExecutor[T SharedTx, B SharedBlock](log *zerolog.Logger, client SharedDaemonRpcClient[T, B]) *BaseDaemonRpcClientExecutor[T, B] {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	return &BaseDaemonRpcClientExecutor[T, B]{
		log:                 log,
		ctx:                 ctx,
		cancel:              cancel,
		coin:                client.GetCoinType(),
		client:              client,
		transactionPoolSync: transactionPoolSync{txs: make(map[string]bool)},
		txPoolChns:          &util.SyncMapTypeSafe[string, chan T]{},
		newBlockChns:        &util.SyncMapTypeSafe[string, chan B]{},
	}
}
