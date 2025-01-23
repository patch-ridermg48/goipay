dbConnStr = "postgresql://${DATABASE_USER}:${DATABASE_PASS}@${DATABASE_HOST}:${DATABASE_PORT}/${DATABASE_NAME}"
migrationsDir = ./sql/migrations
sqlcFile = ./sql/sqlc.yaml

protoPathDir = ./proto
protoGoOutDir = ./internal/pb
protoGoGrpcOutDir = $(protoGoOutDir)

cmdServerDir = ./cmd/server

add-migrations:
	goose -dir $(migrationsDir) create $(name) sql
run-migrations:
	goose -dir $(migrationsDir) postgres $(dbConnStr) up
reset-migrations:
	goose -dir $(migrationsDir) postgres $(dbConnStr) reset

build:
	go build -o ./bin/server $(cmdServerDir)/main.go
build-debug:
	go build -gcflags=all="-N -l" -o ./bin/server $(cmdServerDir)/main.go

gen-sqlc:
	sqlc -f $(sqlcFile) generate
gen-pb:
	rm -rf $(protoGoOutDir)/* && ./script/generate_pb.sh -i $(protoPathDir) -o $(protoGoOutDir) v1
gen-certs:
	rm -rf ./certs/* && bash -x ./script/generate_certs.sh
gen-mocks:
	mockery --dir=internal/listener --all --inpackage
	mockery --dir=internal/processor --all --inpackage

coverage:
	go test -timeout 600s -coverprofile=.coverage.out -coverpkg=./internal/handler/v1,./internal/listener,./internal/processor,./internal/util,./internal/db ./... && go tool cover -html=.coverage.out -o .coverage.html