#!/bin/bash
set -e

# Build the docker image
echo "Building docker image..."
docker build -t mb-go .

# Run the container
echo "Running container..."
docker run -d --name mb-go -p 2525:2525 -p 8443:8443 mb-go

# Function to clean up
cleanup() {
    echo "Cleaning up..."
    docker stop mb-go
    docker rm mb-go
}
trap cleanup EXIT

# Wait for server to start
echo "Waiting for server to start..."
sleep 2

# Create HTTPS imposter
echo "Creating HTTPS imposter..."
curl -v -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "https",
    "port": 8443,
    "stubs": [
        {
            "responses": [
                {
                    "is": {
                        "statusCode": 200,
                        "body": "Hello HTTPS"
                    }
                }
            ]
        }
    ]
}'

# Verify HTTPS response
echo "Verifying HTTPS response..."
response=$(curl -k -s https://localhost:8443)

if [ "$response" == "Hello HTTPS" ]; then
    echo "HTTPS verification passed"
else
    echo "HTTPS verification failed"
    echo "Expected: Hello HTTPS"
    echo "Got: $response"
    exit 1
fi
