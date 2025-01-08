package listener

import (
	"log"
	"testing"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/chekist32/goipay/internal/db"
	"github.com/stretchr/testify/assert"
)

func getBTCDaemonRpcClient() *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         "node.exaion.com/api/v1/8a7e0d5b-ae3c-4585-8a71-a99225932226/rpc",
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   false,
	}
	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func TestBTCGetLastBlockHeight(t *testing.T) {
	// Given
	d := NewSharedBTCDaemonRpcClient(getBTCDaemonRpcClient())

	// When
	height, err := d.GetLastBlockHeight()

	// Assert
	assert.NoError(t, err)
	assert.Greater(t, height, uint64(0))
}

func TestBTCGetBlockByHeight(t *testing.T) {
	// Given
	expectedBlockHash := "00000000000000000001aae6eacd54e7784b71010f62e0797c4dc2ef8a6963a7"
	d := NewSharedBTCDaemonRpcClient(getBTCDaemonRpcClient())

	// When
	block, err := d.GetBlockByHeight(877485)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedBlockHash, block.Header.BlockHash().String())
	assert.Equal(t, 1841, len(block.GetTxHashes()))
}

func TestBTCGeTransactionPool(t *testing.T) {
	// Given
	d := NewSharedBTCDaemonRpcClient(getBTCDaemonRpcClient())

	// When
	_, err := d.GetTransactionPool()

	// Assert
	assert.NoError(t, err)
}

func TestBTCGetTransactions(t *testing.T) {
	// Given
	d := NewSharedBTCDaemonRpcClient(getBTCDaemonRpcClient())
	expectedTxId := "179a771d8d84193b23e4a9853e4d158c81a8a41dcd86e9ff220d385a0acac6a7"

	// When
	txs, err := d.GetTransactions([]string{expectedTxId})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, expectedTxId, txs[0].GetTxId())
	assert.Greater(t, txs[0].GetConfirmations(), uint64(0))
	assert.False(t, txs[0].IsDoubleSpendSeen())
}

func TestBTCGetNetworkType(t *testing.T) {
	// Given
	d := NewSharedBTCDaemonRpcClient(getBTCDaemonRpcClient())

	// When
	net, err := d.GetNetworkType()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, MainnetBTC, net)
}

func TestBTCGetCoinType(t *testing.T) {
	// Given
	d := NewSharedBTCDaemonRpcClient(getBTCDaemonRpcClient())

	// When
	coin := d.GetCoinType()

	// Assert
	assert.Equal(t, db.CoinTypeBTC, coin)
}
