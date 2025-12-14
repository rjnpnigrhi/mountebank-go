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

# Create an imposter with injection using console.log
echo "Creating imposter with console.log injection..."
curl -i -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4547,
    "stubs": [{
      "responses": [{
        "inject": "function (request, state, logger) { console.log(\"Testing console.log\"); return { statusCode: 200, body: \"Console test\" }; }"
      }]
    }]
  }'

echo ""
echo "Sending request..."
# Send request to imposter
# We expect this to FAIL initially with a 500 error or similar due to ReferenceError
RESPONSE_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:4547)

if [ "$RESPONSE_CODE" == "200" ]; then
  echo "✅ Verification passed (console.log didn't crash)"
else
  echo "❌ Verification failed (Expected 200, got $RESPONSE_CODE)"
  exit 1
fi
