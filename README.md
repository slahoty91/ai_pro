# AI Pro - Document Extraction and Search

A Go-based application for extracting text from documents (PDFs, images), processing them with OpenAI embeddings, and storing them in MongoDB and Qdrant for semantic search.

## Quick Start

### Prerequisites

- Docker installed
- Docker Compose installed (see below if not available)
- OpenAI API key

**Installing Docker Compose:**

If you get `command not found: docker-compose`, install it:

**On Mac (using Homebrew):**
```bash
brew install docker-compose
```

**On Mac (manual install):**
```bash
chmod +x install-compose.sh
./install-compose.sh
```

**On Linux:**
```bash
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

**Note:** Modern Docker Desktop includes `docker compose` (space) as a plugin. If that works, use `docker compose` instead of `docker-compose`.

### One-Command Setup

1. Set your OpenAI API key as an environment variable:
   ```bash
   export OPENAI_API_KEY=your_api_key_here
   ```

2. Start all services (MongoDB, Qdrant, and the app) with one command:
   ```bash
   docker-compose up --build
   ```
   
   **Alternative (if you have Docker Compose plugin):**
   ```bash
   docker compose up --build
   ```

That's it! The application will be available at `http://localhost:8080`

### What Gets Started

- **MongoDB** - Running on port 27017 (database: `extractor_db`)
- **Qdrant** - Vector database on ports 6333 (REST) and 6334 (gRPC)
- **App** - Go application on port 8080

All services are connected via a Docker network and will automatically wait for dependencies to be healthy before starting.

### Running in Background

To run in detached mode:
```bash
docker-compose up -d --build
```

### Stopping Services

```bash
docker-compose down
```

To also remove volumes (deletes all data):
```bash
docker-compose down -v
```

### Troubleshooting

**Port already in use errors:**

If you get errors like "port is already allocated", run the cleanup script:

```bash
./cleanup.sh
```

This will:
- Stop all ai_pro containers
- Free up ports (6333, 6334, 27017, 8080)
- Clean up any stuck containers

Then try starting again:
```bash
docker-compose up --build
```

**Qdrant container unhealthy:**

If Qdrant fails to start, check logs:
```bash
docker-compose logs qdrant
```

The healthcheck allows 10 seconds for Qdrant to start. If it's still failing, you can temporarily remove the healthcheck condition from `depends_on` in docker-compose.yml.

### Viewing Logs

```bash
docker-compose logs -f
```

Or for a specific service:
```bash
docker-compose logs -f app
docker-compose logs -f mongodb
docker-compose logs -f qdrant
```

## API Endpoints

- `GET /` - Health check
- `POST /documents` - Upload and process a document
- `GET /documents` - List all documents
- `GET /get-document/:id` - Get a specific document
- `POST /search` - Semantic search across documents
- `DELETE /delete_by_id/:id` - Delete a document by ID
- `DELETE /delete_all_documents` - Delete all documents

## Environment Variables

The following environment variables can be set (defaults are used if not provided):

- `OPENAI_API_KEY` - **Required** - Your OpenAI API key
- `MONGODB_URI` - MongoDB connection string (default: `mongodb://mongodb:27017` for Docker)
- `QDRANT_HOST` - Qdrant host (default: `qdrant` for Docker)
- `QDRANT_PORT` - Qdrant port (default: `6334`)

## Data Persistence

Data is persisted in Docker volumes:
- `mongodb_data` - MongoDB database files
- `qdrant_data` - Qdrant vector database files

These volumes persist even when containers are stopped, so your data remains safe.

## Building and Pushing Docker Image

### Multi-Platform Build (Recommended)

The image is built for **multiple platforms** to work on Windows, Linux, and Mac:

- **linux/amd64** - Windows (Docker Desktop/WSL2), Linux (x86_64), Mac (Intel)
- **linux/arm64** - Linux (ARM64), Mac (Apple Silicon/M1/M2/M3)

**Build and push multi-platform image:**

```bash
chmod +x docker-build.sh
./docker-build.sh your-username
```

This automatically:
- Builds for both platforms
- Pushes to Docker Hub
- Works on all systems (Windows, Linux, Mac)

**Prerequisites for multi-platform builds:**
- Docker Desktop (Mac/Windows) or Docker with buildx (Linux)
- Buildx is included in Docker Desktop by default
- For Linux, install buildx: `docker buildx install`

**First-time setup (if needed):**
```bash
# Create buildx builder (script does this automatically)
docker buildx create --name multiarch-builder --use --bootstrap
```

### Single-Platform Build (Faster, Local Only)

For quick local testing (current platform only):

```bash
chmod +x docker-build-local.sh
./docker-build-local.sh your-username
```

Or manually:

```bash
docker build -t your-username/ai-pro:latest .
```

**Note:** Single-platform images may not work on all systems. Use multi-platform build for distribution.

### Pull and Run Pre-built Image

If you have a pre-built image on Docker Hub, you can use it instead of building:

1. **Pull the image:**
   ```bash
   docker pull your-username/ai-pro:latest
   ```

2. **Run with docker-compose using pre-built image:**
   ```bash
   export DOCKER_IMAGE=your-username/ai-pro:latest
   export OPENAI_API_KEY=your_api_key_here
   docker-compose -f docker-compose.prod.yml up
   ```

Or set the image directly in `docker-compose.prod.yml`:
```yaml
app:
  image: your-username/ai-pro:latest  # Replace with your image
```

### Image Tags

You can tag different versions with multi-platform builds:

```bash
# Build and push version tag
./docker-build.sh your-username v1.0.0

# Build and push latest tag
./docker-build.sh your-username latest
```

### Platform Compatibility

The multi-platform image works on:
- ✅ **Windows** - Via Docker Desktop or WSL2
- ✅ **Linux** - x86_64 and ARM64
- ✅ **Mac** - Intel and Apple Silicon (M1/M2/M3)

Docker automatically selects the correct architecture when you pull the image.

