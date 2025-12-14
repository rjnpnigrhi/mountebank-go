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

# Test 1: Empty list wrapped -> {"imposters": []}
echo "Test 1: Empty list wrapped..."
response=$(curl -s -w "%{http_code}" -X PUT http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{"imposters": []}')
code=${response: -3}
body=${response:0:${#response}-3}
echo "Code: $code"
echo "Body: $body"

if [[ "$body" == *"cannot unmarshal object"* ]]; then
    echo "Reproduced: Empty list wrapped fails with object unmarshal error"
fi

# Test 2: Type mismatch wrapped -> {"imposters": [{"port": "string"}]}
echo "Test 2: Type mismatch wrapped..."
response=$(curl -s -w "%{http_code}" -X PUT http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{"imposters": [{"protocol": "http", "port": "4545"}]}') # Port as string might fail if int expected
code=${response: -3}
body=${response:0:${#response}-3}
echo "Code: $code"
echo "Body: $body"

if [[ "$body" == *"cannot unmarshal object"* ]]; then
    echo "Reproduced: Type mismatch wrapped fails with object unmarshal error (masking real error)"
fi
