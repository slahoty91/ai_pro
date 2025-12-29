package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Document struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	DocID     int64              `json:"doc_id" bson:"doc_id"`
	Title     string             `json:"title" bson:"title"`
	Content   string             `json:"content" bson:"content"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

type Chunks struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	DocID     string             `json:"doc_id" bson:"doc_id"`
	Index     int64              `json:"index" bson:"index"`
	Text      string             `json:"text" bson:"text"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

type Embeddings struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	ChunkID   string             `json:"chunk_id" bson:"chunk_id"`
	Vector    []float32          `json:"vector" bson:"vector"`
	DocID     string             `json:"doc_id" bson:"doc_id"`
	Index     int64              `json:"index" bson:"index"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}
