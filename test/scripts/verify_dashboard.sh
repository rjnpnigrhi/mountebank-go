#!/bin/sh
set -e

# Install curl if missing (Alpine)
if ! command -v curl >/dev/null 2>&1; then
    apk add --no-cache curl
fi

# Build the binary
echo "Building mountebank-go..."
go build -o mb cmd/mb/main.go

# Start the server in background
./mb start --port 2529 &
PID=$!
echo "Server started with PID $PID"

# Wait for server to start
sleep 2

# Function to cleanup
cleanup() {
    echo "Stopping server..."
    kill $PID
    rm mb
}
trap cleanup EXIT

# Test 1: Dashboard (HTML)
echo "Testing Dashboard (HTML)..."
RESPONSE=$(curl -s -H "Accept: text/html" http://localhost:2529/)
if [[ "$RESPONSE" == *"<!DOCTYPE html>"* ]]; then
    echo "✅ Dashboard served correctly"
else
    echo "❌ Dashboard failed. Response:"
    echo "$RESPONSE"
    exit 1
fi

# Test 2: API (JSON)
echo "Testing API (JSON)..."
RESPONSE=$(curl -s -H "Accept: application/json" http://localhost:2529/)
if [[ "$RESPONSE" == *"_links"* ]]; then
    echo "✅ API served correctly"
else
    echo "❌ API failed. Response:"
    echo "$RESPONSE"
    exit 1
fi

# Test 3: Static Assets
echo "Testing Static Assets..."
RESPONSE=$(curl -s http://localhost:2529/style.css)
if [[ "$RESPONSE" == *"--primary-color"* ]]; then
    echo "✅ CSS served correctly"
else
    echo "❌ CSS failed. Response:"
    echo "$RESPONSE"
    exit 1
fi

echo "All tests passed!"
