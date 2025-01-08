package listener

import (
	"log"
	"net/url"
	"testing"

	"github.com/chekist32/go-monero/daemon"
	"github.com/chekist32/goipay/internal/db"
	"github.com/stretchr/testify/assert"
)

func getXMRDaemonRpcClient() daemon.IDaemonRpcClient {
	uu, err := url.Parse("https://node.sethforprivacy.com")
	if err != nil {
		log.Fatal(err)
	}

	return daemon.NewDaemonRpcClient(daemon.NewRpcConnection(uu, "", ""))
}

func TestXMRGetLastBlockHeight(t *testing.T) {
	// Given
	d := NewSharedXMRDaemonRpcClient(getXMRDaemonRpcClient())

	// When
	height, err := d.GetLastBlockHeight()

	// Assert
	assert.NoError(t, err)
	assert.Greater(t, height, uint64(0))
}

func TestXMRGetBlockByHeight(t *testing.T) {
	// Given
	expectedBlockHash := "516b11e7c080be7439641fa7c38fc3ad0a9a4e555b956169f2045f9f8df408ef"
	d := NewSharedXMRDaemonRpcClient(getXMRDaemonRpcClient())

	// When
	block, err := d.GetBlockByHeight(3314292)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedBlockHash, block.BlockHeader.Hash)
	assert.Equal(t, 134, len(block.GetTxHashes()))
}

func TestXMRGeTransactionPool(t *testing.T) {
	// Given
	d := NewSharedXMRDaemonRpcClient(getXMRDaemonRpcClient())

	// When
	_, err := d.GetTransactionPool()

	// Assert
	assert.NoError(t, err)
}

func TestXMRGetTransactions(t *testing.T) {
	// Given
	d := NewSharedXMRDaemonRpcClient(getXMRDaemonRpcClient())
	expectedTxId := "8099a654574de5b130409419d8e85dc381db1d684ee6de620eb623a990f27445"

	// When
	txs, err := d.GetTransactions([]string{expectedTxId})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, expectedTxId, txs[0].GetTxId())
	assert.Greater(t, txs[0].GetConfirmations(), uint64(0))
	assert.False(t, txs[0].IsDoubleSpendSeen())
}

func TestXMRGetNetworkType(t *testing.T) {
	// Given
	d := NewSharedXMRDaemonRpcClient(getXMRDaemonRpcClient())

	// When
	net, err := d.GetNetworkType()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, MainnetXMR, net)
}

func TestXMRGetCoinType(t *testing.T) {
	// Given
	d := NewSharedXMRDaemonRpcClient(getXMRDaemonRpcClient())

	// When
	coin := d.GetCoinType()

	// Assert
	assert.Equal(t, db.CoinTypeXMR, coin)
}
