#!/bin/bash

# Install Docker Compose for Mac
# This script installs docker-compose as a standalone binary

set -e

echo "Installing Docker Compose..."

# Check if running on Mac
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "Detected macOS"
    
    # Check if Homebrew is installed
    if command -v brew &> /dev/null; then
        echo "Installing docker-compose via Homebrew..."
        brew install docker-compose
    else
        echo "Homebrew not found. Installing docker-compose manually..."
        # Download docker-compose binary
        COMPOSE_VERSION="v2.24.5"
        sudo curl -L "https://github.com/docker/compose/releases/download/${COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
        sudo chmod +x /usr/local/bin/docker-compose
    fi
else
    echo "For Linux, install docker-compose:"
    echo "  sudo curl -L \"https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)\" -o /usr/local/bin/docker-compose"
    echo "  sudo chmod +x /usr/local/bin/docker-compose"
    exit 1
fi

echo ""
echo "Verifying installation..."
docker-compose --version

echo ""
echo "✅ Docker Compose installed successfully!"
echo "You can now use: docker-compose up --build"

