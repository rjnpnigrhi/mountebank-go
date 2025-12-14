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

# Create an initial imposter
echo "Creating initial imposter..."
curl -s -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4545
}'

# Verify it exists
echo "Verifying initial imposter..."
response=$(curl -s http://localhost:2525/imposters/4545)
if [[ "$response" == *"\"port\":4545"* ]]; then
    echo "Initial imposter verified"
else
    echo "Initial imposter verification failed"
    exit 1
fi

# Overwrite with a new list (different port)
echo "Overwriting imposters..."
response=$(curl -s -w "%{http_code}" -X PUT http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "imposters": [
        {
            "protocol": "http",
            "port": 4546
        }
    ]
}')

http_code=${response: -3}
body=${response:0:${#response}-3}

echo "Response code: $http_code"
echo "Response body: $body"

if [ "$http_code" -ne 200 ]; then
    echo "Overwrite failed"
    exit 1
fi

# Verify old imposter is gone
echo "Verifying old imposter is gone..."
response=$(curl -s -w "%{http_code}" http://localhost:2525/imposters/4545)
http_code_old=${response: -3}
if [ "$http_code_old" -eq 404 ]; then
    echo "Old imposter gone"
else
    echo "Old imposter still exists (code $http_code_old)"
    exit 1
fi

# Verify new imposter exists
echo "Verifying new imposter exists..."
response=$(curl -s http://localhost:2525/imposters/4546)
if [[ "$response" == *"\"port\":4546"* ]]; then
    echo "New imposter verified"
else
    echo "New imposter verification failed"
    exit 1
fi
