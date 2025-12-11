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

# Update the stub WITHOUT the wrapper
echo "Updating stub without wrapper..."
response=$(curl -s -w "%{http_code}" -X PUT http://localhost:2525/imposters/4545/stubs/0 \
  -H "Content-Type: application/json" \
  -d '{
    "responses": [
        {
            "is": {
                "body": "updated"
            }
        }
    ]
}')

http_code=${response: -3}
body=${response:0:${#response}-3}

echo "Response code: $http_code"
echo "Response body: $body"

# Verify updated stub
echo "Verifying updated stub..."
response=$(curl -s http://localhost:4545/)
echo "Got response: $response"

if [ "$response" == "updated" ]; then
    echo "Updated stub verified (wrapper not needed?)"
else
    echo "Updated stub verification failed. Expected 'updated', got '$response'"
fi
