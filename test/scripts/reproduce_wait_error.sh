#!/bin/bash

# Build and run docker container
docker build --no-cache -t mb-go .
docker rm -f mb-go || true
docker run -d --name mb-go -p 2525:2525 -p 4545:4545 mb-go start

# Wait for server to start
sleep 5

# Create imposter with wait behavior as number
echo "Creating imposter with wait behavior..."
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
              { "wait": 1000 }
            ]
          }
        ]
      }
    ]
  ' 2>&1)

echo "Response: $response"

if echo "$response" | grep -q "cannot unmarshal number"; then
  echo "FAIL: Reproduced unmarshaling error"
elif echo "$response" | grep -q "201 Created"; then
  echo "PASS: No unmarshaling error (Created)"
else
  echo "UNKNOWN: Unexpected response"
  docker logs mb-go
fi

# Cleanup
docker rm -f mb-go
