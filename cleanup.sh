#!/bin/bash

# Cleanup script for Docker containers and ports
# Usage: ./cleanup.sh

set -e

echo "Cleaning up Docker containers and ports..."

# Stop and remove ai_pro containers
echo "Stopping ai_pro containers..."
docker-compose down 2>/dev/null || true

# Remove any remaining ai_pro containers
echo "Removing ai_pro containers..."
docker ps -a | grep -E "ai_pro" | awk '{print $1}' | xargs -r docker rm -f 2>/dev/null || true

# Kill any process using port 6333 (Qdrant)
echo "Freeing port 6333..."
lsof -ti :6333 | xargs kill -9 2>/dev/null || echo "Port 6333 is already free"

# Kill any process using port 6334 (Qdrant gRPC)
echo "Freeing port 6334..."
lsof -ti :6334 | xargs kill -9 2>/dev/null || echo "Port 6334 is already free"

# Kill any process using port 27017 (MongoDB)
echo "Freeing port 27017..."
lsof -ti :27017 | xargs kill -9 2>/dev/null || echo "Port 27017 is already free"

# Kill any process using port 8080 (App)
echo "Freeing port 8080..."
lsof -ti :8080 | xargs kill -9 2>/dev/null || echo "Port 8080 is already free"

echo ""
echo "✅ Cleanup complete!"
echo "You can now run: docker-compose up --build"

