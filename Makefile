BIN := bin
GOFLAGS := -trimpath -ldflags "-s -w"

.PHONY: test vet fmt build clean run-mortar run-sunder

test:
	go test ./... -count=1

vet:
	go vet ./...

fmt:
	gofmt -l -w .

build:
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BIN)/mortard ./cmd/mortard
	CGO_ENABLED=0 go build $(GOFLAGS) -o $(BIN)/sunderd ./cmd/sunderd

clean:
	rm -rf $(BIN) data

run-mortar:
	ADDR=:8091 DATA_DIR=./data/mortar MIGRATIONS_DIR=./migrations go run ./cmd/mortard

run-sunder:
	ADDR=:8092 DATA_DIR=./data/sunder MIGRATIONS_DIR=./migrations DAILY_SEED_SECRET=dev-secret go run ./cmd/sunderd
