package listener

import (
	"log"
	"testing"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/chekist32/goipay/internal/db"
	ltcrpc "github.com/ltcsuite/ltcd/rpcclient"
	"github.com/stretchr/testify/assert"
)

func getLTCDaemonRpcClient1() *rpcclient.Client {
	connCfg := &rpcclient.ConnConfig{
		Host:         "api.chainup.net/litecoin/mainnet/0b1abdf17ecc4b20b110ee73e17e7493",
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

func getLTCDaemonRpcClient2() *ltcrpc.Client {
	connCfg := &ltcrpc.ConnConfig{
		Host:         "api.chainup.net/litecoin/mainnet/0b1abdf17ecc4b20b110ee73e17e7493",
		User:         "user",
		Pass:         "pass",
		HTTPPostMode: true,
		DisableTLS:   false,
	}
	client, err := ltcrpc.New(connCfg, nil)
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func TestLTCGetLastBlockHeight(t *testing.T) {
	// Given
	d := NewSharedLTCDaemonRpcClient(getLTCDaemonRpcClient1(), getLTCDaemonRpcClient2())

	// When
	height, err := d.GetLastBlockHeight()

	// Assert
	assert.NoError(t, err)
	assert.Greater(t, height, uint64(0))
}

func TestLTCGetBlockByHeight(t *testing.T) {
	// Given
	expectedBlockHash := "cdbc2bf7d8ff2e90d5f720775b95be1833b2cf7c7b27ce7f385cb068bdef3113"
	d := NewSharedLTCDaemonRpcClient(getLTCDaemonRpcClient1(), getLTCDaemonRpcClient2())

	// When
	block, err := d.GetBlockByHeight(2829638)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, expectedBlockHash, block.Header.BlockHash().String())
	assert.Equal(t, 83, len(block.GetTxHashes()))
}

func TestLTCGeTransactionPool(t *testing.T) {
	// Given
	d := NewSharedLTCDaemonRpcClient(getLTCDaemonRpcClient1(), getLTCDaemonRpcClient2())

	// When
	_, err := d.GetTransactionPool()

	// Assert
	assert.NoError(t, err)
}

func TestLTCGetTransactions(t *testing.T) {
	// Given
	d := NewSharedLTCDaemonRpcClient(getLTCDaemonRpcClient1(), getLTCDaemonRpcClient2())
	expectedTxId := "e8ecc5e31df3cf6ae35f1949462a1dcd4690f54d80683ba1c5355b98df123eca"

	// When
	txs, err := d.GetTransactions([]string{expectedTxId})

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, expectedTxId, txs[0].GetTxId())
	assert.Greater(t, txs[0].GetConfirmations(), uint64(0))
	assert.False(t, txs[0].IsDoubleSpendSeen())
}

func TestLTCGetNetworkType(t *testing.T) {
	// Given
	d := NewSharedLTCDaemonRpcClient(getLTCDaemonRpcClient1(), getLTCDaemonRpcClient2())

	// When
	net, err := d.GetNetworkType()

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, MainnetLTC, net)
}

func TestLTCGetCoinType(t *testing.T) {
	// Given
	d := NewSharedLTCDaemonRpcClient(getLTCDaemonRpcClient1(), getLTCDaemonRpcClient2())

	// When
	coin := d.GetCoinType()

	// Assert
	assert.Equal(t, db.CoinTypeLTC, coin)
}
