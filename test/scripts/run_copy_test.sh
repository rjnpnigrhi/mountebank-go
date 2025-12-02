#!/bin/bash
set -e
docker run --rm -v $(pwd):/app -w /app golang:1.21-alpine go test -v test/integration/copy_test.go
