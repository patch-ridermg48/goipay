package listener

import (
	"github.com/chekist32/go-monero/daemon"
	"github.com/chekist32/goipay/internal/util"
	"github.com/rs/zerolog"
)

type XMRDaemonRpcClientExecutor struct {
	client daemon.IDaemonRpcClient

	baseDaemonRpcClientExecutor[daemon.MoneroTx, daemon.GetBlockResult]
}

func (d *XMRDaemonRpcClientExecutor) syncBlock() {
	height, err := d.client.GetLastBlockHeader(true)
	if err != nil {
		d.log.Err(err).Str("method", "last_block_header").Msg(util.DefaultFailedFetchingDaemonMsg)
		return
	}

	for {
		select {
		case <-d.ctx.Done():
			return
		default:
			if height.Result.BlockHeader.Height <= d.blockSync.lastBlockHeight.Load() {
				return
			}

			block, err := d.client.GetBlockByHeight(true, d.blockSync.lastBlockHeight.Load())
			if err != nil {
				d.log.Err(err).Str("method", "get_block").Msg(util.DefaultFailedFetchingDaemonMsg)
				return
			}
			d.log.Info().Msgf("Synced blockheight: %v", block.Result.BlockHeader.Height)

			d.broadcastNewBlock(&block.Result)

			d.blockSync.lastBlockHeight.Add(1)
		}
	}
}

func (d *XMRDaemonRpcClientExecutor) syncTransactionPool() {
	txs, err := d.client.GetTransactionPool()
	if err != nil {
		d.log.Err(err).Str("method", "get_transaction_pool").Msg(util.DefaultFailedFetchingDaemonMsg)
		return
	}

	fetchedTxs := txs.Transactions
	prevTxs := d.transactionPoolSync.txs
	newTxs := make(map[string]bool)

	for i := 0; i < len(fetchedTxs); i++ {
		newTxs[fetchedTxs[i].IdHash] = true

		if prevTxs[fetchedTxs[i].IdHash] {
			continue
		}

		d.broadcastNewTx(&fetchedTxs[i])
	}

	d.transactionPoolSync.txs = newTxs
}

func (d *XMRDaemonRpcClientExecutor) Start(startBlock uint64) {
	d.baseDaemonRpcClientExecutor.start(d, startBlock)
}

func NewXMRDaemonRpcClientExecutor(client daemon.IDaemonRpcClient, log *zerolog.Logger) *XMRDaemonRpcClientExecutor {
	return &XMRDaemonRpcClientExecutor{
		baseDaemonRpcClientExecutor: newBaseDaemonRpcClientExecutor[daemon.MoneroTx, daemon.GetBlockResult](log),
		client:                      client,
	}
}
