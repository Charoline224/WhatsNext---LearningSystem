package model

import "time"

type MaterialChunk struct {
	ID              string    `db:"id" json:"id"`
	UserID          string    `db:"user_id" json:"-"`
	LearningSpaceID string    `db:"learning_space_id" json:"learning_space_id"`
	MaterialID      string    `db:"material_id" json:"material_id"`
	ChunkIndex      int       `db:"chunk_index" json:"chunk_index"`
	Content         string    `db:"content" json:"content"`
	SourceType      string    `db:"source_type" json:"source_type"`
	SourceStart     *int      `db:"source_start" json:"source_start"`
	SourceEnd       *int      `db:"source_end" json:"source_end"`
	CharCount       int       `db:"char_count" json:"char_count"`
	TokenEstimate   int       `db:"token_estimate" json:"token_estimate"`
	ContentHash     []byte    `db:"content_hash" json:"-"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
}
