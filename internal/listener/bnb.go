package listener

import (
	"context"
	"math"
	"unsafe"

	"github.com/chekist32/goipay/internal/db"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
)

type BNBBlock ETHBlock

func (b BNBBlock) GetTxHashes() []string {
	return ETHBlock(b).GetTxHashes()
}

type BNBTx ETHTx

func (t BNBTx) GetTxId() string {
	return ETHTx(t).GetTxId()
}
func (t BNBTx) GetConfirmations() uint64 {
	return ETHTx(t).GetConfirmations()
}
func (t BNBTx) IsDoubleSpendSeen() bool {
	return ETHTx(t).IsDoubleSpendSeen()
}

type SharedBNBDaemonRpcClient struct {
	SharedETHDaemonRpcClient
}

func (c *SharedBNBDaemonRpcClient) GetBlockByHeight(height uint64) (BNBBlock, error) {
	res, err := c.SharedETHDaemonRpcClient.GetBlockByHeight(height)
	return BNBBlock(res), err
}
func (c *SharedBNBDaemonRpcClient) GetTransactions(txHashes []string) ([]BNBTx, error) {
	res, err := c.SharedETHDaemonRpcClient.GetTransactions(txHashes)
	return *(*[]BNBTx)(unsafe.Pointer(&res)), err
}
func (c *SharedBNBDaemonRpcClient) GetNetworkType() (NetworkType, error) {
	res, err := c.SharedETHDaemonRpcClient.client.NetworkID(context.Background())
	if err != nil {
		return math.MaxUint8, err
	}

	switch res.Uint64() {
	case 56:
		return MainnetBNB, nil
	case 97:
		return TestnetBNB, nil
	case 714:
		return PrivateBNB, nil
	default:
		return math.MaxUint8, err
	}
}
func (c *SharedBNBDaemonRpcClient) GetCoinType() db.CoinType {
	return db.CoinTypeBNB
}

func NewSharedBNBDaemonRpcClient(client *ethclient.Client) *SharedBNBDaemonRpcClient {
	return &SharedBNBDaemonRpcClient{
		SharedETHDaemonRpcClient: *NewSharedETHDaemonRpcClient(client),
	}
}

type BNBDaemonRpcClientExecutor struct {
	BaseDaemonRpcClientExecutor[BNBTx, BNBBlock]
}

func NewBNBDaemonRpcClientExecutor(log *zerolog.Logger, client *ethclient.Client) *BNBDaemonRpcClientExecutor {
	return &BNBDaemonRpcClientExecutor{
		BaseDaemonRpcClientExecutor: *NewBaseDaemonRpcClientExecutor(log, NewSharedBNBDaemonRpcClient(client)),
	}
}
