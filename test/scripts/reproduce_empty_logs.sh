#!/bin/bash
set -e

# Build the docker image
echo "Building docker image..."
docker build -t mb-go .

# Run the container
echo "Running container..."
# Remove existing container if it exists
docker rm -f mb-go || true
docker run -d --name mb-go -p 2525:2525 -p 4545-4555:4545-4555 mb-go

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

# Get logs immediately
echo "Getting logs..."
response=$(curl -s http://localhost:2525/logs)
echo "Response: $response"

if [[ "$response" == *"\"logs\":[]"* ]]; then
    echo "Reproduced: Logs are empty"
    exit 1
elif [[ "$response" == *"\"logs\":["* ]]; then
    echo "Logs found"
else
    echo "Unexpected response"
    exit 1
fi
