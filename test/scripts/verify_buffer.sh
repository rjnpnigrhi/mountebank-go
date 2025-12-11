#!/bin/bash
set -e

# Build the application
echo "Building mountebank-go..."
go build -o mb cmd/mb/main.go

# Start mountebank with injection allowed in background
echo "Starting mountebank with --allowInjection..."
./mb start --allowInjection --nologfile --loglevel debug &
MB_PID=$!

# Ensure mountebank is stopped when script exits
trap "kill $MB_PID" EXIT

# Wait for mountebank to start
sleep 2

# Find random available port
PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()')

# Create an imposter with injection using Buffer
echo "Creating imposter with Buffer injection on port $PORT..."
curl -i -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d "{
    \"protocol\": \"http\",
    \"port\": $PORT,
    \"stubs\": [{
      \"responses\": [{
        \"inject\": \"function (request, state, logger) { var b = Buffer.from(\\\"hello\\\"); return { statusCode: 200, body: b.toString(\\\"base64\\\") }; }\"
      }]
    }]
  }"

echo ""
echo "Sending request..."
# Send request to imposter
# We expect this to FAIL initially with a 500 error due to ReferenceError
RESPONSE_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:$PORT)

if [ "$RESPONSE_CODE" == "200" ]; then
  echo "✅ Verification passed (Buffer works)"
else
  echo "❌ Verification failed (Expected 200, got $RESPONSE_CODE)"
  curl -s http://localhost:$PORT
  echo ""
  exit 1
fi
