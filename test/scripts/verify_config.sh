#!/bin/bash
set -e

echo "Starting mountebank-go container..."
docker run -d --name mb-test -p 2525:2525 -p 4545:4545 mountebank-go:latest

echo "Waiting for server to start..."
sleep 2

echo "Creating imposter..."
curl -X POST http://localhost:2525/imposters \
  -H "Content-Type: application/json" \
  -d '{
    "protocol": "http",
    "port": 4545,
    "stubs": [{
      "responses": [{
        "is": {
          "statusCode": 200,
          "body": "Config Test"
        }
      }]
    }]
  }'

echo "Verifying imposter works..."
RESPONSE=$(curl -s http://localhost:4545)
if [ "$RESPONSE" != "Config Test" ]; then
  echo "Imposter check failed. Expected 'Config Test', got '$RESPONSE'"
  exit 1
fi

echo "Saving configuration..."
docker exec mb-test /app/mb save --savefile /app/config.json

echo "Copying config file to host..."
docker cp mb-test:/app/config.json ./config.json

echo "Stopping container..."
docker stop mb-test
docker rm mb-test

echo "Starting new container with config file..."
docker run -d --name mb-test-2 -p 2525:2525 -p 4545:4545 -v $(pwd)/config.json:/app/config.json mountebank-go:latest start --configfile /app/config.json --host 0.0.0.0

echo "Waiting for server to start..."
sleep 2

echo "Verifying imposter still works..."
RESPONSE=$(curl -s http://localhost:4545)
if [ "$RESPONSE" != "Config Test" ]; then
  echo "Imposter persistence check failed. Expected 'Config Test', got '$RESPONSE'"
  exit 1
fi

echo "Cleaning up..."
docker stop mb-test-2
docker rm mb-test-2
rm config.json

echo "Verification successful!"
