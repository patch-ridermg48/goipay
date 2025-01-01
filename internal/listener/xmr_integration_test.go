package listener

import (
	"log"
	"net/url"
	"testing"

	"github.com/chekist32/go-monero/daemon"
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
	d := sharedXMRDaemonRpcClient{client: getXMRDaemonRpcClient()}
	height, err := d.getLastBlockHeight()

	assert.NoError(t, err)
	assert.Greater(t, height, uint64(0))
}

func TestXMRGetBlockByHeight(t *testing.T) {
	expectedBlockHash := "516b11e7c080be7439641fa7c38fc3ad0a9a4e555b956169f2045f9f8df408ef"

	d := sharedXMRDaemonRpcClient{client: getXMRDaemonRpcClient()}
	block, err := d.getBlockByHeight(3314292)

	assert.NoError(t, err)
	assert.Equal(t, expectedBlockHash, block.BlockHeader.Hash)
}

func TestXMRGeTransactionPool(t *testing.T) {
	d := sharedXMRDaemonRpcClient{client: getXMRDaemonRpcClient()}
	_, err := d.getTransactionPool()

	assert.NoError(t, err)
}
