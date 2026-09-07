package job

import (
	"context"
	"fmt"

	"whatsnext/backend/internal/ai"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/rag"
	"whatsnext/backend/internal/repository"
)

type vectorStore interface {
	Upsert(context.Context, []rag.ChunkPoint) error
}

type embeddingJobStore interface {
	ClaimJob(context.Context, string) (model.GenerationJob, bool, error)
	UpdateJobProgress(context.Context, string, int) error
	FailJob(context.Context, model.GenerationJob, string, string) error
	CompleteEmbeddingJob(context.Context, string) error
}

type embeddingRecordStore interface {
	ListPendingForMaterial(context.Context, string, string) ([]repository.PendingChunkEmbedding, error)
	MarkBatchIndexed(context.Context, []string) error
	MarkMaterialFailed(context.Context, string, string, string) error
}

type EmbeddingProcessor struct {
	jobs       embeddingJobStore
	embeddings embeddingRecordStore
	embedder   ai.Embedder
	vectors    vectorStore
	model      string
	batchSize  int
}

func NewEmbeddingProcessor(jobs embeddingJobStore, embeddings embeddingRecordStore, embedder ai.Embedder, vectors vectorStore, model string, batchSize int) *EmbeddingProcessor {
	return &EmbeddingProcessor{jobs: jobs, embeddings: embeddings, embedder: embedder, vectors: vectors, model: model, batchSize: batchSize}
}

func (p *EmbeddingProcessor) Process(ctx context.Context, jobID string) error {
	job, claimed, err := p.jobs.ClaimJob(ctx, jobID)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	if job.JobType != "embed_material" {
		return fmt.Errorf("job %s has unsupported type %s", job.ID, job.JobType)
	}
	items, err := p.embeddings.ListPendingForMaterial(ctx, job.MaterialID, p.model)
	if err == nil {
		for start := 0; start < len(items); start += p.batchSize {
			end := start + p.batchSize
			if end > len(items) {
				end = len(items)
			}
			batch := items[start:end]
			texts := make([]string, len(batch))
			for i := range batch {
				texts[i] = batch[i].Content
			}
			var vectors [][]float32
			vectors, err = p.embedder.Embed(ctx, texts)
			if err != nil {
				break
			}
			if len(vectors) != len(batch) {
				err = fmt.Errorf("embedding provider returned %d vectors for %d chunks", len(vectors), len(batch))
				break
			}
			points := make([]rag.ChunkPoint, len(batch))
			ids := make([]string, len(batch))
			for i, item := range batch {
				points[i] = rag.ChunkPoint{
					ID: item.QdrantPointID, Vector: vectors[i], UserID: item.UserID,
					LearningSpaceID: item.LearningSpaceID, MaterialID: item.MaterialID,
					ChunkID: item.ChunkID, ChunkIndex: item.ChunkIndex,
				}
				ids[i] = item.ID
			}
			if err = p.vectors.Upsert(ctx, points); err != nil {
				break
			}
			if err = p.embeddings.MarkBatchIndexed(ctx, ids); err != nil {
				break
			}
			progress := 10 + (80 * end / len(items))
			_ = p.jobs.UpdateJobProgress(ctx, job.ID, progress)
		}
	}
	if err != nil {
		message := err.Error()
		if failErr := p.jobs.FailJob(ctx, job, "EMBEDDING_INDEX_FAILED", message); failErr != nil {
			return fmt.Errorf("embedding error: %v; persist failure: %w", err, failErr)
		}
		if job.Attempts >= job.MaxAttempts {
			if markErr := p.embeddings.MarkMaterialFailed(ctx, job.MaterialID, p.model, message); markErr != nil {
				return fmt.Errorf("embedding error: %v; mark chunks failed: %w", err, markErr)
			}
		}
		return err
	}
	return p.jobs.CompleteEmbeddingJob(ctx, job.ID)
}
