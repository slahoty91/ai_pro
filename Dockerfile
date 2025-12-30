# ---------- Build stage ----------
FROM --platform=$BUILDPLATFORM golang:1.24-bookworm AS builder

WORKDIR /app

RUN apt-get update && apt-get install -y \
    ca-certificates \
    pkg-config \
    tesseract-ocr \
    libtesseract-dev \
    libleptonica-dev \
    poppler-utils \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1
RUN go build -o ai_pro main.go


# ---------- Runtime stage ----------
FROM debian:bookworm-slim

WORKDIR /app

# App binary
COPY --from=builder /app/ai_pro .

# OCR + PDF runtime deps (NO apt here)
COPY --from=builder /usr/bin/pdftoppm /usr/bin/pdftoppm
COPY --from=builder /usr/lib /usr/lib
COPY --from=builder /usr/share/tesseract-ocr /usr/share/tesseract-ocr
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

EXPOSE 8080
CMD ["./ai_pro"]
