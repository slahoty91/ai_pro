package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
	"github.com/slahoty/ai/internal"
	"github.com/slahoty/ai/models"
	"github.com/slahoty/ai/types"
	"github.com/slahoty/ai/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found. Using system environment variables.")
	}

	fmt.Println("Hello world")
	mongoURI := os.Getenv("MONGODB_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Connected to MongoDB")
	}
	defer client.Disconnect(context.TODO())
	var qdrantClient *qdrant.Client
	qdrantClient, err = internal.InitQdrant()
	if err != nil {
		log.Fatal(err)
	} else {
		fmt.Println("Connected to Qdrant")
	}
	defer qdrantClient.Close()
	db := client.Database("extractor_db")
	fmt.Println("Inserted a document")
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set. Please set it in your .env file or environment variables.")
	}
	openaiClient := openai.NewClient(apiKey)
	ginServer := gin.Default()
	ginServer.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "Hello world"})
	})
	ginServer.POST("/documents", func(c *gin.Context) {

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// Get and validate file_type
		fileTypeStr := c.PostForm("file_type")
		schemaStr := c.PostForm("schema")
		if fileTypeStr == "" && schemaStr == "" {
			c.JSON(400, gin.H{"error": "file_type is required. Must be 'resume' or 'prescription' or 'schema' is required."})
			return
		}
		fileType := types.FileType(fileTypeStr)
		if fileTypeStr != "" && (fileType != types.FileTypeResume && fileType != types.FileTypePrescription) {
			c.JSON(400, gin.H{"error": "file_type must be 'resume' or 'prescription'"})
			return
		}
		var (
			userSchema map[string]interface{}
			hasSchema  bool
		)
		if schemaStr != "" {
			err := json.Unmarshal([]byte(schemaStr), &userSchema)
			if err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			hasSchema = true
		}
		// Open the uploaded file
		src, err := file.Open()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		defer src.Close()

		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "upload-*"+filepath.Ext(file.Filename))
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		tempPath := tmpFile.Name()
		defer os.Remove(tempPath)

		// Copy the uploaded file content to the temporary file
		if _, err := io.Copy(tmpFile, src); err != nil {
			tmpFile.Close()
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if err := tmpFile.Close(); err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Verify temp file exists and has content
		fileInfo, err := os.Stat(tempPath)
		if err != nil {
			log.Printf("Error checking temp file: %v", err)
			c.JSON(500, gin.H{"error": "Failed to verify uploaded file"})
			return
		}
		if fileInfo.Size() == 0 {
			log.Printf("Error: Uploaded file is empty")
			c.JSON(500, gin.H{"error": "Uploaded file is empty"})
			return
		}
		log.Printf("Temp file created: %s, size: %d bytes", tempPath, fileInfo.Size())

		// fmt.Println(tempPath, "tempPath")
		content, err := utils.ExtractText(tempPath, filepath.Ext(file.Filename))
		if err != nil {
			log.Printf("Error extracting text from %s: %v", tempPath, err)
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if content == "" {
			log.Printf("Warning: Extracted content is empty for file %s", file.Filename)
			c.JSON(500, gin.H{"error": "Failed to extract text from file. The file might be corrupted or in an unsupported format."})
			return
		}
		log.Printf("Successfully extracted %d characters from %s", len(content), file.Filename)
		// fmt.Println(content, "contentttttttt")
		// Create document first (outside goroutines to avoid race conditions)
		var document models.Document
		title := c.PostForm("title")
		if title == "" {
			document.Title = file.Filename
		} else {
			document.Title = title
		}
		document.Content = content
		collection := db.Collection("documents")
		count, err := collection.CountDocuments(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		document.DocID = count + 1
		document.ID = primitive.NewObjectID()
		document.CreatedAt = time.Now()
		document.UpdatedAt = time.Now()
		res, err := collection.InsertOne(context.TODO(), document)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		fmt.Println(res.InsertedID)

		// Prepare variables for goroutines
		var (
			structuredData interface{}
			structErr      error
			embedErr       error
			chunks         []string
			workerCount    int
			activeWorkers  int32
		)

		// Use channels for error communication
		structErrChan := make(chan error, 1)
		embedErrChan := make(chan error, 1)

		pipeLineWaitGroup := sync.WaitGroup{}
		pipeLineWaitGroup.Add(2)

		// Goroutine 1: Extract structured data based on file_type
		go func() {
			defer pipeLineWaitGroup.Done()
			if hasSchema {
				data, err := internal.ExtractDataWithSchema(openaiClient, content, userSchema)
				if err != nil {
					structErr = err
				} else {
					structuredData = data
				}
			}
			switch fileType {
			case types.FileTypeResume:
				profileData, err := internal.ExtractStruredData(openaiClient, content)
				if err != nil {
					structErr = err
				} else {
					structuredData = profileData
				}
			case types.FileTypePrescription:
				prescriptionData, err := internal.ExtractPrescriptionData(openaiClient, content)
				if err != nil {
					structErr = err
				} else {
					structuredData = prescriptionData
				}
			}
			structErrChan <- structErr
		}()

		// Goroutine 2: Process embeddings
		go func() {
			defer pipeLineWaitGroup.Done()
			defer func() {
				embedErrChan <- embedErr
			}()

			chunkCollection := db.Collection("chunks")
			chunks = utils.ChunkText(document.Content, 350)

			embeddingCollection := db.Collection("embeddings")
			type EmbedingJob struct {
				ChunkText string
				Index     int
				ChunkID   string
			}
			jobs := make(chan EmbedingJob, len(chunks))
			workerCount = 10
			var wg sync.WaitGroup
			var workerUsed = make([]int32, workerCount)
			var vectorID uint64
			batchSize := 10

			// Create chunks in database first
			for i, chunk := range chunks {
				chunkObjID := primitive.NewObjectID()
				_, err := chunkCollection.InsertOne(context.TODO(), models.Chunks{
					ID:    chunkObjID,
					DocID: document.ID.Hex(),
					Index: int64(i),
					Text:  chunk,
				})
				if err != nil {
					embedErr = err
					return
				}
				chunkID := chunkObjID.Hex()
				jobs <- EmbedingJob{
					ChunkText: chunk,
					Index:     i,
					ChunkID:   chunkID,
				}
			}
			close(jobs)

			// Worker function
			worker := func(workerID int32) {
				defer wg.Done()
				for {
					batch := make([]EmbedingJob, 0, batchSize)
					for i := 0; i < batchSize; i++ {
						job, ok := <-jobs
						if !ok {
							break
						}
						batch = append(batch, job)
					}
					if len(batch) == 0 {
						return
					}
					if atomic.CompareAndSwapInt32(&workerUsed[workerID], 0, 1) {
						atomic.AddInt32(&activeWorkers, 1)
					}
					texts := make([]string, len(batch))
					for i, job := range batch {
						texts[i] = job.ChunkText
					}
					vec, err := internal.EmbetText(openaiClient, texts)
					if err != nil {
						embedErr = err
						return
					}
					for i, job := range batch {
						id := atomic.AddUint64(&vectorID, 1)
						vecItem := vec[i]
						_, err = embeddingCollection.InsertOne(context.TODO(), models.Embeddings{
							ID:      primitive.NewObjectID(),
							ChunkID: job.ChunkID,
							Vector:  vecItem,
							DocID:   document.ID.Hex(),
							Index:   int64(job.Index),
						})
						if err != nil {
							embedErr = err
							return
						}
						_, err = qdrantClient.Upsert(context.Background(), &qdrant.UpsertPoints{
							CollectionName: "chunk_vectors",
							Points: []*qdrant.PointStruct{
								{
									Id: &qdrant.PointId{
										PointIdOptions: &qdrant.PointId_Num{
											Num: id,
										},
									},
									Vectors: &qdrant.Vectors{
										VectorsOptions: &qdrant.Vectors_Vector{
											Vector: &qdrant.Vector{
												Data: vecItem,
											},
										},
									},
									Payload: map[string]*qdrant.Value{
										"chunk_id": {
											Kind: &qdrant.Value_StringValue{
												StringValue: job.ChunkID,
											},
										},
										"doc_id": {
											Kind: &qdrant.Value_StringValue{
												StringValue: document.ID.Hex(),
											},
										},
										"index": {
											Kind: &qdrant.Value_IntegerValue{
												IntegerValue: int64(job.Index),
											},
										},
									},
								},
							},
						})
						if err != nil {
							embedErr = err
							return
						}
					}
				}
			}

			wg.Add(workerCount)
			for i := 0; i < workerCount; i++ {
				go worker(int32(i))
			}
			wg.Wait()
			fmt.Println(activeWorkers, "activeWorkersactiveWorkers")
		}()

		// Wait for both goroutines to complete
		pipeLineWaitGroup.Wait()

		// Check for errors
		structErr = <-structErrChan
		embedErr = <-embedErrChan

		if structErr != nil {
			c.JSON(500, gin.H{"error": structErr.Error()})
			return
		}
		if embedErr != nil {
			c.JSON(500, gin.H{"error": embedErr.Error()})
			return
		}

		// Insert structured data (only once)
		structuredCollection := db.Collection("structured_data")
		_, err = structuredCollection.InsertOne(context.TODO(), bson.M{
			"document_id": document.ID.Hex(),
			"data":        structuredData,
			"created_at":  time.Now(),
		})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// fmt.Println(structuredData, "structuredData")

		c.JSON(200, gin.H{
			"message":            "Document created successfully",
			"document":           document,
			"structuredData":     structuredData,
			"chunks_count":       len(chunks),
			"workers_configured": workerCount,
			"workers_used":       activeWorkers,
		})
	})
	ginServer.POST("/search", func(c *gin.Context) {
		var req types.SearchRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		if req.Limit <= 0 {
			req.Limit = 5
		}
		vecs, err := internal.EmbetText(openaiClient, []string{req.Query})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if len(vecs) == 0 {
			c.JSON(500, gin.H{"error": "no embedding vector returned"})
			return
		}
		vec := vecs[0] // Unwrap embedding vector from slice
		limit := uint64(req.Limit)
		searchRes, err := qdrantClient.Query(context.Background(), &qdrant.QueryPoints{
			CollectionName: "chunk_vectors",
			Query:          qdrant.NewQueryNearest(qdrant.NewVectorInput(vec...)),
			Limit:          &limit,
			WithPayload:    qdrant.NewWithPayloadEnable(true),
		})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"results": searchRes, "query": req.Query, "limit": req.Limit})
	})
	ginServer.GET("/documents", func(c *gin.Context) {
		collection := db.Collection("documents")
		documents, err := collection.Find(context.TODO(), bson.M{})
		// fmt.Println(documents, "documents")
		var docs []models.Document
		for documents.Next(context.TODO()) {
			var doc models.Document
			if err := documents.Decode(&doc); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			docs = append(docs, doc)
		}
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, docs)
	})
	ginServer.GET("/get-document/:id", func(c *gin.Context) {
		id := c.Param("id")
		collection := db.Collection("documents")
		// Convert string ID to ObjectID
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid ID format"})
			return
		}
		var document models.Document
		err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&document)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, document)
	})
	ginServer.DELETE("/delete_by_id/:id", func(c *gin.Context) {
		collection_doc := db.Collection("documents")
		collection_chunks := db.Collection("chunks")
		collection_embeddings := db.Collection("embeddings")
		parm := c.Param("id")
		_, err := collection_doc.DeleteOne(context.TODO(), bson.M{"_id": parm})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		_, err = collection_chunks.DeleteMany(context.TODO(), bson.M{"doc_id": parm})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		_, err = collection_embeddings.DeleteMany(context.TODO(), bson.M{"doc_id": parm})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
		}
	})
	ginServer.DELETE("/delete_all_documents", func(c *gin.Context) {
		collection := db.Collection("documents")
		_, err := collection.DeleteMany(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		chunkCollection := db.Collection("chunks")
		_, err = chunkCollection.DeleteMany(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		embeddingCollection := db.Collection("embeddings")
		_, err = embeddingCollection.DeleteMany(context.TODO(), bson.M{})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"message": "All documents deleted successfully"})
	})
	ginServer.Run(":8080")
}
