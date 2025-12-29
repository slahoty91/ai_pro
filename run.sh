#!/bin/bash

# Set CGO environment variables for Tesseract/Leptonica on macOS
# Using /opt/homebrew/include which contains symlinks to the actual header locations
export CGO_ENABLED=1
export CGO_CXXFLAGS="-I/opt/homebrew/include"
export CGO_LDFLAGS="-L/opt/homebrew/lib -ltesseract -lleptonica"

# Run the Go application
go run main.go

