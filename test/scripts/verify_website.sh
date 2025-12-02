#!/bin/bash
set -e

# Build image
docker build -t mountebank-go .

# Run container
docker rm -f mb-go || true
docker run -d -p 2525:2525 -p 4545-4555:4545-4555 --name mb-go mountebank-go

# Wait for server
sleep 2

# Verify home page (HTML)
echo "Verifying home page..."
curl -v -H "Accept: text/html" http://localhost:2525/ > home.html
if grep -q "Welcome, friend" home.html; then
    echo "Home page verified"
else
    echo "Home page verification failed"
    cat home.html
    exit 1
fi

# Verify docs page (HTML)
echo "Verifying docs page..."
curl -v -H "Accept: text/html" http://localhost:2525/docs/api/overview > docs.html
if grep -q "API Overview" docs.html; then
    echo "Docs page verified"
else
    echo "Docs page verification failed"
    cat docs.html
    exit 1
fi

# Verify static asset
echo "Verifying static asset..."
curl -v http://localhost:2525/images/mountebank.png > /dev/null
