.PHONY: build test clean install run help

# Build the binary
build:
	go build -o sway-compat

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f sway-compat
	rm -f coverage.out coverage.html

# Install the binary to /usr/local/bin
install: build
	sudo cp sway-compat /usr/local/bin/

# Run the CLI
run:
	go run main.go

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o sway-compat-linux-amd64
	GOOS=linux GOARCH=arm64 go build -o sway-compat-linux-arm64

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to /usr/local/bin"
	@echo "  run           - Run the CLI"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  build-all     - Build for multiple platforms"
	@echo "  help          - Show this help message"
