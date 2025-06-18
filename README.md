# Fast Celery Ping

A fast, self-contained Go alternative to Python's `celery inspect ping` command. This tool provides efficient worker discovery for Celery deployments using Redis as the message broker.

## Features

- **Fast**: Significantly faster than the Python implementation
- **Self-contained**: Single binary with no external dependencies
- **Redis Support**: Currently supports Redis broker with easy extensibility
- **Compatible**: Maintains compatibility with existing Celery deployments
- **Configurable**: Supports both environment variables and command-line flags

## Installation

```bash
# Build from source
git clone <repository-url>
cd fast-celery-ping
go build -o fast-celery-ping
```

## Usage

### Basic Usage

```bash
# Use default Redis connection (redis://localhost:6379/0)
./fast-celery-ping

# Specify custom Redis URL
./fast-celery-ping --broker-url redis://localhost:6379/1

# Set custom timeout and verbose output
./fast-celery-ping --timeout 5s --verbose

# Output in text format instead of JSON
./fast-celery-ping --format text
```

### Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `--broker-url` | `CELERY_BROKER_URL` | `redis://localhost:6379/0` | Redis connection URL |
| `--timeout` | `CELERY_PING_TIMEOUT` | `1s` | Timeout for ping responses |
| `--format` | `OUTPUT_FORMAT` | `json` | Output format (json/text) |
| `--database` | `REDIS_DB` | `0` | Redis database number |
| `--username` | `REDIS_USERNAME` | | Redis username |
| `--password` | `REDIS_PASSWORD` | | Redis password |
| `--verbose` | `VERBOSE` | `false` | Enable verbose output |

### Examples

```bash
# Using environment variables
export CELERY_BROKER_URL="redis://localhost:6379/2"
export CELERY_PING_TIMEOUT="3s"
./fast-celery-ping

# Authentication with Redis
./fast-celery-ping --broker-url redis://localhost:6379/0 --username myuser --password mypass

# Text output format
./fast-celery-ping --format text
# Output: worker@hostname: OK pong
#         1 nodes online.

# JSON output format (default)
./fast-celery-ping --format json
# Output: {
#           "worker@hostname": {
#             "ok": "pong"
#           }
#         }
```

## Architecture

The application is built with a modular architecture that supports easy extension:

- **Broker Interface**: Abstract interface for different message brokers
- **Redis Implementation**: Handles Celery's pidbox control message protocol
- **Protocol Handler**: Manages Celery-specific message formatting
- **CLI Interface**: Command-line interface built with Cobra

## Performance

This Go implementation provides significant performance improvements over the Python version:

- Faster startup time (no interpreter overhead)
- Lower memory usage
- More efficient Redis connection handling
- Concurrent message processing

## Future Enhancements

- RabbitMQ broker support
- Additional output formats (XML, YAML)
- Prometheus metrics integration
- Docker image
- Helm chart for Kubernetes deployments

## Development

```bash
# Run tests
go test ./...

# Format code
go fmt ./...

# Check for issues
go vet ./...

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o fast-celery-ping-linux-amd64
GOOS=darwin GOARCH=amd64 go build -o fast-celery-ping-darwin-amd64
GOOS=windows GOARCH=amd64 go build -o fast-celery-ping-windows-amd64.exe
```

## License

This project is licensed under the MIT License. See the [LICENSE.md](LICENSE.md) file for details.

## Contributing

Contributions are welcome! Please open issues or submit pull requests.
