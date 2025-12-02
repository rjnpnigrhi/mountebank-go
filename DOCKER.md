# Docker Quick Start Guide

## Building the Image

```bash
# Build the Docker image
docker build -t mountebank-go:latest .

# Or use make
make docker-build
```

## Running the Container

### Basic Usage

```bash
# Run mountebank server
docker run -d \
  --name mountebank \
  -p 2525:2525 \
  -p 4545-4555:4545-4555 \
  mountebank-go:latest

# Or use make
make docker-run
```

### With Custom Options

```bash
# Run with debug logging
docker run -d \
  --name mountebank \
  -p 2525:2525 \
  mountebank-go:latest start --loglevel debug --port 2525

# Run with JavaScript injection enabled (use with caution)
docker run -d \
  --name mountebank \
  -p 2525:2525 \
  mountebank-go:latest start --allowInjection
```

### With Volume Mounts

```bash
# Mount configuration file
docker run -d \
  --name mountebank \
  -p 2525:2525 \
  -v $(pwd)/config.json:/app/config.json \
  mountebank-go:latest start --configfile /app/config.json

# Mount data directory for persistence
docker run -d \
  --name mountebank \
  -p 2525:2525 \
  -v $(pwd)/data:/app/data \
  mountebank-go:latest
```

## Managing the Container

```bash
# View logs
docker logs -f mountebank

# Or use make
make docker-logs

# Stop container
docker stop mountebank

# Remove container
docker rm mountebank

# Stop and remove
make docker-stop

# Restart container
docker restart mountebank
```

## Using Docker Compose

### Start Services

```bash
# Start in detached mode
docker-compose up -d

# Or use make
make docker-compose-up

# Start with build
docker-compose up -d --build
```

### Manage Services

```bash
# View logs
docker-compose logs -f

# Or use make
make docker-compose-logs

# Stop services
docker-compose down

# Or use make
make docker-compose-down

# Restart services
docker-compose restart
```

### Custom Configuration

Edit `docker-compose.yml` to customize:

```yaml
services:
  mountebank:
    environment:
      - MB_LOG_LEVEL=debug  # Change log level
    ports:
      - "3000:2525"         # Change host port
    volumes:
      - ./config.json:/app/config.json  # Mount config
```

## Testing the Container

```bash
# Check if container is running
docker ps | grep mountebank

# Check health status
docker inspect --format='{{.State.Health.Status}}' mountebank

# Test the API
curl http://localhost:2525/

# Create a test imposter
curl -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4545,
    "stubs": [{
      "responses": [{
        "is": {
          "statusCode": 200,
          "body": "Hello from Docker!"
        }
      }]
    }]
  }'

# Test the imposter
curl http://localhost:4545
```

## Troubleshooting

### Container won't start

```bash
# Check logs
docker logs mountebank

# Check if port is already in use
lsof -i :2525

# Run in foreground to see errors
docker run --rm -p 2525:2525 mountebank-go:latest
```

### Can't connect to imposters

```bash
# Make sure ports are exposed
docker ps

# Check if imposter was created
curl http://localhost:2525/imposters

# Check container networking
docker inspect mountebank | grep IPAddress
```

### Image is too large

The image uses multi-stage builds and Alpine Linux to minimize size:

```bash
# Check image size
docker images | grep mountebank-go

# Expected size: ~20-30MB
```

## Production Deployment

### Best Practices

1. **Use specific version tags**:
   ```bash
   docker build -t mountebank-go:2.9.3 .
   docker run -d mountebank-go:2.9.3
   ```

2. **Set resource limits**:
   ```bash
   docker run -d \
     --memory="512m" \
     --cpus="1.0" \
     mountebank-go:latest
   ```

3. **Use health checks**:
   ```bash
   # Health check is built into the image
   docker inspect --format='{{.State.Health}}' mountebank
   ```

4. **Enable restart policy**:
   ```bash
   docker run -d \
     --restart=unless-stopped \
     mountebank-go:latest
   ```

5. **Use Docker Compose for production**:
   ```yaml
   services:
     mountebank:
       image: mountebank-go:2.9.3
       restart: unless-stopped
       deploy:
         resources:
           limits:
             memory: 512M
             cpus: '1.0'
   ```

## Kubernetes Deployment

### Basic Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mountebank
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mountebank
  template:
    metadata:
      labels:
        app: mountebank
    spec:
      containers:
      - name: mountebank
        image: mountebank-go:latest
        ports:
        - containerPort: 2525
        livenessProbe:
          httpGet:
            path: /
            port: 2525
          initialDelaySeconds: 5
          periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: mountebank
spec:
  selector:
    app: mountebank
  ports:
  - port: 2525
    targetPort: 2525
  type: LoadBalancer
```

## Environment Variables

The container supports these environment variables:

- `MB_PORT`: Server port (default: 2525)
- `MB_LOG_LEVEL`: Log level (debug, info, warn, error)
- `MB_ALLOW_INJECTION`: Enable JavaScript injection (true/false)

Example:

```bash
docker run -d \
  -e MB_PORT=3000 \
  -e MB_LOG_LEVEL=debug \
  -p 3000:3000 \
  mountebank-go:latest
```
