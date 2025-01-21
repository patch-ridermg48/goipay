package util

import (
	"math"

	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/dto"
	pb_v1 "github.com/chekist32/goipay/internal/pb/v1"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func StringToPgUUID(uuidStr string) (*pgtype.UUID, error) {
	uuid := &pgtype.UUID{}
	if err := uuid.Scan(uuidStr); err != nil {
		return nil, err
	}

	return uuid, nil
}

func PgUUIDToString(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}

	str, _ := uuid.MarshalJSON()
	return string(str[1 : len(str)-1])
}

func PbCoinToDbCoin(coin pb_v1.CoinType) (db.CoinType, error) {
	switch coin {
	case pb_v1.CoinType_XMR:
		return db.CoinTypeXMR, nil
	case pb_v1.CoinType_BTC:
		return db.CoinTypeBTC, nil
	case pb_v1.CoinType_ETH:
		return db.CoinTypeETH, nil
	case pb_v1.CoinType_LTC:
		return db.CoinTypeLTC, nil
	case pb_v1.CoinType_TON:
		return db.CoinTypeTON, nil

	// ERC20
	case pb_v1.CoinType_USDT_ERC20:
		return db.CoinTypeUSDTERC20, nil
	case pb_v1.CoinType_USDC_ERC20:
		return db.CoinTypeUSDCERC20, nil
	case pb_v1.CoinType_DAI_ERC20:
		return db.CoinTypeDAIERC20, nil
	case pb_v1.CoinType_WBTC_ERC20:
		return db.CoinTypeWBTCERC20, nil
	case pb_v1.CoinType_UNI_ERC20:
		return db.CoinTypeUNIERC20, nil
	case pb_v1.CoinType_LINK_ERC20:
		return db.CoinTypeLINKERC20, nil
	case pb_v1.CoinType_CRV_ERC20:
		return db.CoinTypeCRVERC20, nil
	case pb_v1.CoinType_MATIC_ERC20:
		return db.CoinTypeMATICERC20, nil
	case pb_v1.CoinType_SHIB_ERC20:
		return db.CoinTypeSHIBERC20, nil
	case pb_v1.CoinType_BNB_ERC20:
		return db.CoinTypeBNBERC20, nil
	case pb_v1.CoinType_ATOM_ERC20:
		return db.CoinTypeATOMERC20, nil
	case pb_v1.CoinType_ARB_ERC20:
		return db.CoinTypeARBERC20, nil
	case pb_v1.CoinType_AAVE_ERC20:
		return db.CoinTypeAAVEERC20, nil
	}

	return "", invalidProtoBufCoinTypeErr
}

func DbCoinToPbCoin(coin db.CoinType) (pb_v1.CoinType, error) {
	switch coin {
	case db.CoinTypeXMR:
		return pb_v1.CoinType_XMR, nil
	case db.CoinTypeBTC:
		return pb_v1.CoinType_BTC, nil
	case db.CoinTypeETH:
		return pb_v1.CoinType_ETH, nil
	case db.CoinTypeLTC:
		return pb_v1.CoinType_LTC, nil
	case db.CoinTypeTON:
		return pb_v1.CoinType_TON, nil

	// ERC20
	case db.CoinTypeUSDTERC20:
		return pb_v1.CoinType_USDT_ERC20, nil
	case db.CoinTypeUSDCERC20:
		return pb_v1.CoinType_USDC_ERC20, nil
	case db.CoinTypeDAIERC20:
		return pb_v1.CoinType_DAI_ERC20, nil
	case db.CoinTypeWBTCERC20:
		return pb_v1.CoinType_WBTC_ERC20, nil
	case db.CoinTypeUNIERC20:
		return pb_v1.CoinType_UNI_ERC20, nil
	case db.CoinTypeLINKERC20:
		return pb_v1.CoinType_LINK_ERC20, nil
	case db.CoinTypeAAVEERC20:
		return pb_v1.CoinType_AAVE_ERC20, nil
	case db.CoinTypeCRVERC20:
		return pb_v1.CoinType_CRV_ERC20, nil
	case db.CoinTypeMATICERC20:
		return pb_v1.CoinType_MATIC_ERC20, nil
	case db.CoinTypeSHIBERC20:
		return pb_v1.CoinType_SHIB_ERC20, nil
	case db.CoinTypeBNBERC20:
		return pb_v1.CoinType_BNB_ERC20, nil
	case db.CoinTypeATOMERC20:
		return pb_v1.CoinType_ATOM_ERC20, nil
	case db.CoinTypeARBERC20:
		return pb_v1.CoinType_ARB_ERC20, nil
	}

	return math.MaxInt32, invalidDbCoinTypeErr
}

func DbInvoiceStatusToPbInvoiceStatus(status db.InvoiceStatusType) (pb_v1.InvoiceStatusType, error) {
	switch status {
	case db.InvoiceStatusTypePENDING:
		return pb_v1.InvoiceStatusType_PENDING, nil
	case db.InvoiceStatusTypePENDINGMEMPOOL:
		return pb_v1.InvoiceStatusType_PENDING_MEMPOOL, nil
	case db.InvoiceStatusTypeCONFIRMED:
		return pb_v1.InvoiceStatusType_CONFIRMED, nil
	case db.InvoiceStatusTypeEXPIRED:
		return pb_v1.InvoiceStatusType_EXPIRED, nil
	}

	return math.MaxInt32, invalidDbStatusTypeErr
}

func DbInvoiceToPbInvoice(invoice *db.Invoice) *pb_v1.Invoice {
	coin, _ := DbCoinToPbCoin(invoice.Coin)
	status, _ := DbInvoiceStatusToPbInvoiceStatus(invoice.Status)

	return &pb_v1.Invoice{
		Id:                    PgUUIDToString(invoice.ID),
		CryptoAddress:         invoice.CryptoAddress,
		Coin:                  coin,
		RequiredAmount:        invoice.RequiredAmount,
		ActualAmount:          invoice.ActualAmount.Float64,
		ConfirmationsRequired: uint32(invoice.ConfirmationsRequired),
		CreatedAt:             timestamppb.New(invoice.CreatedAt.Time),
		ConfirmedAt:           timestamppb.New(invoice.ConfirmedAt.Time),
		Status:                status,
		ExpiresAt:             timestamppb.New(invoice.ExpiresAt.Time),
		TxId:                  invoice.TxID.String,
		UserId:                PgUUIDToString(invoice.UserID),
	}
}

func PbNewInvoiceToProcessorNewInvoice(req *pb_v1.CreateInvoiceRequest) *dto.NewInvoiceRequest {
	coin, _ := PbCoinToDbCoin(req.Coin)

	return &dto.NewInvoiceRequest{
		UserId:        req.UserId,
		Coin:          coin,
		Amount:        req.Amount,
		Timeout:       req.Timeout,
		Confirmations: req.Confirmations,
	}
}
