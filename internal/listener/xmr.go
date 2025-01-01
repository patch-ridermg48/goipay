package listener

import (
	"github.com/chekist32/go-monero/daemon"
	"github.com/rs/zerolog"
)

type XMRTx daemon.MoneroTx

func (t XMRTx) getTxId() string {
	return t.IdHash
}

type XMRBlock daemon.GetBlockResult

type sharedXMRDaemonRpcClient struct {
	client daemon.IDaemonRpcClient
}

func (c *sharedXMRDaemonRpcClient) getLastBlockHeight() (uint64, error) {
	res, err := c.client.GetLastBlockHeader(true)
	if err != nil {
		return 0, err
	}

	return res.Result.BlockHeader.Height, nil
}
func (c *sharedXMRDaemonRpcClient) getBlockByHeight(height uint64) (XMRBlock, error) {
	res, err := c.client.GetBlockByHeight(true, height)
	if err != nil {
		return XMRBlock{}, err
	}

	return XMRBlock(res.Result), nil
}
func (c *sharedXMRDaemonRpcClient) getTransactionPool() ([]XMRTx, error) {
	res, err := c.client.GetTransactionPool()
	if err != nil {
		return nil, err
	}

	txs := make([]XMRTx, 0, len(res.Transactions))
	for i := 0; i < len(res.Transactions); i++ {
		txs = append(txs, XMRTx(res.Transactions[i]))
	}

	return txs, nil
}

type XMRDaemonRpcClientExecutor struct {
	baseDaemonRpcClientExecutor[XMRTx, XMRBlock]
}

func NewXMRDaemonRpcClientExecutor(client daemon.IDaemonRpcClient, log *zerolog.Logger) *XMRDaemonRpcClientExecutor {
	return &XMRDaemonRpcClientExecutor{
		baseDaemonRpcClientExecutor: *newBaseDaemonRpcClientExecutor(log, &sharedXMRDaemonRpcClient{client: client}),
	}
}
