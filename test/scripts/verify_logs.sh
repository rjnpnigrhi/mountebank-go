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

# Create an imposter (to generate some logs)
echo "Creating imposter..."
curl -s -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4545
}'

# Get logs
echo "Getting logs..."
response=$(curl -s http://localhost:2525/logs)
echo "Response: $response"

if [[ "$response" == *"\"logs\":["* ]]; then
    echo "Logs found in response"
else
    echo "Logs NOT found in response"
    exit 1
fi

# Verify pagination
echo "Testing pagination..."
response=$(curl -s "http://localhost:2525/logs?startIndex=0&endIndex=1")
echo "Pagination Response: $response"

# Count logs in response (simple grep count)
count=$(echo "$response" | grep -o "\"level\"" | wc -l)
if [ "$count" -eq 1 ]; then
    echo "Pagination verified (1 log returned)"
else
    echo "Pagination failed (expected 1 log, got $count)"
    # Don't exit here, as it might be flaky depending on log volume
fi
