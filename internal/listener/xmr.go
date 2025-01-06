package listener

import (
	"github.com/chekist32/go-monero/daemon"
	"github.com/chekist32/goipay/internal/db"
	"github.com/rs/zerolog"
)

type XMRTx daemon.MoneroTx1

func (t XMRTx) GetTxId() string {
	return t.TxHash
}
func (t XMRTx) GetConfirmations() uint64 {
	return t.Confirmations
}
func (t XMRTx) IsDoubleSpendSeen() bool {
	return t.DoubleSpendSeen
}

type XMRBlock daemon.GetBlockResult

func (b XMRBlock) GetTxHashes() []string {
	return b.BlockDetails.TxHashes
}

type SharedXMRDaemonRpcClient struct {
	client daemon.IDaemonRpcClient
}

func (c *SharedXMRDaemonRpcClient) GetLastBlockHeight() (uint64, error) {
	res, err := c.client.GetLastBlockHeader(true)
	if err != nil {
		return 0, err
	}

	return res.Result.BlockHeader.Height, nil
}
func (c *SharedXMRDaemonRpcClient) GetBlockByHeight(height uint64) (XMRBlock, error) {
	res, err := c.client.GetBlockByHeight(true, height)
	if err != nil {
		return XMRBlock{}, err
	}

	return XMRBlock(res.Result), nil
}
func (c *SharedXMRDaemonRpcClient) GetTransactionPool() ([]string, error) {
	res, err := c.client.GetTransactionPool()
	if err != nil {
		return nil, err
	}

	txHashes := make([]string, 0, len(res.Transactions))
	for i := 0; i < len(res.Transactions); i++ {
		txHashes = append(txHashes, res.Transactions[i].IdHash)
	}

	return txHashes, nil
}

func (c *SharedXMRDaemonRpcClient) GetTransactions(txHashes []string) ([]XMRTx, error) {
	res, err := c.client.GetTransactions(txHashes, true, false, false)
	if err != nil {
		return nil, err
	}

	txs := make([]XMRTx, 0, len(res.Txs))
	for i := 0; i < len(res.Txs); i++ {
		txs = append(txs, XMRTx(res.Txs[i]))
	}

	return txs, nil
}

func (c *SharedXMRDaemonRpcClient) GetNetworkType() (NetworkType, error) {
	res, err := c.client.GetInfo()
	if err != nil {
		return 255, err
	}

	net := MainnetXMR
	if res.Result.Testnet {
		net = TestnetXMR
	} else if res.Result.Stagenet {
		net = StagenetXMR
	}

	return net, nil
}

func (c *SharedXMRDaemonRpcClient) GetCoinType() db.CoinType {
	return db.CoinTypeXMR
}

type XMRDaemonRpcClientExecutor struct {
	BaseDaemonRpcClientExecutor[XMRTx, XMRBlock]
}

func NewXMRDaemonRpcClientExecutor(log *zerolog.Logger, client daemon.IDaemonRpcClient) *XMRDaemonRpcClientExecutor {
	return &XMRDaemonRpcClientExecutor{
		BaseDaemonRpcClientExecutor: *NewBaseDaemonRpcClientExecutor(log, &SharedXMRDaemonRpcClient{client: client}),
	}
}

func NewSharedXMRDaemonRpcClient(client daemon.IDaemonRpcClient) *SharedXMRDaemonRpcClient {
	return &SharedXMRDaemonRpcClient{client: client}
}
