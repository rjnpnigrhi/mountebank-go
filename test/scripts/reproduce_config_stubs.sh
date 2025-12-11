#!/bin/bash
set -e

# Build the application
echo "Building mountebank-go..."
go build -o mb cmd/mb/main.go

# Create a config file with stubs
cat > test_config.json <<EOF
{
  "imposters": [
    {
      "port": 4548,
      "protocol": "http",
      "stubs": [
        {
          "predicates": [{ "equals": { "path": "/test" } }],
          "responses": [{ "is": { "statusCode": 200, "body": "Config stub loaded" } }]
        }
      ]
    }
  ]
}
EOF

# Start mountebank with config file
echo "Starting mountebank with --configfile..."
./mb start --configfile test_config.json --nologfile &
MB_PID=$!

# Ensure mountebank is stopped when script exits
trap "kill $MB_PID && rm test_config.json" EXIT

# Wait for mountebank to start
sleep 2

# Check if imposter exists and has stubs
echo "Checking imposter stubs..."
RESPONSE=$(curl -s http://localhost:2525/imposters/4548)
echo "Imposter config: $RESPONSE"

if [[ "$RESPONSE" == *"stubs"* && "$RESPONSE" == *"Config stub loaded"* ]]; then
    # We might need to check the stubs array specifically if 'stubs' key is present but empty
    STUBS_COUNT=$(echo $RESPONSE | grep -o "Config stub loaded" | wc -l)
    if [ "$STUBS_COUNT" -gt 0 ]; then
        echo "✅ Verification passed: Stubs loaded from config file"
    else
        echo "❌ Verification failed: Stubs key present but content missing?"
        exit 1
    fi
else
    echo "❌ Verification failed: Stubs not found in imposter config"
    # Provide more info
    curl -s http://localhost:4548/test
    exit 1
fi
