# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO (Tesseract OCR)
RUN apk add --no-cache \
    gcc \
    g++ \
    make \
    tesseract-ocr \
    tesseract-ocr-dev \
    leptonica \
    leptonica-dev \
    poppler-utils \
    pkgconfig \
    git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled
ENV CGO_ENABLED=1
RUN go build -o ai_pro main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    leptonica \
    poppler-utils \
    ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/ai_pro .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./ai_pro"]

