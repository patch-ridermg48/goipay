package listener

import (
	"errors"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/chekist32/goipay/internal/db"
	ltcrpc "github.com/ltcsuite/ltcd/rpcclient"
	"github.com/rs/zerolog"
)

type LTCBlock BTCBlock

func (b LTCBlock) GetTxHashes() []string {
	return BTCBlock(b).GetTxHashes()
}

type LTCTx BTCTx

func (t LTCTx) GetConfirmations() uint64 {
	return BTCTx(t).GetConfirmations()
}
func (t LTCTx) GetTxId() string {
	return BTCTx(t).GetTxId()
}
func (t LTCTx) IsDoubleSpendSeen() bool {
	return BTCTx(t).IsDoubleSpendSeen()
}

type SharedLTCDaemonRpcClient struct {
	SharedBTCDaemonRpcClient
	ltcClient *ltcrpc.Client
}

func (c *SharedLTCDaemonRpcClient) GetNetworkType() (NetworkType, error) {
	res, err := c.ltcClient.GetBlockChainInfo()
	if err != nil {
		return 255, err
	}

	switch res.Chain {
	case "main":
		return MainnetLTC, nil
	case "mainnet":
		return MainnetLTC, nil
	case "test":
		return TestnetLTC, nil
	case "testnet":
		return TestnetLTC, nil
	case "regtest":
		return RegtestLTC, nil
	case "signet":
		return SignetLTC, nil
	default:
		return 255, errors.New("invalid network type")
	}
}
func (c *SharedLTCDaemonRpcClient) GetTransactions(txHashes []string) ([]LTCTx, error) {
	res, err := c.SharedBTCDaemonRpcClient.GetTransactions(txHashes)
	if err != nil {
		return nil, err
	}
	txCount := len(res)

	ltcTxs := make([]LTCTx, 0, txCount)
	for i := 0; i < txCount; i++ {
		ltcTxs = append(ltcTxs, LTCTx(res[i]))
	}

	return ltcTxs, nil
}
func (c *SharedLTCDaemonRpcClient) GetBlockByHeight(height uint64) (LTCBlock, error) {
	res, err := c.SharedBTCDaemonRpcClient.GetBlockByHeight(height)
	return LTCBlock(res), err
}
func (c *SharedLTCDaemonRpcClient) GetCoinType() db.CoinType {
	return db.CoinTypeLTC
}

func NewSharedLTCDaemonRpcClient(client *rpcclient.Client, ltcClient *ltcrpc.Client) *SharedLTCDaemonRpcClient {
	return &SharedLTCDaemonRpcClient{SharedBTCDaemonRpcClient: *NewSharedBTCDaemonRpcClient(client), ltcClient: ltcClient}
}

type LTCDaemonRpcClientExecutor struct {
	BaseDaemonRpcClientExecutor[LTCTx, LTCBlock]
}

func NewLTCDaemonRpcClientExecutor(log *zerolog.Logger, client *rpcclient.Client, ltcClient *ltcrpc.Client) *LTCDaemonRpcClientExecutor {
	return &LTCDaemonRpcClientExecutor{
		BaseDaemonRpcClientExecutor: *NewBaseDaemonRpcClientExecutor(log, NewSharedLTCDaemonRpcClient(client, ltcClient)),
	}
}
