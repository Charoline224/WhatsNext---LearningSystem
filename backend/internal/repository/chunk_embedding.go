package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"whatsnext/backend/internal/model"
)

const chunkEmbeddingColumns = `id,user_id,learning_space_id,material_id,chunk_id,qdrant_point_id,embedding_model,vector_dimension,content_hash,status,failure_reason,indexed_at,created_at,updated_at`

type ChunkEmbeddingRepository struct{ db *sqlx.DB }

type PendingChunkEmbedding struct {
	model.ChunkEmbedding
	Content    string `db:"content"`
	ChunkIndex int    `db:"chunk_index"`
}

type MaterialIndexCounts struct {
	Total   int `db:"total"`
	Pending int `db:"pending"`
	Indexed int `db:"indexed"`
	Failed  int `db:"failed"`
}

func NewChunkEmbeddingRepository(db *sqlx.DB) *ChunkEmbeddingRepository {
	return &ChunkEmbeddingRepository{db: db}
}

func (r *ChunkEmbeddingRepository) CountMaterialStatus(ctx context.Context, userID, spaceID, materialID, embeddingModel string) (MaterialIndexCounts, error) {
	var counts MaterialIndexCounts
	err := r.db.GetContext(ctx, &counts, `SELECT COUNT(*) AS total,
        COALESCE(SUM(status='pending'),0) AS pending,
        COALESCE(SUM(status='indexed'),0) AS indexed,
        COALESCE(SUM(status='failed'),0) AS failed
        FROM chunk_embeddings
        WHERE user_id=? AND learning_space_id=? AND material_id=? AND embedding_model=?`,
		userID, spaceID, materialID, embeddingModel)
	if err != nil {
		return MaterialIndexCounts{}, fmt.Errorf("count material index status: %w", err)
	}
	return counts, nil
}

// UpsertPending records indexing intent without storing the vector in MySQL.
// Re-indexing the same chunk and model resets a failed/indexed record to pending.
func (r *ChunkEmbeddingRepository) UpsertPending(ctx context.Context, item model.ChunkEmbedding) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO chunk_embeddings
        (id,user_id,learning_space_id,material_id,chunk_id,qdrant_point_id,embedding_model,vector_dimension,content_hash,status)
        VALUES(?,?,?,?,?,?,?,?,?,'pending')
        ON DUPLICATE KEY UPDATE
          qdrant_point_id=VALUES(qdrant_point_id),vector_dimension=VALUES(vector_dimension),
          content_hash=VALUES(content_hash),status='pending',failure_reason=NULL,indexed_at=NULL`,
		item.ID, item.UserID, item.LearningSpaceID, item.MaterialID, item.ChunkID,
		item.QdrantPointID, item.EmbeddingModel, item.VectorDimension, item.ContentHash)
	if err != nil {
		return fmt.Errorf("upsert pending chunk embedding: %w", err)
	}
	return nil
}

func (r *ChunkEmbeddingRepository) GetByChunkAndModel(ctx context.Context, chunkID, embeddingModel string) (model.ChunkEmbedding, error) {
	var item model.ChunkEmbedding
	err := r.db.GetContext(ctx, &item, `SELECT `+chunkEmbeddingColumns+` FROM chunk_embeddings WHERE chunk_id=? AND embedding_model=?`, chunkID, embeddingModel)
	if errors.Is(err, sql.ErrNoRows) {
		return model.ChunkEmbedding{}, ErrNotFound
	}
	if err != nil {
		return model.ChunkEmbedding{}, fmt.Errorf("get chunk embedding: %w", err)
	}
	return item, nil
}

func (r *ChunkEmbeddingRepository) ListPending(ctx context.Context, limit int) ([]model.ChunkEmbedding, error) {
	if limit <= 0 {
		return []model.ChunkEmbedding{}, nil
	}
	items := make([]model.ChunkEmbedding, 0)
	err := r.db.SelectContext(ctx, &items, `SELECT `+chunkEmbeddingColumns+` FROM chunk_embeddings WHERE status='pending' ORDER BY created_at LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending chunk embeddings: %w", err)
	}
	return items, nil
}

func (r *ChunkEmbeddingRepository) ListPendingForMaterial(ctx context.Context, materialID, embeddingModel string) ([]PendingChunkEmbedding, error) {
	items := make([]PendingChunkEmbedding, 0)
	err := r.db.SelectContext(ctx, &items, `SELECT ce.`+strings.ReplaceAll(chunkEmbeddingColumns, ",", ",ce.")+`,mc.content,mc.chunk_index
        FROM chunk_embeddings ce JOIN material_chunks mc ON mc.id=ce.chunk_id
        WHERE ce.material_id=? AND ce.embedding_model=? AND ce.status='pending'
        ORDER BY mc.chunk_index`, materialID, embeddingModel)
	if err != nil {
		return nil, fmt.Errorf("list material pending embeddings: %w", err)
	}
	return items, nil
}

func (r *ChunkEmbeddingRepository) MarkBatchIndexed(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In(`UPDATE chunk_embeddings SET status='indexed',failure_reason=NULL,indexed_at=? WHERE id IN (?) AND status='pending'`, time.Now().UTC(), ids)
	if err != nil {
		return fmt.Errorf("build indexed batch update: %w", err)
	}
	if _, err = r.db.ExecContext(ctx, r.db.Rebind(query), args...); err != nil {
		return fmt.Errorf("mark embedding batch indexed: %w", err)
	}
	return nil
}

func (r *ChunkEmbeddingRepository) MarkMaterialFailed(ctx context.Context, materialID, embeddingModel, reason string) error {
	if len(reason) > 1000 {
		reason = reason[:1000]
	}
	_, err := r.db.ExecContext(ctx, `UPDATE chunk_embeddings SET status='failed',failure_reason=?,indexed_at=NULL WHERE material_id=? AND embedding_model=? AND status='pending'`, reason, materialID, embeddingModel)
	if err != nil {
		return fmt.Errorf("mark material embeddings failed: %w", err)
	}
	return nil
}

func (r *ChunkEmbeddingRepository) MarkIndexed(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE chunk_embeddings SET status='indexed',failure_reason=NULL,indexed_at=? WHERE id=?`, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("mark chunk embedding indexed: %w", err)
	}
	return requireAffected(result, "chunk embedding")
}

func (r *ChunkEmbeddingRepository) MarkFailed(ctx context.Context, id, reason string) error {
	if len(reason) > 1000 {
		reason = reason[:1000]
	}
	result, err := r.db.ExecContext(ctx, `UPDATE chunk_embeddings SET status='failed',failure_reason=?,indexed_at=NULL WHERE id=?`, reason, id)
	if err != nil {
		return fmt.Errorf("mark chunk embedding failed: %w", err)
	}
	return requireAffected(result, "chunk embedding")
}

func requireAffected(result sql.Result, resource string) error {
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect %s update: %w", resource, err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}
