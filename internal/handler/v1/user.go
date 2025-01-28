package v1

import (
	"context"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/chekist32/go-monero/utils"
	"github.com/chekist32/goipay/internal/db"
	pb_v1 "github.com/chekist32/goipay/internal/pb/v1"
	"github.com/chekist32/goipay/internal/util"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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
	if _, err := utils.NewPrivateKey(in.PrivViewKey); err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("An error occurred while creating the XMR private view key.")
		return status.Error(codes.InvalidArgument, "Invalid XMR private view key.")
	}
	if _, err := utils.NewPublicKey(in.PubSpendKey); err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("An error occurred while creating the XMR public spend key.")
		return status.Error(codes.InvalidArgument, "Invalid XMR public spend key.")
	}

	if _, err := q.DeleteAllCryptoAddressByUserIdAndCoin(ctx, db.DeleteAllCryptoAddressByUserIdAndCoinParams{Coin: db.CoinTypeXMR, UserID: cryptData.UserID}); err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "DeleteAllCryptoAddressByUserIdAndCoin").Msg(util.DefaultFailedSqlQueryMsg)
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	if !cryptData.XmrID.Valid {
		xmrData, err := q.CreateXMRCryptoData(ctx, db.CreateXMRCryptoDataParams{PrivViewKey: in.PrivViewKey, PubSpendKey: in.PubSpendKey})
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "CreateXMRCryptoData").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}

		if _, err := q.SetXMRCryptoDataByUserId(ctx, db.SetXMRCryptoDataByUserIdParams{UserID: cryptData.UserID, XmrID: xmrData.ID}); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "SetXMRCryptoDataByUserId").Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		return nil
	}

	if _, err := q.UpdateKeysXMRCryptoDataById(ctx, db.UpdateKeysXMRCryptoDataByIdParams{ID: cryptData.XmrID, PrivViewKey: in.PrivViewKey, PubSpendKey: in.PubSpendKey}); err != nil {
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	return nil
}

