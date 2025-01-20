package listener

import (
	"context"
	"errors"
	"math/big"

	"github.com/chekist32/goipay/internal/db"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
)

type ETHBlock struct {
	block *types.Block
}

func (b ETHBlock) GetTxHashes() []string {
	txHashes := make([]string, 0, b.block.Transactions().Len())
	for _, v := range b.block.Transactions() {
		txHashes = append(txHashes, v.Hash().Hex())
	}

	return txHashes
}

type ETHTx struct {
	Tx            *types.Transaction
	Status        uint8
	Confirmations uint64
	Logs          []*types.Log
}

func (t ETHTx) GetTxId() string {
	return t.Tx.Hash().Hex()
}
func (t ETHTx) GetConfirmations() uint64 {
	return t.Confirmations
}
func (t ETHTx) IsDoubleSpendSeen() bool {
	return t.Status == 0
}

type SharedETHDaemonRpcClient struct {
	client *ethclient.Client
}

func (c *SharedETHDaemonRpcClient) GetLastBlockHeight() (uint64, error) {
	height, err := c.client.BlockNumber(context.Background())
	if err != nil {
		return 0, err
	}

	return height, nil
}
func (c *SharedETHDaemonRpcClient) GetBlockByHeight(height uint64) (ETHBlock, error) {
	block, err := c.client.BlockByNumber(context.Background(), big.NewInt(int64(height)))
	if err != nil {
		return ETHBlock{}, err
	}

	return ETHBlock{block: block}, nil
}
func (c *SharedETHDaemonRpcClient) GetTransactionPool() ([]string, error) {
	return []string{}, nil
}
func (c *SharedETHDaemonRpcClient) GetTransactions(txHashes []string) ([]ETHTx, error) {
	txHashesCount := len(txHashes)

	hashes := make([]common.Hash, 0, txHashesCount)
	for i := 0; i < txHashesCount; i++ {
		hashes = append(hashes, common.HexToHash(txHashes[i]))
	}

	lastBlockHeight, err := c.GetLastBlockHeight()
	if err != nil {
		return nil, err
	}

	txs := make([]ETHTx, 0, txHashesCount)
	for i := 0; i < txHashesCount; i++ {
		tx, _, err := c.client.TransactionByHash(context.Background(), hashes[i])
		if err != nil {
			return nil, err
		}
		txReceipt, err := c.client.TransactionReceipt(context.Background(), hashes[i])
		if err != nil {
			return nil, err
		}
		txs = append(txs, ETHTx{
			Tx:            tx,
			Status:        uint8(txReceipt.Status),
			Logs:          txReceipt.Logs,
			Confirmations: lastBlockHeight - txReceipt.BlockNumber.Uint64(),
		})
	}

	return txs, nil
}
func (c *SharedETHDaemonRpcClient) GetNetworkType() (NetworkType, error) {
	netVer, err := c.client.NetworkID(context.Background())
	if err != nil {
		return 255, err
	}

	switch netVer.Uint64() {
	case 1:
		return MainnetETH, nil
	case 5:
		return GoerliETH, nil
	case 11155111:
		return SepoliaETH, nil
	case 1337:
		return PrivateETH, nil
	default:
		return 255, errors.New("invalid network type")
	}
}
func (c *SharedETHDaemonRpcClient) GetCoinType() db.CoinType {
	return db.CoinTypeETH
}

func NewSharedETHDaemonRpcClient(client *ethclient.Client) *SharedETHDaemonRpcClient {
	return &SharedETHDaemonRpcClient{client: client}
}

type ETHDaemonRpcClientExecutor struct {
	BaseDaemonRpcClientExecutor[ETHTx, ETHBlock]
}

func NewETHDaemonRpcClientExecutor(log *zerolog.Logger, client *ethclient.Client) *ETHDaemonRpcClientExecutor {
	return &ETHDaemonRpcClientExecutor{
		BaseDaemonRpcClientExecutor: *NewBaseDaemonRpcClientExecutor(log, NewSharedETHDaemonRpcClient(client)),
	}
}
