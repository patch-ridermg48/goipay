package listener

import (
	"errors"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/wire"
	"github.com/chekist32/goipay/internal/db"
	"github.com/rs/zerolog"
)

type BTCBlock wire.MsgBlock

func (b BTCBlock) GetTxHashes() []string {
	txHashes := make([]string, 0, len(b.Transactions))
	for i := 0; i < len(b.Transactions); i++ {
		txHashes = append(txHashes, b.Transactions[i].TxHash().String())
	}

	return txHashes
}

type BTCTx btcjson.TxRawResult

func (t BTCTx) GetTxId() string {
	return t.Txid
}
func (t BTCTx) GetConfirmations() uint64 {
	return t.Confirmations
}
func (t BTCTx) IsDoubleSpendSeen() bool {
	return false
}

type SharedBTCDaemonRpcClient struct {
	client *rpcclient.Client
}

func (c *SharedBTCDaemonRpcClient) GetLastBlockHeight() (uint64, error) {
	height, err := c.client.GetBlockCount()
	if err != nil {
		return 0, err
	}

	return uint64(height), nil
}
func (c *SharedBTCDaemonRpcClient) GetBlockByHeight(height uint64) (BTCBlock, error) {
	hash, err := c.client.GetBlockHash(int64(height))
	if err != nil {
		return BTCBlock{}, err
	}
	block, err := c.client.GetBlock(hash)
	if err != nil {
		return BTCBlock{}, err
	}

	return BTCBlock(*block), nil
}
func (c *SharedBTCDaemonRpcClient) GetTransactionPool() ([]string, error) {
	hashes, err := c.client.GetRawMempool()
	if err != nil {
		return nil, err
	}

	txHashes := make([]string, 0, len(hashes))
	for i := 0; i < len(hashes); i++ {
		txHashes = append(txHashes, hashes[i].String())
	}

	return txHashes, nil
}
func (c *SharedBTCDaemonRpcClient) GetTransactions(txHashes []string) ([]BTCTx, error) {
	txHashesCount := len(txHashes)

	hashes := make([]*chainhash.Hash, 0, txHashesCount)
	for i := 0; i < txHashesCount; i++ {
		hash, err := chainhash.NewHashFromStr(txHashes[i])
		if err != nil {
			return nil, err
		}
		hashes = append(hashes, hash)
	}

	txsAsync := make([]rpcclient.FutureGetRawTransactionVerboseResult, 0, txHashesCount)
	for i := 0; i < txHashesCount; i++ {
		txsAsync = append(txsAsync, c.client.GetRawTransactionVerboseAsync(hashes[i]))
	}

	txs := make([]BTCTx, 0, txHashesCount)
	for i := 0; i < txHashesCount; i++ {
		tx, err := txsAsync[i].Receive()
		if err != nil {
			return nil, err
		}
		txs = append(txs, BTCTx(*tx))
	}

	return txs, nil
}
func (c *SharedBTCDaemonRpcClient) GetNetworkType() (NetworkType, error) {
	res, err := c.client.GetBlockChainInfo()
	if err != nil {
		return 255, err
	}

	switch res.Chain {
	case "main":
		return MainnetBTC, nil
	case "mainnet":
		return MainnetBTC, nil
	case "test":
		return TestnetBTC, nil
	case "testnet":
		return TestnetBTC, nil
	case "regtest":
		return RegtestBTC, nil
	case "signet":
		return SignetBTC, nil
	default:
		return 255, errors.New("invalid network type")
	}
}
func (c *SharedBTCDaemonRpcClient) GetCoinType() db.CoinType {
	return db.CoinTypeBTC
}

func NewSharedBTCDaemonRpcClient(client *rpcclient.Client) *SharedBTCDaemonRpcClient {
	return &SharedBTCDaemonRpcClient{client: client}
}

type BTCDaemonRpcClientExecutor struct {
	BaseDaemonRpcClientExecutor[BTCTx, BTCBlock]
}

func NewBTCDaemonRpcClientExecutor(log *zerolog.Logger, client *rpcclient.Client) *BTCDaemonRpcClientExecutor {
	return &BTCDaemonRpcClientExecutor{
		BaseDaemonRpcClientExecutor: *NewBaseDaemonRpcClientExecutor(log, &SharedBTCDaemonRpcClient{client: client}),
	}
}
