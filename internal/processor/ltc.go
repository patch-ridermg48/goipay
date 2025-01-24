package processor

import (
	"context"
	"net/url"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/dto"
	"github.com/chekist32/goipay/internal/listener"
	"github.com/chekist32/goipay/internal/util"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/ltcutil"
	"github.com/ltcsuite/ltcd/ltcutil/hdkeychain"
	ltcrpc "github.com/ltcsuite/ltcd/rpcclient"
	"github.com/rs/zerolog"
)

type ltcProcessor struct {
	baseCryptoProcessor[listener.LTCTx, listener.LTCBlock]
}

func verifyLTCTxHandler(ctx context.Context, q *db.Queries, data *verifyTxHandlerData[listener.LTCTx]) (float64, error) {
	var amount float64 = 0
	for i := 0; i < len(data.tx.Vout); i++ {
		txOut := &data.tx.Vout[i]

		if txOut.ScriptPubKey.Address == data.invoice.CryptoAddress ||
			(len(txOut.ScriptPubKey.Addresses) == 1 && txOut.ScriptPubKey.Addresses[0] == data.invoice.CryptoAddress) {
			amount += data.tx.Vout[i].Value
		}
	}

	return amount, nil
}

func generateNextLTCAddressHandler(ctx context.Context, q *db.Queries, data *generateNextAddressHandlerData) (db.CryptoAddress, error) {
	var addr db.CryptoAddress

	net, err := func() (*chaincfg.Params, error) {
		switch data.network {
		case listener.MainnetLTC:
			return &chaincfg.MainNetParams, nil
		case listener.TestnetLTC:
			return &chaincfg.TestNet4Params, nil
		case listener.SignetLTC:
			return &chaincfg.SigNetParams, nil
		case listener.RegtestLTC:
			return &chaincfg.RegressionNetParams, nil
		default:
			return nil, util.InvalidNetworkTypeErr
		}
	}()
	if err != nil {
		return addr, err
	}

	cd, err := q.FindCryptoDataByUserId(ctx, data.userId)
	if err != nil {
		return addr, err
	}

	indices, err := q.FindIndicesAndLockLTCCryptoDataById(ctx, cd.LtcID)
	if err != nil {
		return addr, err
	}

	mPubStr, err := q.FindKeysAndLockLTCCryptoDataById(ctx, cd.LtcID)
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

	pubKeyHash := ltcutil.Hash160(pubKey.SerializeCompressed())
	newAddr, err := ltcutil.NewAddressWitnessPubKeyHash(pubKeyHash, net)
	if err != nil {
		return addr, err
	}

	addr, err = q.CreateCryptoAddress(ctx, db.CreateCryptoAddressParams{Address: newAddr.EncodeAddress(), Coin: db.CoinTypeLTC, IsOccupied: true, UserID: data.userId})
	if err != nil {
		return addr, err
	}

	if _, err := q.UpdateIndicesLTCCryptoDataById(ctx, db.UpdateIndicesLTCCryptoDataByIdParams{ID: cd.LtcID, LastMajorIndex: indices.LastMajorIndex, LastMinorIndex: indices.LastMinorIndex}); err != nil {
		return addr, err
	}

	return addr, nil
}

func newLtcProcessor(log *zerolog.Logger, dbConnPool *pgxpool.Pool, invoiceCn chan<- db.Invoice, c *dto.DaemonsConfig) (*ltcProcessor, error) {
	u, err := url.Parse(c.Ltc.Url)
	if err != nil {
		return nil, err
	}

	conf := &rpcclient.ConnConfig{
		Host:         u.Host + u.RequestURI(),
		User:         c.Ltc.User,
		Pass:         c.Ltc.Pass,
		DisableTLS:   u.Scheme != "https",
		HTTPPostMode: true,
	}
	client, err := rpcclient.New(conf, nil)
	if err != nil {
		return nil, err
	}
	ltcClient, err := ltcrpc.New(&ltcrpc.ConnConfig{
		Host:         conf.Host,
		User:         conf.User,
		Pass:         conf.Pass,
		DisableTLS:   conf.DisableTLS,
		HTTPPostMode: conf.HTTPPostMode,
	}, nil)
	if err != nil {
		return nil, err
	}

	base, err := newBaseCryptoProcessor(
		log,
		dbConnPool,
		invoiceCn,
		listener.NewSharedLTCDaemonRpcClient(client, ltcClient),
		verifyLTCTxHandler,
		generateNextLTCAddressHandler,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &ltcProcessor{baseCryptoProcessor: *base}, nil
}
