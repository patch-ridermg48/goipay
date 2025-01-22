package v1

import (
	"context"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/chekist32/go-monero/utils"
	"github.com/chekist32/goipay/internal/db"
	pb_v1 "github.com/chekist32/goipay/internal/pb/v1"
	"github.com/chekist32/goipay/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	ltchdkeychain "github.com/ltcsuite/ltcd/ltcutil/hdkeychain"
	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserGrpc struct {
	dbConnPool *pgxpool.Pool
	log        *zerolog.Logger
	pb_v1.UnimplementedUserServiceServer
}

func (u *UserGrpc) createUser(ctx context.Context, q *db.Queries, in *pb_v1.RegisterUserRequest) (*pgtype.UUID, error) {
	// Without userId in the request
	if in.UserId == nil {
		userId, err := q.CreateUser(ctx)
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "CreateUser").Msg(util.DefaultFailedSqlQueryMsg)
			return nil, status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}

		return &userId, err
	}

	// With userId in the request
	userIdReq, err := util.StringToPgUUID(*in.UserId)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg(util.InvalidUserIdInvalidUUIDMsg)
		return nil, status.Error(codes.InvalidArgument, util.InvalidUserIdInvalidUUIDMsg)
	}

	if err := checkIfUserExistsUUID(ctx, u.log, q, *userIdReq); err == nil {
		return nil, status.Error(codes.InvalidArgument, util.InvalidUserIdUserExistsMsg)
	}

	userId, err := q.CreateUserWithId(ctx, *userIdReq)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "CreateUserWithId").Msg(util.DefaultFailedSqlQueryMsg)
		return nil, status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	return &userId, err
}

func (u *UserGrpc) RegisterUser(ctx context.Context, in *pb_v1.RegisterUserRequest) (*pb_v1.RegisterUserResponse, error) {
	q, tx, err := util.InitDbQueriesWithTx(ctx, u.dbConnPool)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg(util.DefaultFailedSqlTxInitMsg)
		return nil, status.Error(codes.Internal, util.DefaultFailedSqlTxInitMsg)
	}
	defer tx.Rollback(ctx)

	userId, err := u.createUser(ctx, q, in)
	if err != nil {
		return nil, err
	}

	_, err = q.CreateCryptoData(ctx, db.CreateCryptoDataParams{UserID: *userId})
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "CreateUser").Msg(util.DefaultFailedSqlQueryMsg)
		return nil, status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	tx.Commit(ctx)

	return &pb_v1.RegisterUserResponse{UserId: util.PgUUIDToString(*userId)}, nil
}

func (u *UserGrpc) handleXmrCryptoDataUpdate(ctx context.Context, q *db.Queries, in *pb_v1.XmrKeysUpdateRequest, cryptData *db.CryptoDatum) error {
	_, err := utils.NewPrivateKey(in.PrivViewKey)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("An error occurred while creating the XMR private view key.")
		return status.Error(codes.InvalidArgument, "Invalid XMR private view key.")
	}
	_, err = utils.NewPublicKey(in.PubSpendKey)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("An error occurred while creating the XMR public spend key.")
		return status.Error(codes.InvalidArgument, "Invalid XMR public spend key.")
	}

	_, err = q.DeleteAllCryptoAddressByUserIdAndCoin(ctx, db.DeleteAllCryptoAddressByUserIdAndCoinParams{Coin: db.CoinTypeXMR, UserID: cryptData.UserID})
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "DeleteAllCryptoAddressByUserIdAndCoin").Msg(util.DefaultFailedSqlQueryMsg)
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	if !cryptData.XmrID.Valid {
		xmrData, err := q.CreateXMRCryptoData(ctx, db.CreateXMRCryptoDataParams{PrivViewKey: in.PrivViewKey, PubSpendKey: in.PubSpendKey})
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "CreateXMRCryptoData").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		_, err = q.SetXMRCryptoDataByUserId(ctx, db.SetXMRCryptoDataByUserIdParams{UserID: cryptData.UserID, XmrID: xmrData.ID})
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "SetXMRCryptoDataByUserId").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		return nil
	}
	_, err = q.UpdateKeysXMRCryptoDataById(ctx, db.UpdateKeysXMRCryptoDataByIdParams{ID: cryptData.XmrID, PrivViewKey: in.PrivViewKey, PubSpendKey: in.PubSpendKey})
	if err != nil {
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	return nil
}

func (u *UserGrpc) handleBtcCryptoDataUpdate(ctx context.Context, q *db.Queries, in *pb_v1.BtcKeysUpdateRequest, cryptData *db.CryptoDatum) error {
	_, err := hdkeychain.NewKeyFromString(in.MasterPubKey)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("An error occurred while creating the BTC master public key.")
		return status.Error(codes.InvalidArgument, "Invalid BTC master public key.")
	}

	_, err = q.DeleteAllCryptoAddressByUserIdAndCoin(ctx, db.DeleteAllCryptoAddressByUserIdAndCoinParams{Coin: db.CoinTypeBTC, UserID: cryptData.UserID})
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "DeleteAllCryptoAddressByUserIdAndCoin").Msg(util.DefaultFailedSqlQueryMsg)
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	if !cryptData.BtcID.Valid {
		btcData, err := q.CreateBTCCryptoData(ctx, in.MasterPubKey)
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "CreateBTCCryptoData").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		_, err = q.SetBTCCryptoDataByUserId(ctx, db.SetBTCCryptoDataByUserIdParams{UserID: cryptData.UserID, BtcID: btcData.ID})
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "SetBTCCryptoDataByUserId").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		return nil
	}
	_, err = q.UpdateKeysBTCCryptoDataById(ctx, db.UpdateKeysBTCCryptoDataByIdParams{ID: cryptData.BtcID, MasterPubKey: in.MasterPubKey})
	if err != nil {
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	return nil
}

