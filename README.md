# Mountebank Go

A Go port of [mountebank](http://www.mbtest.org/), the open source service virtualization tool.

## Overview

mountebank-go is a complete reimplementation of mountebank in Go, providing:

- **Multiple Protocol Support**: HTTP, HTTPS, TCP, SMTP
- **Powerful Predicates**: Match requests using equals, deepEquals, contains, matches, and more
- **Response Behaviors**: Modify responses with wait, decorate, copy, lookup
- **Proxy Recording**: Record and replay interactions with real services
- **JavaScript Injection**: Custom logic using embedded JavaScript runtime
- **REST API**: Full-featured API for managing imposters
- **Prometheus Metrics**: Built-in observability

## Installation

### Prerequisites

- Go 1.21 or higher

### Build from Source

```bash
git clone https://github.com/mountebank-testing/mountebank-go.git
cd mountebank-go
go build -o mb cmd/mb/main.go
```

### Install

```bash
go install github.com/mountebank-testing/mountebank-go/cmd/mb@latest
```

### Docker

```bash
# Build Docker image
docker build -t mountebank-go .

# Run with Docker
docker run -p 2525:2525 -p 4545-4555:4545-4555 mountebank-go

# Or use docker-compose
docker-compose up -d
```

## Quick Start

### Start the server

```bash
./mb start
```

The server will start on port 2525 by default. Visit http://localhost:2525 for documentation.

### Create an HTTP imposter

```bash
curl -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4545,
    "stubs": [{
      "responses": [{
        "is": {
          "statusCode": 200,
          "headers": {"Content-Type": "application/json"},
          "body": "{\"message\": \"Hello from mountebank-go!\"}"
        }
      }]
    }]
  }'
```

### Test the imposter

```bash
curl http://localhost:4545
# Returns: {"message": "Hello from mountebank-go!"}
```

## Command Line Options

```bash
# Start server on custom port
./mb start --port 3000

# Enable JavaScript injection (use with caution)
./mb start --allowInjection

# Set log level
./mb start --loglevel debug

# Load configuration from file
./mb start --configfile imposters.json

# Stop server
./mb stop

# Restart server
./mb restart

# Save current imposters
./mb save --savefile imposters.json

# Replay saved imposters
./mb replay --configfile imposters.json
```

## Features

### Predicates

Match incoming requests using various predicates:

- `equals`: Exact match
- `deepEquals`: Deep object comparison
- `contains`: Substring match
- `startsWith`: Prefix match
- `endsWith`: Suffix match
- `matches`: Regular expression match
- `exists`: Field presence check
- `not`: Logical negation
- `or`: Logical OR
- `and`: Logical AND
- `inject`: Custom JavaScript logic

### Behaviors

Transform responses with behaviors:

- `wait`: Add latency
- `decorate`: Modify response with JavaScript
- `copy`: Copy values from request to response
- `lookup`: Lookup values from data source
- `shellTransform`: Transform using shell command

### Proxy Modes

Record interactions with real services:

- `proxyOnce`: Record once, replay thereafter
- `proxyAlways`: Always proxy and record
- `proxyTransparent`: Proxy without recording

## API Documentation

Full API documentation is available at http://localhost:2525 when the server is running.

### Key Endpoints

- `GET /imposters` - List all imposters
- `POST /imposters` - Create an imposter
- `GET /imposters/:port` - Get imposter details
- `DELETE /imposters/:port` - Delete an imposter
- `PUT /imposters/:port/stubs` - Replace all stubs
- `POST /imposters/:port/stubs` - Add a stub
- `DELETE /imposters/:port/stubs/:index` - Delete a stub
- `GET /metrics` - Prometheus metrics

## Differences from JavaScript Version

### Performance

- **Faster startup**: Go's compiled nature provides instant startup
- **Lower memory**: More efficient memory usage
- **Better concurrency**: Native goroutines for handling concurrent requests

### Breaking Changes

1. **Configuration format**: Minor differences due to Go's type system
2. **Error messages**: Formatted according to Go conventions
3. **JavaScript injection**: Uses goja runtime (slight compatibility differences)

### Migration from JavaScript Version

1. Export your configuration:
   ```bash
   mb save --savefile config.json
   ```

2. Install mountebank-go and import:
   ```bash
   ./mb replay --configfile config.json
   ```

3. Test thoroughly before production use

## Development

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test ./test/integration/...

# With coverage
go test -cover ./...
```

### Building

```bash
# Development build
go build -o mb cmd/mb/main.go

# Production build with optimizations
go build -ldflags="-s -w" -o mb cmd/mb/main.go
```

## Docker

### Building the Docker Image

```bash
# Build the image
docker build -t mountebank-go:latest .

# Build with specific version tag
docker build -t mountebank-go:2.9.3 .
```

### Running with Docker

```bash
# Run mountebank server
docker run -d \
  --name mountebank \
  -p 2525:2525 \
  -p 4545-4555:4545-4555 \
  mountebank-go:latest

# Run with custom options
docker run -d \
  --name mountebank \
  -p 2525:2525 \
  -e MB_LOG_LEVEL=debug \
  mountebank-go:latest start --loglevel debug

# View logs
docker logs -f mountebank

# Stop container
docker stop mountebank
```

### Using Docker Compose

```bash
# Start services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Rebuild and restart
docker-compose up -d --build
```

### Docker Image Details

- **Base Image**: Alpine Linux (minimal size)
- **Size**: ~20MB (multi-stage build)
- **User**: Non-root user for security
- **Health Check**: Built-in health check on port 2525
- **Exposed Ports**: 2525 (main), 4545-4555 (imposters)

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

This project is a port of the original [mountebank](https://github.com/mountebank-testing/mountebank) created by Brandon Byars. All credit for the design and concepts goes to the original project.