func (u *UserGrpc) handleHDKeysCryptoDataUpdate(ctx context.Context, q *db.Queries, masterPubKey string, coin db.CoinType, cryptData *db.CryptoDatum) error {
	cryptoId, createCryptoCryptoData, setCryptoCryptoDataByUserId, updateKeysCryptoCryptoDataById, err := func() (
		pgtype.UUID,
		func(masterPubKey string) (pgtype.UUID, error),
		func(userId, cryptoId pgtype.UUID) error,
		func(cryptoId pgtype.UUID, masterPubKey string) error,
		error,
	) {
		switch coin {
		case db.CoinTypeLTC:
			return cryptData.LtcID,
				func(masterPubKey string) (pgtype.UUID, error) {
					data, err := q.CreateLTCCryptoData(ctx, masterPubKey)
					return data.ID, err
				},
				func(userId, cryptoId pgtype.UUID) error {
					_, err := q.SetLTCCryptoDataByUserId(ctx, db.SetLTCCryptoDataByUserIdParams{UserID: userId, LtcID: cryptoId})
					return err
				},
				func(cryptoId pgtype.UUID, masterPubKey string) error {
					_, err := q.UpdateKeysLTCCryptoDataById(ctx, db.UpdateKeysLTCCryptoDataByIdParams{ID: cryptoId, MasterPubKey: masterPubKey})
					return err
				},
				nil
		case db.CoinTypeBTC:
			return cryptData.BtcID,
				func(masterPubKey string) (pgtype.UUID, error) {
					data, err := q.CreateBTCCryptoData(ctx, masterPubKey)
					return data.ID, err
				},
				func(userId, cryptoId pgtype.UUID) error {
					_, err := q.SetBTCCryptoDataByUserId(ctx, db.SetBTCCryptoDataByUserIdParams{UserID: userId, BtcID: cryptoId})
					return err
				},
				func(cryptoId pgtype.UUID, masterPubKey string) error {
					_, err := q.UpdateKeysBTCCryptoDataById(ctx, db.UpdateKeysBTCCryptoDataByIdParams{ID: cryptoId, MasterPubKey: masterPubKey})
					return err
				},
				nil
		case db.CoinTypeETH:
			return cryptData.EthID,
				func(masterPubKey string) (pgtype.UUID, error) {
					data, err := q.CreateETHCryptoData(ctx, masterPubKey)
					return data.ID, err
				},
				func(userId, cryptoId pgtype.UUID) error {
					_, err := q.SetETHCryptoDataByUserId(ctx, db.SetETHCryptoDataByUserIdParams{UserID: userId, EthID: cryptoId})
					return err
				},
				func(cryptoId pgtype.UUID, masterPubKey string) error {
					_, err := q.UpdateKeysETHCryptoDataById(ctx, db.UpdateKeysETHCryptoDataByIdParams{ID: cryptoId, MasterPubKey: masterPubKey})
					return err
				},
				nil
		case db.CoinTypeBNB:
			return cryptData.BnbID,
				func(masterPubKey string) (pgtype.UUID, error) {
					data, err := q.CreateBNBCryptoData(ctx, masterPubKey)
					return data.ID, err
				},
				func(userId, cryptoId pgtype.UUID) error {
					_, err := q.SetBNBCryptoDataByUserId(ctx, db.SetBNBCryptoDataByUserIdParams{UserID: userId, BnbID: cryptoId})
					return err
				},
				func(cryptoId pgtype.UUID, masterPubKey string) error {
					_, err := q.UpdateKeysBNBCryptoDataById(ctx, db.UpdateKeysBNBCryptoDataByIdParams{ID: cryptoId, MasterPubKey: masterPubKey})
					return err
				},
				nil
		default:
			return pgtype.UUID{}, nil, nil, nil, errors.New("unsupported coin type")
		}
	}()
	if err != nil {
		return err
	}

	if _, err := hdkeychain.NewKeyFromString(masterPubKey); err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg(fmt.Sprintf("An error occurred while creating the %v master public key.", coin))
		return status.Error(codes.InvalidArgument, fmt.Sprintf("Invalid %v master public key.", coin))
	}

	if _, err = q.DeleteAllCryptoAddressByUserIdAndCoin(ctx, db.DeleteAllCryptoAddressByUserIdAndCoinParams{Coin: coin, UserID: cryptData.UserID}); err != nil {
		u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", "DeleteAllCryptoAddressByUserIdAndCoin").Msg(util.DefaultFailedSqlQueryMsg)
		return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
	}

	if !cryptoId.Valid {
		newCryptoId, err := createCryptoCryptoData(masterPubKey)
		if err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", fmt.Sprintf("Create%vCryptoData", coin)).Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		if err := setCryptoCryptoDataByUserId(cryptData.UserID, newCryptoId); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Str("queryName", fmt.Sprintf("Set%vCryptoDataByUserId", coin)).Msg(util.DefaultFailedSqlQueryMsg)
			return status.Error(codes.Internal, util.DefaultFailedSqlQueryMsg)
		}
		return nil
	}

	if err := updateKeysCryptoCryptoDataById(cryptoId, masterPubKey); err != nil {
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
		if err := u.handleHDKeysCryptoDataUpdate(ctx, q, in.BtcReq.MasterPubKey, db.CoinTypeBTC, &cryptData); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("")
			return nil, err
		}
	}
	if in.LtcReq != nil {
		if err := u.handleHDKeysCryptoDataUpdate(ctx, q, in.LtcReq.MasterPubKey, db.CoinTypeLTC, &cryptData); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("")
			return nil, err
		}
	}
	if in.EthReq != nil {
		if err := u.handleHDKeysCryptoDataUpdate(ctx, q, in.EthReq.MasterPubKey, db.CoinTypeETH, &cryptData); err != nil {
			u.log.Err(err).Str(util.RequestIdLogKey, util.GetRequestIdOrEmptyString(ctx)).Msg("")
			return nil, err
		}
	}
	if in.BnbReq != nil {
		if err := u.handleHDKeysCryptoDataUpdate(ctx, q, in.BnbReq.MasterPubKey, db.CoinTypeBNB, &cryptData); err != nil {
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
