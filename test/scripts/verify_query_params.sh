#!/bin/bash

# Build and run docker container
docker build --no-cache -t mb-go .
docker rm -f mb-go || true
docker run -d --name mb-go -p 2525:2525 -p 4546:4546 mb-go start

# Wait for server to start
sleep 5

# Create an imposter
echo "Creating imposter..."
curl -s -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "port": 4546,
    "protocol": "http",
    "recordRequests": true,
    "stubs": [
      {
        "predicates": [{ "equals": { "method": "GET" } }],
        "responses": [{ "is": { "statusCode": 200 } }]
      }
    ]
  }'

# Send a request to generate a record
# echo "Sending request..."
# curl -s http://localhost:4546/

# Test 1: GET /imposters (list)
echo "Test 1: GET /imposters"
response=$(curl -v http://localhost:2525/imposters 2>&1)
if echo "$response" | grep -q '"imposters":\['; then
  echo "PASS: imposters found"
else
  echo "FAIL: imposters not found"
  echo "Response: $response"
  docker logs mb-go
fi

# Test 2: GET /imposters/4546?replayable=true - should NOT have requests
echo "Test 2: Replayable GET"
response=$(curl -s "http://localhost:2525/imposters/4546?replayable=true")
if echo "$response" | grep -q '"requests":\['; then
  echo "FAIL: requests found when replayable=true"
  echo "Response: $response"
else
  echo "PASS: requests not found when replayable=true"
  echo "Response: $response"
fi

# Cleanup
docker rm -f mb-go
