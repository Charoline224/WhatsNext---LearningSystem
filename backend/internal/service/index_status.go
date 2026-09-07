package service

import (
	"context"
	"errors"

	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/repository"
)

type indexStatusMaterialStore interface {
	GetMaterial(context.Context, string, string) (model.LearningMaterial, error)
	GetLatestEmbeddingJobForMaterial(context.Context, string, string, string) (model.GenerationJob, error)
}

type indexStatusRecordStore interface {
	CountMaterialStatus(context.Context, string, string, string, string) (repository.MaterialIndexCounts, error)
}

type IndexStatusService struct {
	spaces    retrievalSpaceStore
	materials indexStatusMaterialStore
	records   indexStatusRecordStore
	model     string
}

func NewIndexStatusService(spaces retrievalSpaceStore, materials indexStatusMaterialStore, records indexStatusRecordStore, embeddingModel string) *IndexStatusService {
	return &IndexStatusService{spaces: spaces, materials: materials, records: records, model: embeddingModel}
}

func (s *IndexStatusService) Get(ctx context.Context, userID, spaceID, materialID string) (dto.MaterialIndexStatus, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.MaterialIndexStatus{}, ErrNotFound
	}
	material, err := s.materials.GetMaterial(ctx, userID, materialID)
	if err != nil || material.LearningSpaceID != spaceID {
		return dto.MaterialIndexStatus{}, ErrNotFound
	}
	counts, err := s.records.CountMaterialStatus(ctx, userID, spaceID, materialID, s.model)
	if err != nil {
		return dto.MaterialIndexStatus{}, err
	}
	result := dto.MaterialIndexStatus{
		MaterialID: materialID, Status: indexStatus(counts, ""), EmbeddingModel: s.model,
		TotalChunks: counts.Total, PendingChunks: counts.Pending,
		IndexedChunks: counts.Indexed, FailedChunks: counts.Failed,
	}
	if counts.Total > 0 {
		result.Progress = counts.Indexed * 100 / counts.Total
	}
	job, jobErr := s.materials.GetLatestEmbeddingJobForMaterial(ctx, userID, spaceID, materialID)
	if jobErr != nil && !errors.Is(jobErr, repository.ErrNotFound) {
		return dto.MaterialIndexStatus{}, jobErr
	}
	if jobErr == nil {
		result.Job = &job
		result.Status = indexStatus(counts, job.Status)
	}
	return result, nil
}

func indexStatus(counts repository.MaterialIndexCounts, jobStatus string) string {
	if counts.Total == 0 {
		return "not_started"
	}
	if counts.Indexed == counts.Total {
		return "indexed"
	}
	if counts.Failed == counts.Total {
		return "failed"
	}
	if counts.Failed > 0 {
		return "partial"
	}
	if jobStatus == "processing" || counts.Indexed > 0 {
		return "indexing"
	}
	return "pending"
}
