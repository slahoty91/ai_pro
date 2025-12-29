#!/bin/bash

# Quick local build script (single platform, faster)
# Usage: ./docker-build-local.sh [your-dockerhub-username] [tag]

set -e

DOCKER_USERNAME=${1:-${DOCKER_USERNAME:-"your-username"}}
IMAGE_NAME="ai-pro"
TAG=${2:-"latest"}
FULL_IMAGE_NAME="${DOCKER_USERNAME}/${IMAGE_NAME}:${TAG}"

echo "Building Docker image locally: ${FULL_IMAGE_NAME}"
docker build -t ${FULL_IMAGE_NAME} .

echo ""
echo "Build complete! Image: ${FULL_IMAGE_NAME}"
echo ""
echo "Note: This is a single-platform build for your current architecture."
echo "For multi-platform builds (Windows/Linux/Mac), use: ./docker-build.sh"
echo ""
read -p "Do you want to push to Docker Hub now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Pushing ${FULL_IMAGE_NAME} to Docker Hub..."
    docker push ${FULL_IMAGE_NAME}
    echo "Push complete!"
    echo ""
    echo "Note: This single-platform image may not work on all systems."
    echo "For cross-platform support, use: ./docker-build.sh"
fi

