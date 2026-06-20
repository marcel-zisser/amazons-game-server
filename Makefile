.PHONY: proto build build-server build-client run run-server run-client clean help deps

GOBIN := $(shell go env GOBIN)
PLAYER ?= TestBot

# Default target
help:
	@echo "Available targets:"
	@echo "  make proto         - Generate Go code from proto files"
	@echo "  make build         - Build both server and client binaries"
	@echo "  make build-server  - Build the server binary"
	@echo "  make build-client  - Build the client binary"
	@echo "  make run-server    - Run the server"
	@echo "  make run-client    - Run the client (connect to localhost:50051)"
	@echo "  make clean         - Clean generated files and build artifacts"
	@echo "  make deps          - Download and verify dependencies"

# Generate Go code from proto files
proto:
	@echo "Generating Go code from proto files..."
	mkdir -p api/proto/gen
	cp api/proto/*.proto api/proto/gen/
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--plugin=protoc-gen-go="$(GOBIN)/protoc-gen-go" \
		--plugin=protoc-gen-go-grpc="$(GOBIN)/protoc-gen-go-grpc" \
		api/proto/gen/*.proto
	rm api/proto/gen/*.proto

# Download and verify dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

# Build the server and client binaries
build: proto deps build-server build-client
	@echo "✅ Build complete"

# Build the server binary
build-server: proto deps
	@echo "Building server..."
	go build -o bin/amazons-server ./cmd/server

# Build the client binary
build-client: proto deps
	@echo "Building client..."
	go build -o bin/amazons-client ./cmd/client

# Run the server
run-server: build-server
	@echo "Starting server..."
	./bin/amazons-server

# Run the client (set PLAYER variable to pass a player name)
run-client: build-client
	@echo "Running client with player: $(PLAYER)"
	./bin/amazons-client -player $(PLAYER)	

# Alias for run-server (backwards compatibility)
run: run-server

# Clean up generated files and build artifacts
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	rm -rf api/proto/gen
