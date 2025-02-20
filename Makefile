.PHONY: all build test proto clean

all: proto build test

build:
	go build -o bin/timeserver ./cmd/timeserver

test:
	go test -v ./...

proto:
	@echo "Generating gRPC code from protobuf definitions..."
	@if ! which protoc > /dev/null; then \
		echo "protoc not found. Please install Protocol Buffers:"; \
		echo "  Mac: brew install protobuf"; \
		echo "  Linux: apt-get install protobuf-compiler"; \
		echo "  Or download from https://github.com/protocolbuffers/protobuf/releases"; \
		exit 1; \
	fi
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/grpc/timeserver.proto

clean:
	rm -f bin/timeserver
	find . -name "*.pb.go" -type f -delete
