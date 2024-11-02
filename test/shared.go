package test

import (
	"github.com/chekist32/go-monero/daemon"
	"github.com/stretchr/testify/mock"
)

type MockXMRDaemonRpcClient struct {
	mock.Mock
}

func (m *MockXMRDaemonRpcClient) GetLastBlockHeader(includeHex bool) (*daemon.JsonRpcGenericResponse[daemon.GetBlockHeaderResult], error) {
	args := m.Called(includeHex)
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetBlockHeaderResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetBlockByHeight(includeHex bool, height uint64) (*daemon.JsonRpcGenericResponse[daemon.GetBlockResult], error) {
	args := m.Called(includeHex, height)
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetBlockResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetTransactionPool() (*daemon.GetTransactionPoolResponse, error) {
	args := m.Called()
	return args.Get(0).(*daemon.GetTransactionPoolResponse), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetBlockByHash(fillPowHash bool, hash string) (*daemon.JsonRpcGenericResponse[daemon.GetBlockResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetBlockResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetBlockCount() (*daemon.JsonRpcGenericResponse[daemon.GetBlockCountResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetBlockCountResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetBlockHeaderByHash(fillPowHash bool, hash string) (*daemon.JsonRpcGenericResponse[daemon.GetBlockHeaderResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetBlockHeaderResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetBlockHeaderByHeight(fillPowHash bool, height uint64) (*daemon.JsonRpcGenericResponse[daemon.GetBlockHeaderResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetBlockHeaderResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetBlockHeadersRange(fillPowHash bool, startHeight uint64, endHeight uint64) (*daemon.JsonRpcGenericResponse[daemon.GetBlockHeadersRangeResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetBlockHeadersRangeResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetBlockTemplate(wallet string, reverseSize uint64) (*daemon.JsonRpcGenericResponse[daemon.GetBlockTemplateResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetBlockTemplateResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetCurrentHeight() (*daemon.GetHeightResponse, error) {
	args := m.Called()
	return args.Get(0).(*daemon.GetHeightResponse), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetFeeEstimate() (*daemon.JsonRpcGenericResponse[daemon.GetFeeEstimateResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetFeeEstimateResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetInfo() (*daemon.JsonRpcGenericResponse[daemon.GetInfoResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetInfoResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetTransactions(txHashes []string, decodeAsJson bool, prune bool, split bool) (*daemon.GetTransactionsResponse, error) {
	args := m.Called()
	return args.Get(0).(*daemon.GetTransactionsResponse), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) GetVersion() (*daemon.JsonRpcGenericResponse[daemon.GetVersionResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.GetVersionResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) OnGetBlockHash(height uint64) (*daemon.JsonRpcGenericResponse[daemon.OnGetBlockHashResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.OnGetBlockHashResult]), args.Error(1)
}
func (m *MockXMRDaemonRpcClient) SetRpcConnection(connection *daemon.RpcConnection) {}
func (m *MockXMRDaemonRpcClient) SubmitBlock(blobData []string) (*daemon.JsonRpcGenericResponse[daemon.SubmitBlockResult], error) {
	args := m.Called()
	return args.Get(0).(*daemon.JsonRpcGenericResponse[daemon.SubmitBlockResult]), args.Error(1)
}
