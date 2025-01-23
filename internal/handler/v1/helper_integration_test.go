package v1

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/chekist32/goipay/internal/db"
	"github.com/chekist32/goipay/internal/util"
	"github.com/chekist32/goipay/test"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCheckIfUserExists(t *testing.T) {
	ctx := context.Background()

	dbConn, _, close := test.SpinUpPostgresContainerAndGetPgxpool(fmt.Sprintf("%v/../../../sql/migrations", os.Getenv("PWD")))
	defer close(ctx)

	q := db.New(dbConn)
	expectedUserId, err := q.CreateUser(ctx)
	if err != nil {
		log.Fatal(err)
	}

	t.Run("checkIfUserExistsUUID", func(t *testing.T) {
		t.Run("Should Return No Error", func(t *testing.T) {
			assert.NoError(t, checkIfUserExistsUUID(ctx, &zerolog.Logger{}, q, expectedUserId))
		})
		t.Run("Should Return Error", func(t *testing.T) {
			var wrongUUID pgtype.UUID
			if err := wrongUUID.Scan(uuid.NewString()); err != nil {
				log.Fatal(err)
			}

			assert.EqualError(t, checkIfUserExistsUUID(ctx, &zerolog.Logger{}, q, wrongUUID), status.Error(codes.InvalidArgument, util.InvalidUserIdUserDoesNotExistMsg).Error())
		})
	})

	t.Run("checkIfUserExistsString", func(t *testing.T) {
		t.Run("Should Return No Error", func(t *testing.T) {
			assert.NoError(t, checkIfUserExistsString(ctx, &zerolog.Logger{}, q, expectedUserId.String()))
		})
		t.Run("Should Return Error", func(t *testing.T) {
			var wrongUUID pgtype.UUID
			if err := wrongUUID.Scan(uuid.NewString()); err != nil {
				log.Fatal(err)
			}

			assert.EqualError(t, checkIfUserExistsString(ctx, &zerolog.Logger{}, q, wrongUUID.String()), status.Error(codes.InvalidArgument, util.InvalidUserIdUserDoesNotExistMsg).Error())
		})
		t.Run("Should Return Error (Invalid UUID)", func(t *testing.T) {
			assert.EqualError(t, checkIfUserExistsString(ctx, &zerolog.Logger{}, q, ""), status.Error(codes.InvalidArgument, util.InvalidUserIdInvalidUUIDMsg).Error())
		})
	})

}
