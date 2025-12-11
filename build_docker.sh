#!/bin/bash
set -e

# Default variables
IMAGE_NAME="rjnpnigrhi/mountebank-go"
DATE_TAG=$(date +%Y%m%d)

# Check for docker buildx
if ! docker buildx version > /dev/null 2>&1; then
    echo "Error: docker buildx is not available. Please install Docker Desktop or enable buildx."
    exit 1
fi

# Create and bootstrap a new builder if it doesn't exist
BUILDER_NAME="multiplatform-builder"
if ! docker buildx inspect $BUILDER_NAME > /dev/null 2>&1; then
    echo "Creating new builder: $BUILDER_NAME"
    docker buildx create --name $BUILDER_NAME --use
    docker buildx inspect --bootstrap
else
    echo "Using existing builder: $BUILDER_NAME"
    docker buildx use $BUILDER_NAME
fi

echo "Building and pushing image: $IMAGE_NAME"
echo "Tags: latest, $DATE_TAG"

docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t "$IMAGE_NAME:latest" \
  -t "$IMAGE_NAME:$DATE_TAG" \
  --push .

echo "Build complete!"
