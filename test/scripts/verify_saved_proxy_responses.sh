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

# Create an imposter
echo "Creating imposter..."
curl -s -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4545
}'

# Call DeleteSavedProxyResponses
echo "Deleting saved proxy responses..."
response=$(curl -s -w "%{http_code}" -X DELETE http://localhost:2525/imposters/4545/savedProxyResponses)

http_code=${response: -3}
body=${response:0:${#response}-3}

echo "Response code: $http_code"
echo "Response body: $body"

if [ "$http_code" -eq 200 ]; then
    echo "DeleteSavedProxyResponses passed"
else
    echo "DeleteSavedProxyResponses failed"
    exit 1
fi
