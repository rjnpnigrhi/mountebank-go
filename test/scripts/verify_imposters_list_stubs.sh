#!/bin/bash

# Build and run docker container
docker build --no-cache -t mb-go .
docker rm -f mb-go || true
docker run -d --name mb-go -p 2525:2525 -p 4545:4545 mb-go start

# Wait for server to start
sleep 5

# Create imposter with stubs
echo "Creating imposter..."
curl -s -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "port": 4545,
    "protocol": "http",
    "stubs": [
      {
        "responses": [{ "is": { "statusCode": 200 } }]
      }
    ]
  }'

# Test 1: GET /imposters (list) - Should NOT have stubs (according to user)
echo "Test 1: GET /imposters (default)"
response=$(curl -s http://localhost:2525/imposters)
if echo "$response" | grep -q '"stubs":\['; then
  echo "FAIL: stubs found in list (default)"
else
  echo "PASS: stubs not found in list (default)"
fi

# Test 2: GET /imposters?replayable=true - Should HAVE stubs
echo "Test 2: GET /imposters?replayable=true"
response=$(curl -s "http://localhost:2525/imposters?replayable=true")
if echo "$response" | grep -q '"stubs":\['; then
  echo "PASS: stubs found when replayable=true"
else
  echo "FAIL: stubs not found when replayable=true"
fi

# Test 3: GET /imposters/4545 (single) - Should HAVE stubs
echo "Test 3: GET /imposters/4545"
response=$(curl -s http://localhost:2525/imposters/4545)
if echo "$response" | grep -q '"stubs":\['; then
  echo "PASS: stubs found in single imposter"
else
  echo "FAIL: stubs not found in single imposter"
fi

# Cleanup
docker rm -f mb-go
