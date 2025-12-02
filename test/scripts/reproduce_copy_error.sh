#!/bin/bash

# Build and run docker container
docker build --no-cache -t mb-go .
docker rm -f mb-go || true
docker run -d --name mb-go -p 2525:2525 -p 4545:4545 mb-go start

# Wait for server to start
sleep 5

# Create imposter with copy behavior as object (not array)
echo "Creating imposter with copy behavior as object..."
response=$(curl -v -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "port": 4545,
    "protocol": "http",
    "stubs": [
      {
        "responses": [
          {
            "is": { "statusCode": 200 },
            "behaviors": [
              {
                "copy": {
                  "from": "path",
                  "into": "${BOOK}",
                  "using": { "method": "regex", "selector": "\\d+" }
                }
              }
            ]
          }
        ]
      }
    ]
  ' 2>&1)

echo "Response: $response"

if echo "$response" | grep -q "cannot unmarshal"; then
  echo "FAIL: Reproduced unmarshaling error"
elif echo "$response" | grep -q "201 Created"; then
  echo "PASS: No unmarshaling error (Created)"
else
  echo "UNKNOWN: Unexpected response"
  docker logs mb-go
fi

# Cleanup
docker rm -f mb-go
