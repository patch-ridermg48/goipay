package processor

import (
	"context"
	"math/big"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/dto"
	"github.com/chekist32/goipay/internal/listener"
	"github.com/chekist32/goipay/internal/util"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

const (
	transferMethodSignature string = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
)

var (
	tokenData map[db.CoinType]struct {
		contractAddress string
		decimals        uint64
	} = map[db.CoinType]struct {
		contractAddress string
		decimals        uint64
	}{
		db.CoinTypeUSDTERC20:  {contractAddress: "0xdAC17F958D2ee523a2206206994597C13D831ec7", decimals: 10e5},
		db.CoinTypeUSDCERC20:  {contractAddress: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48", decimals: 10e5},
		db.CoinTypeDAIERC20:   {contractAddress: "0x6B175474E89094C44Da98b954EedeAC495271d0F", decimals: 10e17},
		db.CoinTypeWBTCERC20:  {contractAddress: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599", decimals: 10e7},
		db.CoinTypeUNIERC20:   {contractAddress: "0x1f9840a85d5aF5bf1D1762F925BDADdC4201F984", decimals: 10e17},
		db.CoinTypeLINKERC20:  {contractAddress: "0x514910771AF9Ca656af840dff83E8264EcF986CA", decimals: 10e17},
		db.CoinTypeAAVEERC20:  {contractAddress: "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9", decimals: 10e17},
		db.CoinTypeCRVERC20:   {contractAddress: "0xD533a949740bb3306d119CC777fa900bA034cd52", decimals: 10e17},
		db.CoinTypeMATICERC20: {contractAddress: "0x7D1AfA7B718fb893dB30A3aBc0Cfc608AaCfeBB0", decimals: 10e17},
		db.CoinTypeSHIBERC20:  {contractAddress: "0x95aD61b0a150d79219dCF64E1E6Cc01f0B64C4cE", decimals: 10e17},
		db.CoinTypeBNBERC20:   {contractAddress: "0xB8c77482e45F1F44dE1745F52C74426C631bDD52", decimals: 10e17},
		db.CoinTypeATOMERC20:  {contractAddress: "0x8D983cb9388EaC77af0474fA441C4815500Cb7BB", decimals: 10e5},
		db.CoinTypeARBERC20:   {contractAddress: "0xB50721BCf8d664c30412Cfbc6cf7a15145234ad1", decimals: 10e17},
	}
)

type ethProcessor struct {
	baseCryptoProcessor[listener.ETHTx, listener.ETHBlock]
}

func verifyETHTxHandler(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[listener.ETHTx]) (float64, error) {
	var amount float64 = 0

	if data.tx.IsDoubleSpendSeen() {
		return amount, nil
	}

	logsCount := len(data.tx.Logs)

	if logsCount > 0 {
		tokenData, ok := tokenData[data.invoice.Coin]
		if !ok {
			return amount, nil
		}

		for i := 0; i < logsCount; i++ {
			log := data.tx.Logs[i]
			if len(log.Topics) < 3 ||
				log.Topics[0].Hex() != transferMethodSignature ||
				log.Address.Hex() != tokenData.contractAddress {
				continue
			}

			if common.BytesToAddress(log.Topics[2].Bytes()).Hex() == data.invoice.CryptoAddress {
				am, _ := new(big.Float).Quo(
					new(big.Float).SetInt(new(big.Int).SetBytes(log.Data)),
					new(big.Float).SetInt(new(big.Int).SetUint64(tokenData.decimals)),
				).Float64()
				amount += am
			}
		}
	} else if toAddr := data.tx.Tx.To(); toAddr != nil && data.invoice.CryptoAddress == toAddr.Hex() {
		amount += float64(data.tx.Tx.Value().Uint64()) / 1e18
	}

	return amount, nil
}

func generateNextETHAddressHandler(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
	var addr db.CryptoAddress

	cd, err := q.FindCryptoDataByUserId(ctx, data.userId)
	if err != nil {
		return addr, err
	}

	indices, err := q.FindIndicesAndLockETHCryptoDataById(ctx, cd.EthID)
	if err != nil {
		return addr, err
	}

	mPubStr, err := q.FindKeysAndLockETHCryptoDataById(ctx, cd.EthID)
	if err != nil {
		return addr, err
	}

	mPub, err := hdkeychain.NewKeyFromString(mPubStr)
	if err != nil {
		return addr, err
	}

	indices.LastMinorIndex++
	if indices.LastMinorIndex <= 0 {
		indices.LastMinorIndex = 0
		indices.LastMajorIndex++
	}

	majMPub, err := mPub.Derive(uint32(indices.LastMajorIndex))
	if err != nil {
		return addr, err
	}
	minMPub, err := majMPub.Derive(uint32(indices.LastMinorIndex))
	if err != nil {
		return addr, err
	}

	pubKey, err := minMPub.ECPubKey()
	if err != nil {
		return addr, err
	}

	addr, err = q.CreateCryptoAddress(ctx, db.CreateCryptoAddressParams{Address: crypto.PubkeyToAddress(*pubKey.ToECDSA()).Hex(), Coin: db.CoinTypeETH, IsOccupied: true, UserID: data.userId})
	if err != nil {
		return addr, err
	}

	if _, err := q.UpdateIndicesETHCryptoDataById(ctx, db.UpdateIndicesETHCryptoDataByIdParams{ID: cd.EthID, LastMajorIndex: indices.LastMajorIndex, LastMinorIndex: indices.LastMinorIndex}); err != nil {
		return addr, err
	}

	return addr, nil
}

func newEthProcessor(log *zerolog.Logger, dbConnPool *pgxpool.Pool, invoiceCn chan<- db.Invoice, c *dto.DaemonsConfig) (*ethProcessor, error) {
	client, err := ethclient.Dial(c.Eth.Url)
	if err != nil {
		return nil, err
	}

	base, err := newBaseCryptoProcessor(
		log,
		dbConnPool,
		invoiceCn,
		listener.NewSharedETHDaemonRpcClient(client),
		verifyETHTxHandler,
		generateNextETHAddressHandler,
		util.GetMapKeys(tokenData),
	)
	if err != nil {
		return nil, err
	}

	return &ethProcessor{baseCryptoProcessor: *base}, nil
}
