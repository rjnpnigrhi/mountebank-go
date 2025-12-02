#!/bin/bash
set -e
docker run --rm -v $(pwd):/app -w /app golang:1.21-alpine sh test/scripts/verify_dashboard.sh