func (u *UserGrpc) handleLtcCryptoDataUpdate(ctx context.Context, q *db.Queries, in *pb_v1.LtcKeysUpdateRequest, cryptData *db.CryptoDatum) error {
	_, err := ltchdkeychain.NewKeyFromString(in.MasterPubKey)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("An error occurred while creating the LTC master public key.")
		return status.Error(codes.InvalidArgument, "Invalid LTC master public key.")
	}

	_, err = q.DeleteAllCryptoAddressByUserIdAndCoin(ctx, db.DeleteAllCryptoAddressByUserIdAndCoinParams{Coin: db.CoinTypeLTC, UserID: cryptData.UserID})
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "DeleteAllCryptoAddressByUserIdAndCoin").Msg(util.DefaultFailedSqlQueryMsg)
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	if !cryptData.LtcID.Valid {
		ltcData, err := q.CreateLTCCryptoData(ctx, in.MasterPubKey)
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "CreateLTCCryptoData").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		_, err = q.SetLTCCryptoDataByUserId(ctx, db.SetLTCCryptoDataByUserIdParams{UserID: cryptData.UserID, LtcID: ltcData.ID})
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "SetLTCCryptoDataByUserId").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		return nil
	}
	_, err = q.UpdateKeysLTCCryptoDataById(ctx, db.UpdateKeysLTCCryptoDataByIdParams{ID: cryptData.LtcID, MasterPubKey: in.MasterPubKey})
	if err != nil {
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	return nil
}

func (u *UserGrpc) handleEthCryptoDataUpdate(ctx context.Context, q *db.Queries, in *pb_v1.EthKeysUpdateRequest, cryptData *db.CryptoDatum) error {
	_, err := hdkeychain.NewKeyFromString(in.MasterPubKey)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("An error occurred while creating the ETH master public key.")
		return status.Error(codes.InvalidArgument, "Invalid ETH master public key.")
	}

	_, err = q.DeleteAllCryptoAddressByUserIdAndCoin(ctx, db.DeleteAllCryptoAddressByUserIdAndCoinParams{Coin: db.CoinTypeETH, UserID: cryptData.UserID})
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "DeleteAllCryptoAddressByUserIdAndCoin").Msg(util.DefaultFailedSqlQueryMsg)
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	if !cryptData.EthID.Valid {
		ethData, err := q.CreateETHCryptoData(ctx, in.MasterPubKey)
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "CreateETHCryptoData").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		_, err = q.SetETHCryptoDataByUserId(ctx, db.SetETHCryptoDataByUserIdParams{UserID: cryptData.UserID, EthID: ethData.ID})
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "SetETHCryptoDataByUserId").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		return nil
	}
	_, err = q.UpdateKeysETHCryptoDataById(ctx, db.UpdateKeysETHCryptoDataByIdParams{ID: cryptData.EthID, MasterPubKey: in.MasterPubKey})
	if err != nil {
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	return nil
}

func (u *UserGrpc) UpdateCryptoKeys(ctx context.Context, in *pb_v1.UpdateCryptoKeysRequest) (*pb_v1.UpdateCryptoKeysResponse, error) {
	q, tx, err := util.InitDbQueriesWithTx(ctx, u.dbConnPool)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg(util.DefaultFailedSqlTxInitMsg)
		return nil, status.Error(codes.Internal, util.DefaultFailedSqlTxInitMsg)
	}
	defer tx.Rollback(ctx)

	userId, err := util.StringToPgUUID(in.UserId)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg(util.FailedStringToPgUUIDMappingMsg)
		return nil, status.Error(codes.InvalidArgument, util.InvalidUserIdInvalidUUIDMsg)
	}

	if err := checkIfUserExistsUUID(ctx, u.log, q, *userId); err != nil {
		return nil, err
	}

	cryptData, err := q.FindCryptoDataByUserId(ctx, *userId)
	if err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "FindCryptoDataByUserId").Msg(util.DefaultFailedSqlQueryMsg)
		return nil, status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	if in.XmrReq != nil {
		if err := u.handleXmrCryptoDataUpdate(ctx, q, in.XmrReq, &cryptData); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("")
			return nil, err
		}
	}
	if in.BtcReq != nil {
		if err := u.handleBtcCryptoDataUpdate(ctx, q, in.BtcReq, &cryptData); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("")
			return nil, err
		}
	}
	if in.LtcReq != nil {
		if err := u.handleLtcCryptoDataUpdate(ctx, q, in.LtcReq, &cryptData); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("")
			return nil, err
		}
	}
	if in.EthReq != nil {
		if err := u.handleEthCryptoDataUpdate(ctx, q, in.EthReq, &cryptData); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("")
			return nil, err
		}
	}

	tx.Commit(ctx)

	return &pb_v1.UpdateCryptoKeysResponse{}, nil
}

func NewUserGrpc(dbConnPool *pgxpool.Pool, log *zerolog.Logger) *UserGrpc {
	return &UserGrpc{dbConnPool: dbConnPool, log: log}
}
