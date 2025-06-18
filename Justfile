# Fast Celery Ping - Development Tasks
#
# Use `just --list` to see all available commands

# Variables
APP_NAME := "fast-celery-ping"
VERSION := "1.0.0"
BUILD_TIME := `date -u +%Y-%m-%dT%H:%M:%SZ`
LDFLAGS := "-s -w -X 'fast-celery-ping/cmd.Version=" + VERSION + "' -X 'fast-celery-ping/cmd.BuildTime=" + BUILD_TIME + "'"

# Default recipe - show help
default:
    @just --list

# Build the application
build:
    @echo "Building {{APP_NAME}}..."
    CGO_ENABLED=0 go build -ldflags="{{LDFLAGS}}" -o {{APP_NAME}} .

# Build for multiple platforms
build-all:
    @echo "Building for multiple platforms..."
    mkdir -p dist
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="{{LDFLAGS}}" -o dist/{{APP_NAME}}-linux-amd64 .
    GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="{{LDFLAGS}}" -o dist/{{APP_NAME}}-linux-arm64 .
    GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="{{LDFLAGS}}" -o dist/{{APP_NAME}}-darwin-amd64 .
    GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="{{LDFLAGS}}" -o dist/{{APP_NAME}}-darwin-arm64 .

# Run tests
test:
    @echo "Running tests..."
    go test -v ./...

# Run tests with coverage
test-coverage:
    @echo "Running tests with coverage..."
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run tests and show coverage in terminal
test-cover:
    @echo "Running tests with coverage..."
    go test -v -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Run benchmarks
bench:
    @echo "Running benchmarks..."
    go test -bench=. -benchmem ./...

# Format code
fmt:
    @echo "Formatting code..."
    go fmt ./...

# Run linter (requires golangci-lint)
lint:
    @echo "Running linter..."
    golangci-lint run

# Tidy dependencies
tidy:
    @echo "Tidying dependencies..."
    go mod tidy

# Update dependencies
update:
    @echo "Updating dependencies..."
    go get -u ./...
    go mod tidy

# Clean build artifacts
clean:
    @echo "Cleaning build artifacts..."
    rm -f {{APP_NAME}}
    rm -rf dist/
    rm -f coverage.out coverage.html

# Install the binary to $GOPATH/bin
install:
    @echo "Installing {{APP_NAME}}..."
    go install -ldflags="{{LDFLAGS}}" .

# Run the application with default help
run *ARGS:
    @echo "Running {{APP_NAME}}..."
    go run . {{ARGS}}

# Quick development test - build and run with Redis
dev-test: build
    @echo "Testing with local Redis..."
    ./{{APP_NAME}} --broker-url redis://localhost:6379/0 --verbose --timeout 5s

# Build Docker image
docker-build:
    @echo "Building Docker image..."
    docker build -t {{APP_NAME}}:{{VERSION}} .
    docker build -t {{APP_NAME}}:latest .

# Build Docker image with multi-platform support
docker-build-multi *ARGS:
    @echo "Building multi-platform Docker image..."
    docker buildx build --platform linux/amd64,linux/arm64 -t {{APP_NAME}}:{{VERSION}} -t {{APP_NAME}}:latest {{ARGS}} .

# Run Docker container
docker-run *ARGS:
    @echo "Running Docker container..."
    docker run --rm {{APP_NAME}}:latest {{ARGS}}
