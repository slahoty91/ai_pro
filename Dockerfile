# ---------- Build stage ----------
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder

WORKDIR /app

# Install CGO + OCR dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    pkg-config \
    tesseract-ocr \
    libtesseract-dev \
    libleptonica-dev \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -o ai_pro main.go


# ---------- Runtime stage ----------
FROM debian:bookworm-slim

WORKDIR /app

# Copy binary
COPY --from=builder /app/ai_pro .

# Copy runtime OCR + certs (NO apt here)
COPY --from=builder /usr/share/tesseract-ocr /usr/share/tesseract-ocr
COPY --from=builder /usr/lib /usr/lib
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

EXPOSE 8080
CMD ["./ai_pro"]
