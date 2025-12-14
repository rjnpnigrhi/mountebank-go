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
                        "body": "test"
                    }
                }
            ]
        }
    ]
}'

# Get the imposter
echo "Getting imposter..."
response=$(curl -s http://localhost:2525/imposters/4545)
echo "Response: $response"

# Check if stubs are present
if [[ "$response" == *"\"stubs\":["* ]]; then
    echo "Stubs found in response"
else
    echo "Stubs NOT found in response"
    exit 1
fi
