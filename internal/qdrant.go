package internal

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	qdrant "github.com/qdrant/go-client/qdrant"
)

// EnsureCollectionExists ensures the chunk_vectors collection exists in Qdrant.
// It attempts to create the collection if it doesn't exist, and ignores errors
// if the collection already exists.
func EnsureCollectionExists(client *qdrant.Client) error {
	collectionName := "chunk_vectors"

	err := client.CreateCollection(context.Background(), &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     1536,
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})

	if err != nil {
		// Check if error indicates collection already exists
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "already exists") ||
			strings.Contains(errMsg, "already exist") ||
			strings.Contains(errMsg, "collection with name") ||
			strings.Contains(errMsg, "duplicate") {
			// Collection already exists, which is fine
			return nil
		}
		// Some other error occurred during creation
		return fmt.Errorf("failed to create collection '%s': %w", collectionName, err)
	}

	return nil
}

func InitQdrant() (*qdrant.Client, error) {
	fmt.Println("Initializing Qdrant")
	host := os.Getenv("QDRANT_HOST")
	if host == "" {
		host = "localhost"
	}
	port := 6334 // gRPC port (6333 is REST API)
	if portStr := os.Getenv("QDRANT_PORT"); portStr != "" {
		if parsedPort, err := strconv.Atoi(portStr); err == nil {
			port = parsedPort
		}
	}
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	// Ensure the collection exists
	if err := EnsureCollectionExists(client); err != nil {
		return nil, fmt.Errorf("failed to ensure collection exists: %w", err)
	}

	fmt.Println("Qdrant collection 'chunk_vectors' is ready")
	return client, nil
}
