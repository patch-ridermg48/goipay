package util

import (
	"context"
	"os"

	"github.com/chekist32/goipay/internal/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDbQueriesWithTx(ctx context.Context, dbConnPool *pgxpool.Pool) (*db.Queries, pgx.Tx, error) {
	tx, err := dbConnPool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}

	return db.New(tx), tx, nil
}

func GetOptionOrEnvValue(env string, opt string) string {
	if opt != "" {
		return opt
	}

	return os.Getenv(env)
}

type CustomMetadata struct {
	RequestId string
}

func GetRequestIdOrEmptyString(ctx context.Context) string {
	md, ok := ctx.Value(MetadataCtxKey).(CustomMetadata)
	if !ok {
		return ""
	}

	return md.RequestId
}

func SliceToSet[T comparable](s []T) map[T]bool {
	size := len(s)

	m := make(map[T]bool, size)
	for i := 0; i < size; i++ {
		m[s[i]] = true
	}

	return m
}

func GetMapKeys[K comparable, V any](m map[K]V) []K {
	size := len(m)

	s := make([]K, 0, size)
	for k := range m {
		s = append(s, k)
	}

	return s
}
