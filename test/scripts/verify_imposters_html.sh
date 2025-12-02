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

# Verify HTML response
echo "Verifying HTML response..."
response=$(curl -s -H "Accept: text/html" http://localhost:2525/imposters)

if [[ "$response" == *"<!doctype html>"* ]] || [[ "$response" == *"<html>"* ]]; then
    echo "HTML verification passed"
else
    echo "HTML verification failed"
    echo "Expected HTML content"
    echo "Got: $response"
    exit 1
fi

# Verify JSON response
echo "Verifying JSON response..."
response=$(curl -s -H "Accept: application/json" http://localhost:2525/imposters)

if [[ "$response" == *"\"imposters\":[]"* ]]; then
    echo "JSON verification passed"
else
    echo "JSON verification failed"
    echo "Expected JSON content"
    echo "Got: $response"
    exit 1
fi
