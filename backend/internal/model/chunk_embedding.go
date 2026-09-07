package model

import "time"

type ChunkEmbedding struct {
	ID              string     `db:"id" json:"id"`
	UserID          string     `db:"user_id" json:"-"`
	LearningSpaceID string     `db:"learning_space_id" json:"learning_space_id"`
	MaterialID      string     `db:"material_id" json:"material_id"`
	ChunkID         string     `db:"chunk_id" json:"chunk_id"`
	QdrantPointID   string     `db:"qdrant_point_id" json:"qdrant_point_id"`
	EmbeddingModel  string     `db:"embedding_model" json:"embedding_model"`
	VectorDimension int        `db:"vector_dimension" json:"vector_dimension"`
	ContentHash     []byte     `db:"content_hash" json:"-"`
	Status          string     `db:"status" json:"status"`
	FailureReason   *string    `db:"failure_reason" json:"failure_reason"`
	IndexedAt       *time.Time `db:"indexed_at" json:"indexed_at"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}
