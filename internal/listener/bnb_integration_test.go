package listener

import (
	"log"
	"testing"

	"github.com/chekist32/goipay/internal/db"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

func getBNBDaemonRpcClient() *ethclient.Client {
	client, err := ethclient.Dial("https://bsc-dataseed.binance.org")
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func TestBNBGetLastBlockHeight(t *testing.T) {
	// Given
	d := NewSharedBNBDaemonRpcClient(getBNBDaemonRpcClient())

	// When
	height, err := d.GetLastBlockHeight()

	// Assert
	assert.NoError(t, err)
	assert.Greater(t, height, uint64(0))
}

func TestBNBGetBlockByHeight(t *testing.T) {
	// Given
	expectedBlockHash := "0x32975b525af41325e99bc3614f4ae991943df90e6c93f5a02cc25fef11ff08a2"
	d := NewSharedBNBDaemonRpcClient(getBNBDaemonRpcClient())

	// When
	block, err := d.GetBlockByHeight(46139291)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedBlockHash, block.block.Hash().String())
	assert.Equal(t, 244, len(block.GetTxHashes()))
}

func TestBNBGeTransactionPool(t *testing.T) {
	// Given
	d := NewSharedBNBDaemonRpcClient(getBNBDaemonRpcClient())

	// When
	_, err := d.GetTransactionPool()

	// Assert
	assert.NoError(t, err)
}

func TestBNBGetTransactions(t *testing.T) {
	// Given
	d := NewSharedBNBDaemonRpcClient(getBNBDaemonRpcClient())
	expectedTxId := "0xcf25f3e87a652dd45b04414149de2671436f6bcafbe147385b549694461f446d"

	// When
	txs, err := d.GetTransactions([]string{expectedTxId})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, expectedTxId, txs[0].GetTxId())
	assert.Greater(t, txs[0].GetConfirmations(), uint64(0))
	assert.False(t, txs[0].IsDoubleSpendSeen())
}

func TestBNBGetNetworkType(t *testing.T) {
	// Given
	d := NewSharedBNBDaemonRpcClient(getBNBDaemonRpcClient())

	// When
	net, err := d.GetNetworkType()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, MainnetBNB, net)
}

func TestBNBGetCoinType(t *testing.T) {
	// Given
	d := NewSharedBNBDaemonRpcClient(getBNBDaemonRpcClient())

	// When
	coin := d.GetCoinType()

	// Assert
	assert.Equal(t, db.CoinTypeBNB, coin)
}
