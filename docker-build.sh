#!/bin/bash

# Docker Build and Push Script for AI Pro (Multi-Platform)
# Builds for: linux/amd64, linux/arm64 (works on Windows, Linux, and Mac)
# Usage: ./docker-build.sh [your-dockerhub-username] [tag] [--local-only]

set -e

# Get Docker Hub username from argument or environment variable
DOCKER_USERNAME=${1:-${DOCKER_USERNAME:-"your-username"}}
IMAGE_NAME="ai-pro"
TAG=${2:-"latest"}

# Check if local-only build is requested (as 3rd argument)
LOCAL_ONLY=false
if [[ "$3" == "--local-only" ]]; then
    LOCAL_ONLY=true
fi

FULL_IMAGE_NAME="${DOCKER_USERNAME}/${IMAGE_NAME}:${TAG}"

# Check if buildx is available
if ! docker buildx version &> /dev/null; then
    echo "Error: docker buildx is not available. Please update Docker Desktop or install buildx."
    exit 1
fi

# Create and use a buildx builder instance for multi-platform builds
if ! docker buildx inspect multiarch-builder &> /dev/null; then
    echo "Creating multi-platform builder..."
    docker buildx create --name multiarch-builder --use --bootstrap
else
    echo "Using existing multi-platform builder..."
    docker buildx use multiarch-builder
fi

if [ "$LOCAL_ONLY" = true ]; then
    echo "Building Docker image locally (current platform only): ${FULL_IMAGE_NAME}"
    docker build -t ${FULL_IMAGE_NAME} .
    echo ""
    echo "Build complete! Image: ${FULL_IMAGE_NAME}"
    echo ""
    echo "To build for multiple platforms, run without --local-only flag"
else
    echo "Building multi-platform Docker image: ${FULL_IMAGE_NAME}"
    echo "Platforms: linux/amd64, linux/arm64"
    echo "This may take longer as it builds for multiple architectures..."
    echo ""
    
    # Build for multiple platforms
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        -t ${FULL_IMAGE_NAME} \
        --push \
        .
    
    echo ""
    echo "Multi-platform build and push complete! Image: ${FULL_IMAGE_NAME}"
    echo "This image will work on:"
    echo "  - Windows (via Docker Desktop/WSL2)"
    echo "  - Linux (x86_64 and ARM64)"
    echo "  - Mac (Intel and Apple Silicon)"
fi

