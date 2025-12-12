#!/bin/bash
set -e

# Build the application
echo "Building mountebank-go..."
go build -o mb cmd/mb/main.go

# Find random available port
PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("", 0)); print(s.getsockname()[1]); s.close()')

# Start mountebank
./mb start --allowInjection --nologfile --loglevel debug &
MB_PID=$!
trap "kill $MB_PID" EXIT
sleep 2

# Create imposter with throwing injection
echo "Creating imposter on port $PORT..."
curl -i -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d "{
    \"protocol\": \"http\",
    \"port\": $PORT,
    \"stubs\": [{
      \"responses\": [{
        \"inject\": \"function (request, state, logger) { throw new Error('Boom!'); }\"
      }]
    }]
  }"

echo ""
echo "Sending request..."
RESPONSE=$(curl -s http://localhost:$PORT || true)
echo "Response Body: $RESPONSE"

if [[ "$RESPONSE" == *"Boom!"* ]]; then
  echo "✅ Verification passed: Response contains error message"
else
  echo "❌ Verification failed: Response does NOT contain error message"
  echo "Response was: $RESPONSE"
  echo "Server Logs:"
  cat mb.log || true # Assuming --logfile was used? No --nologfile was used.
  # So logs are in stdout/stderr of background process.
  # I need to capture it.
  exit 1
fi
