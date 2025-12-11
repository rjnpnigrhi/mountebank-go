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

# Add a stub using the wrapped format
echo "Adding stub..."
response=$(curl -s -w "%{http_code}" -X POST http://localhost:2525/imposters/4545/stubs \
  -H "Content-Type: application/json" \
  -d '{
    "stub": {
        "responses": [
            {
                "is": {
                    "body": "hello world"
                }
            }
        ]
    }
}')

http_code=${response: -3}
body=${response:0:${#response}-3}

echo "Response code: $http_code"
echo "Response body: $body"

# Check if the stub was added correctly
# We expect 200 OK and the body to contain the stub
if [ "$http_code" -eq 200 ]; then
    # Verify the stub is actually there by calling the imposter
    echo "Verifying stub..."
    stub_response=$(curl -s http://localhost:4545/)
    if [ "$stub_response" == "hello world" ]; then
        echo "Stub verification passed"
    else
        echo "Stub verification failed. Expected 'hello world', got '$stub_response'"
        exit 1
    fi
else
    echo "Add stub failed"
    exit 1
fi
