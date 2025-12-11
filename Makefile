.PHONY: build test clean run install

# Build the binary
build:
	go build -o mb cmd/mb/main.go

# Build with optimizations for production
build-prod:
	go build -ldflags="-s -w" -o mb cmd/mb/main.go

# Run tests
test:
	go test ./... -v

# Run tests with coverage
test-cover:
	go test -cover ./...

# Clean build artifacts
clean:
	rm -f mb
	rm -f mb.pid

# Run the server
run: build
	./mb start --allowInjection

# Install dependencies
deps:
	go mod download
	go mod tidy

# Install the binary
install:
	go install cmd/mb/main.go

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Run integration tests
test-integration:
	go test ./test/integration/... -v

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o mb-linux-amd64 cmd/mb/main.go
	GOOS=darwin GOARCH=amd64 go build -o mb-darwin-amd64 cmd/mb/main.go
	GOOS=darwin GOARCH=arm64 go build -o mb-darwin-arm64 cmd/mb/main.go
	GOOS=windows GOARCH=amd64 go build -o mb-windows-amd64.exe cmd/mb/main.go

# Docker targets
docker-build:
	docker build -t mountebank-go:latest .

docker-run:
	docker run -d --name mountebank -p 2525:2525 -p 4545-4555:4545-4555 mountebank-go:latest

docker-stop:
	docker stop mountebank || true
	docker rm mountebank || true

docker-logs:
	docker logs -f mountebank

docker-compose-up:
	docker-compose up -d

docker-compose-down:
	docker-compose down

docker-compose-logs:
	docker-compose logs -f

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the binary"
	@echo "  build-prod    - Build with optimizations"
	@echo "  test          - Run tests"
	@echo "  test-cover    - Run tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  run           - Build and run the server"
	@echo "  deps          - Install dependencies"
	@echo "  install       - Install the binary"
	@echo "  fmt           - Format code"
	@echo "  lint          - Lint code"
	@echo "  build-all     - Build for multiple platforms"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build        - Build Docker image"
	@echo "  docker-run          - Run Docker container"
	@echo "  docker-stop         - Stop and remove Docker container"
	@echo "  docker-logs         - View Docker container logs"
	@echo "  docker-compose-up   - Start with docker-compose"
	@echo "  docker-compose-down - Stop docker-compose services"
	@echo "  docker-compose-logs - View docker-compose logs"
