#!/bin/bash
set -e

# Build the application
echo "Building mountebank-go..."
go build -o mb cmd/mb/main.go

# Start mountebank with config file
echo "Starting mountebank with --configfile impostersejs/test/imposters.ejs..."
./mb start --configfile impostersejs/test/imposters.ejs --allowInjection --nologfile --loglevel info &
MB_PID=$!

# Ensure mountebank is stopped when script exits
trap "kill $MB_PID" EXIT

# Wait for mountebank to start
sleep 2

# Check if imposter exists and has stubs
echo "Checking imposter stubs (port 4012)..."
RESPONSE=$(curl -s http://localhost:2525/imposters/4012)
echo "Imposter config: $RESPONSE"

if [[ "$RESPONSE" == *"stubs"* && "$RESPONSE" == *"360customer"* ]]; then
    # Parse json to count stubs
    # Using python specifically because grep was unreliable
    STUBS_LEN=$(echo $RESPONSE | python3 -c "import sys, json; print(len(json.load(sys.stdin).get('stubs', [])))")
    echo "Stubs count: $STUBS_LEN"
    
    if [ "$STUBS_LEN" -gt 0 ]; then
        echo "✅ Verification passed: Stubs loaded from EJS config file"
    else
        echo "❌ Verification failed: Stubs array is empty"
        exit 1
    fi
else
    echo "❌ Verification failed: Imposter or stubs not found"
    exit 1
fi
