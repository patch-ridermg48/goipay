package listener

import (
	"log"
	"testing"

	"github.com/chekist32/goipay/internal/db"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

func getETHDaemonRpcClient() *ethclient.Client {
	client, err := ethclient.Dial("https://ethereum.publicnode.com")
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func TestETHGetLastBlockHeight(t *testing.T) {
	// Given
	d := NewSharedETHDaemonRpcClient(getETHDaemonRpcClient())

	// When
	height, err := d.GetLastBlockHeight()

	// Assert
	assert.NoError(t, err)
	assert.Greater(t, height, uint64(0))
}

func TestETHGetBlockByHeight(t *testing.T) {
	// Given
	expectedBlockHash := "0x7a28161c888a10e26a33d251d933ff34ee039aa004c0f871aadf5166e3747419"
	d := NewSharedETHDaemonRpcClient(getETHDaemonRpcClient())

	// When
	block, err := d.GetBlockByHeight(21660612)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedBlockHash, block.block.Hash().String())
	assert.Equal(t, 545, len(block.GetTxHashes()))
}

func TestETHGeTransactionPool(t *testing.T) {
	// Given
	d := NewSharedETHDaemonRpcClient(getETHDaemonRpcClient())

	// When
	_, err := d.GetTransactionPool()

	// Assert
	assert.NoError(t, err)
}

func TestETHGetTransactions(t *testing.T) {
	// Given
	d := NewSharedETHDaemonRpcClient(getETHDaemonRpcClient())
	expectedTxId := "0x4caafe1347589f252d2dedd009a5750f8dcb48c86840360a4adc066e691edcbd"

	// When
	txs, err := d.GetTransactions([]string{expectedTxId})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, expectedTxId, txs[0].GetTxId())
	assert.Greater(t, txs[0].GetConfirmations(), uint64(0))
	assert.False(t, txs[0].IsDoubleSpendSeen())
}

func TestETHGetNetworkType(t *testing.T) {
	// Given
	d := NewSharedETHDaemonRpcClient(getETHDaemonRpcClient())

	// When
	net, err := d.GetNetworkType()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, MainnetETH, net)
}

func TestETHGetCoinType(t *testing.T) {
	// Given
	d := NewSharedETHDaemonRpcClient(getETHDaemonRpcClient())

	// When
	coin := d.GetCoinType()

	// Assert
	assert.Equal(t, db.CoinTypeETH, coin)
}
