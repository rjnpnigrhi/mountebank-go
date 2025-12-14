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

# Create an imposter with a stub
echo "Creating imposter..."
curl -s -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4545,
    "stubs": [
        {
            "responses": [
                {
                    "is": {
                        "body": "original"
                    }
                }
            ]
        }
    ]
}'

# Verify original stub
echo "Verifying original stub..."
response=$(curl -s http://localhost:4545/)
if [ "$response" == "original" ]; then
    echo "Original stub verified"
else
    echo "Original stub verification failed. Got: $response"
    exit 1
fi

# Update the stub at index 0
echo "Updating stub..."
curl -s -X PUT http://localhost:2525/imposters/4545/stubs/0 \
  -H "Content-Type: application/json" \
  -d '{
    "stub": {
        "responses": [
            {
                "is": {
                    "body": "updated"
                }
            }
        ]
    }
}'

# Verify updated stub
echo "Verifying updated stub..."
response=$(curl -s http://localhost:4545/)
if [ "$response" == "updated" ]; then
    echo "Updated stub verified"
else
    echo "Updated stub verification failed. Expected 'updated', got '$response'"
    exit 1
fi
