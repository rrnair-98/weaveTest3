.PHONY: all proto build run test

SERVER_BIN :=

PROTO_OUT_DIR := internal/proto/generated

CURRENT_OS := nix
COMPILE_COMMAND :=
ifeq ($(shell go env GOOS),windows)
    SERVER_BIN := server.exe
else
	SERVER_BIN := server
endif

all: build

proto:
	@echo "generating protobuf sources ..."
	protoc internal/proto/gh_service.proto --go-grpc_out=. --go_out=.
	@echo "protobuf source generated"

build: proto
	@echo "building gRPC server..."
	go mod download
	go build -o $(SERVER_BIN) cmd/weaveTest/main.go
	@echo "gRPC server built"

test: build
	@echo "running tests"
	go clean -testcache
	go test ./...
	@echo "done running tests"

run: build
	@echo "running gRPC github server"
	./$(SERVER_BIN)

clean:
	rm -f $(SERVER_BIN)
