#!/bin/bash

# Start the server locally
echo "Building and starting server..."
go build -o /tmp/mb-test cmd/mb/main.go
/tmp/mb-test start &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Create a test imposter
echo "Creating test imposter..."
curl -s -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "port": 4545,
    "protocol": "http",
    "name": "Test Imposter",
    "stubs": [
      {
        "responses": [{ "is": { "statusCode": 200, "body": "Hello" } }]
      }
    ]
  }'

echo ""
echo "Fetching /imposters page..."
curl -H "Accept: text/html" http://localhost:2525/imposters > /tmp/imposters.html

echo "HTML output saved to /tmp/imposters.html"
cat /tmp/imposters.html

# Check if table row exists
if grep -q "<td>Test Imposter</td>" /tmp/imposters.html; then
  echo "✅ SUCCESS: Imposter found in HTML"
else
  echo "❌ FAIL: Imposter not found in HTML"
fi

# Cleanup
kill $SERVER_PID
rm /tmp/mb-test
