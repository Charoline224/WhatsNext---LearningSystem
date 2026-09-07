package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"whatsnext/backend/internal/ai"
	"whatsnext/backend/internal/materialparser"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/repository"
	"whatsnext/backend/internal/storage"
)

type MaterialProcessor struct {
	repo               *repository.MaterialRepository
	storage            *storage.MinIO
	parser             *materialparser.Parser
	chunker            *materialparser.Chunker
	embeddingModel     string
	embeddingDimension int
	examAnalyzer       ai.ExamAnalyzer
}

func NewMaterialProcessor(repo *repository.MaterialRepository, storage *storage.MinIO, parser *materialparser.Parser, chunker *materialparser.Chunker, embeddingModel string, embeddingDimension int, examAnalyzer ai.ExamAnalyzer) *MaterialProcessor {
	return &MaterialProcessor{repo: repo, storage: storage, parser: parser, chunker: chunker, embeddingModel: embeddingModel, embeddingDimension: embeddingDimension, examAnalyzer: examAnalyzer}
}
func (p *MaterialProcessor) Process(ctx context.Context, jobID string) (string, error) {
	job, claimed, err := p.repo.ClaimJob(ctx, jobID)
	if err != nil {
		return "", err
	}
	if !claimed {
		return "", nil
	}
	material, err := p.repo.GetMaterialByID(ctx, job.MaterialID)
	if err == nil {
		var size int64
		size, err = p.storage.Stat(ctx, material.ObjectKey)
		if err == nil && size != material.SizeBytes {
			err = fmt.Errorf("stored object size mismatch")
		}
	}
	var chunks []model.MaterialChunk
	if err == nil {
		object, openErr := p.storage.Open(ctx, material.ObjectKey)
		if openErr != nil {
			err = openErr
		} else {
			defer object.Close()
			var units []materialparser.SourceUnit
			units, err = p.parser.Parse(ctx, material.OriginalName, material.SizeBytes, object)
			if err == nil {
				_ = p.repo.UpdateJobProgress(ctx, job.ID, 60)
				chunks = p.chunker.Build(material, units)
				if len(chunks) == 0 {
					err = fmt.Errorf("material contains no chunks")
				}
			}
		}
	}
	if err != nil {
		message := err.Error()
		if len(message) > 1000 {
			message = message[:1000]
		}
		code := "MATERIAL_PROCESSING_FAILED"
		if strings.Contains(message, "size mismatch") {
			code = "OBJECT_SIZE_MISMATCH"
		}
		if failErr := p.repo.FailJob(ctx, job, code, message); failErr != nil {
			return "", fmt.Errorf("process error: %v; persist failure: %w", err, failErr)
		}
		return "", err
	}
	embeddingJob := model.GenerationJob{ID: newJobID(), UserID: job.UserID, LearningSpaceID: job.LearningSpaceID, MaterialID: job.MaterialID, JobType: "embed_material", Status: "queued", MaxAttempts: 3, AvailableAt: time.Now().UTC()}
	embeddings := make([]model.ChunkEmbedding, len(chunks))
	for i, chunk := range chunks {
		embeddings[i] = model.ChunkEmbedding{
			ID: newJobID(), UserID: chunk.UserID, LearningSpaceID: chunk.LearningSpaceID,
			MaterialID: chunk.MaterialID, ChunkID: chunk.ID, QdrantPointID: chunk.ID,
			EmbeddingModel: p.embeddingModel, VectorDimension: p.embeddingDimension,
			ContentHash: chunk.ContentHash, Status: "pending",
		}
	}
	questions := []ai.GeneratedExamQuestion{}
	if material.MaterialKind == "past_exam" {
		questions, err = p.examAnalyzer.AnalyzeExam(ctx, chunks)
		if err != nil {
			message := err.Error()
			if len(message) > 1000 {
				message = message[:1000]
			}
			if failErr := p.repo.FailJob(ctx, job, "EXAM_ANALYSIS_FAILED", message); failErr != nil {
				return "", fmt.Errorf("analyze exam: %v; persist failure: %w", err, failErr)
			}
			return "", err
		}
	}
	if err = p.repo.CompleteJob(ctx, job, chunks, embeddingJob, embeddings, questions); err != nil {
		return "", err
	}
	return embeddingJob.ID, nil
}

func newJobID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}
