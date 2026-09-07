package dto

import (
	"time"
	"whatsnext/backend/internal/model"
)

type MaterialUploadResult struct {
	Material   model.LearningMaterial `json:"material"`
	Job        model.GenerationJob    `json:"job"`
	ChunkCount int                    `json:"chunk_count"`
}
type MaterialListItem struct {
	Material   model.LearningMaterial `json:"material"`
	Job        model.GenerationJob    `json:"job"`
	ChunkCount int                    `json:"chunk_count"`
}
type ChunkList struct {
	Items []model.MaterialChunk `json:"items"`
}
type MaterialList struct {
	Items []MaterialListItem `json:"items"`
}

type MaterialDetail struct {
	Material      model.LearningMaterial `json:"material"`
	ProcessingJob model.GenerationJob    `json:"processing_job"`
	EmbeddingJob  *model.GenerationJob   `json:"embedding_job"`
	ChunkCount    int                    `json:"chunk_count"`
}

type MaterialRetryResult struct {
	Material model.LearningMaterial `json:"material"`
	Job      model.GenerationJob    `json:"job"`
}

type MaterialDownload struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// MaterialIndexStatus is the MySQL-backed view of a material's vector index.
// Qdrant remains the vector store; these counts describe durable indexing state.
type MaterialIndexStatus struct {
	MaterialID     string               `json:"material_id"`
	Status         string               `json:"status"`
	EmbeddingModel string               `json:"embedding_model"`
	TotalChunks    int                  `json:"total_chunks"`
	PendingChunks  int                  `json:"pending_chunks"`
	IndexedChunks  int                  `json:"indexed_chunks"`
	FailedChunks   int                  `json:"failed_chunks"`
	Progress       int                  `json:"progress"`
	Job            *model.GenerationJob `json:"job"`
}
