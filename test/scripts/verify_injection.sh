#!/bin/bash
set -e

# Build the application
echo "Building mountebank-go..."
go build -o mb cmd/mb/main.go

# Start mountebank with injection allowed in background
echo "Starting mountebank with --allowInjection..."
./mb start --allowInjection --nologfile &
MB_PID=$!

# Ensure mountebank is stopped when script exits
trap "kill $MB_PID" EXIT

# Wait for mountebank to start
sleep 2

# Create an imposter with injection
echo "Creating imposter with injection..."
curl -i -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4546,
    "stubs": [{
      "responses": [{
        "inject": "function (request, state, logger) { state.count = (state.count || 0) + 1; return { statusCode: 200, body: \"Count: \" + state.count }; }"
      }]
    }]
  }'

echo ""
echo "Sending first request..."
# Send request to imposter
RESPONSE=$(curl -s http://localhost:4546)
echo "Response: $RESPONSE"

if [[ "$RESPONSE" == *"Count: 1"* ]]; then
  echo "✅ First request verification passed"
else
  echo "❌ First request verification failed"
  exit 1
fi

echo "Sending second request..."
# Send second request to verify state persistence
RESPONSE=$(curl -s http://localhost:4546)
echo "Response: $RESPONSE"

if [[ "$RESPONSE" == *"Count: 2"* ]]; then
  echo "✅ Second request verification passed (State persisted)"
else
  echo "❌ Second request verification failed"
  exit 1
fi

echo "✅ Injection verification successful!"
