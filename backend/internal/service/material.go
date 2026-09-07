package service

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/queue"
	"whatsnext/backend/internal/repository"
	"whatsnext/backend/internal/storage"
)

type MaterialService struct {
	spaces         *repository.LearningSpaceRepository
	materials      *repository.MaterialRepository
	storage        *storage.MinIO
	queue          *queue.RedisQueue
	maxBytes       int64
	vectors        materialVectorStore
	embeddingModel string
}

type materialVectorStore interface {
	DeleteByMaterial(context.Context, string, string, string) error
}

func NewMaterialService(spaces *repository.LearningSpaceRepository, materials *repository.MaterialRepository, storage *storage.MinIO, queue *queue.RedisQueue, maxBytes int64) *MaterialService {
	return &MaterialService{spaces: spaces, materials: materials, storage: storage, queue: queue, maxBytes: maxBytes}
}
func (s *MaterialService) SetVectorStore(vectors materialVectorStore) { s.vectors = vectors }
func (s *MaterialService) SetEmbeddingModel(model string)             { s.embeddingModel = model }
func (s *MaterialService) Upload(ctx context.Context, userID, spaceID, materialKind, filename string, size int64, file materialFile) (dto.MaterialUploadResult, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.MaterialUploadResult{}, ErrNotFound
	}
	filename = strings.TrimSpace(filepath.Base(filename))
	if filename == "" || len(filename) > 255 {
		return dto.MaterialUploadResult{}, ValidationError{"file", "文件名不能为空且不能超过 255 个字符"}
	}
	if size <= 0 {
		return dto.MaterialUploadResult{}, ValidationError{"file", "文件不能为空"}
	}
	if size > s.maxBytes {
		return dto.MaterialUploadResult{}, ErrFileTooLarge
	}
	if materialKind == "" {
		materialKind = "study"
	}
	if materialKind != "study" && materialKind != "past_exam" {
		return dto.MaterialUploadResult{}, ValidationError{"material_kind", "must be study or past_exam"}
	}
	format, err := detectMaterialFormat(filename, size, file)
	if err != nil {
		return dto.MaterialUploadResult{}, err
	}
	materialID := newID()
	jobID := newID()
	objectKey := fmt.Sprintf("users/%s/spaces/%s/materials/%s%s", userID, spaceID, materialID, format.Extension)
	hash := sha256.New()
	if err := s.storage.Put(ctx, objectKey, format.MIMEType, io.TeeReader(file, hash), size); err != nil {
		return dto.MaterialUploadResult{}, err
	}
	material := model.LearningMaterial{ID: materialID, UserID: userID, LearningSpaceID: spaceID, MaterialKind: materialKind, OriginalName: filename, ObjectKey: objectKey, MIMEType: format.MIMEType, SizeBytes: size, ContentSHA256: hash.Sum(nil), Status: "queued"}
	job := model.GenerationJob{ID: jobID, UserID: userID, LearningSpaceID: spaceID, MaterialID: materialID, JobType: "process_material", Status: "queued", Progress: 0, Attempts: 0, MaxAttempts: 3, AvailableAt: time.Now().UTC()}
	if err := s.materials.CreateWithJob(ctx, material, job); err != nil {
		_ = s.storage.Delete(context.Background(), objectKey)
		return dto.MaterialUploadResult{}, err
	}
	_ = s.queue.Enqueue(ctx, jobID)
	stored, err := s.materials.GetMaterial(ctx, userID, materialID)
	if err != nil {
		return dto.MaterialUploadResult{}, err
	}
	storedJob, err := s.materials.GetJob(ctx, userID, spaceID, jobID)
	if err != nil {
		return dto.MaterialUploadResult{}, err
	}
	return dto.MaterialUploadResult{Material: stored, Job: storedJob}, nil
}
func (s *MaterialService) List(ctx context.Context, userID, spaceID string) (dto.MaterialList, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.MaterialList{}, ErrNotFound
	}
	items, err := s.materials.ListMaterials(ctx, userID, spaceID)
	if err != nil {
		return dto.MaterialList{}, err
	}
	result := make([]dto.MaterialListItem, 0, len(items))
	for _, material := range items {
		job, jobErr := s.materials.GetLatestJobForMaterial(ctx, userID, spaceID, material.ID)
		if jobErr != nil {
			return dto.MaterialList{}, jobErr
		}
		chunkCount, countErr := s.materials.CountChunks(ctx, userID, spaceID, material.ID)
		if countErr != nil {
			return dto.MaterialList{}, countErr
		}
		result = append(result, dto.MaterialListItem{Material: material, Job: job, ChunkCount: chunkCount})
	}
	return dto.MaterialList{Items: result}, nil
}
func (s *MaterialService) ListChunks(ctx context.Context, userID, spaceID, materialID string) (dto.ChunkList, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.ChunkList{}, ErrNotFound
	}
	material, err := s.materials.GetMaterial(ctx, userID, materialID)
	if err != nil || material.LearningSpaceID != spaceID {
		return dto.ChunkList{}, ErrNotFound
	}
	items, err := s.materials.ListChunks(ctx, userID, spaceID, materialID)
	if err != nil {
		return dto.ChunkList{}, err
	}
	return dto.ChunkList{Items: items}, nil
}
func (s *MaterialService) GetJob(ctx context.Context, userID, spaceID, jobID string) (model.GenerationJob, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return model.GenerationJob{}, ErrNotFound
	}
	job, err := s.materials.GetJob(ctx, userID, spaceID, jobID)
	if err == repository.ErrNotFound {
		return model.GenerationJob{}, ErrNotFound
	}
	return job, err
}

func (s *MaterialService) Get(ctx context.Context, userID, spaceID, materialID string) (dto.MaterialDetail, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.MaterialDetail{}, ErrNotFound
	}
	material, err := s.materials.GetMaterial(ctx, userID, materialID)
	if err != nil || material.LearningSpaceID != spaceID {
		return dto.MaterialDetail{}, ErrNotFound
	}
	processingJob, err := s.materials.GetLatestJobForMaterial(ctx, userID, spaceID, materialID)
	if err != nil {
		return dto.MaterialDetail{}, err
	}
	chunkCount, err := s.materials.CountChunks(ctx, userID, spaceID, materialID)
	if err != nil {
		return dto.MaterialDetail{}, err
	}
	result := dto.MaterialDetail{Material: material, ProcessingJob: processingJob, ChunkCount: chunkCount}
	embeddingJob, err := s.materials.GetLatestEmbeddingJobForMaterial(ctx, userID, spaceID, materialID)
	if err == nil {
		result.EmbeddingJob = &embeddingJob
	} else if err != repository.ErrNotFound {
		return dto.MaterialDetail{}, err
	}
	return result, nil
}

func (s *MaterialService) Retry(ctx context.Context, userID, spaceID, materialID string) (dto.MaterialRetryResult, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.MaterialRetryResult{}, ErrNotFound
	}
	material, err := s.materials.GetMaterial(ctx, userID, materialID)
	if err != nil || material.LearningSpaceID != spaceID {
		return dto.MaterialRetryResult{}, ErrNotFound
	}
	jobType := "process_material"
	if material.Status != "failed" {
		embeddingJob, jobErr := s.materials.GetLatestEmbeddingJobForMaterial(ctx, userID, spaceID, materialID)
		if jobErr != nil || embeddingJob.Status != "failed" {
			return dto.MaterialRetryResult{}, ErrConflict
		}
		jobType = "embed_material"
	}
	job := model.GenerationJob{ID: newID(), UserID: userID, LearningSpaceID: spaceID, MaterialID: materialID, JobType: jobType, Status: "queued", MaxAttempts: 3, AvailableAt: time.Now().UTC()}
	if jobType == "process_material" {
		if s.vectors != nil {
			if err = s.vectors.DeleteByMaterial(ctx, userID, spaceID, materialID); err != nil {
				return dto.MaterialRetryResult{}, fmt.Errorf("clear material vectors before retry: %w", err)
			}
		}
		err = s.materials.RetryMaterial(ctx, material, job)
	} else {
		err = s.materials.RetryEmbedding(ctx, material, job, s.embeddingModel)
	}
	if err != nil {
		if err == repository.ErrConflict {
			return dto.MaterialRetryResult{}, ErrConflict
		}
		return dto.MaterialRetryResult{}, err
	}
	_ = s.queue.Enqueue(ctx, job.ID)
	stored, err := s.materials.GetMaterial(ctx, userID, materialID)
	if err != nil {
		return dto.MaterialRetryResult{}, err
	}
	storedJob, err := s.materials.GetJob(ctx, userID, spaceID, job.ID)
	if err != nil {
		return dto.MaterialRetryResult{}, err
	}
	return dto.MaterialRetryResult{Material: stored, Job: storedJob}, nil
}

func (s *MaterialService) Delete(ctx context.Context, userID, spaceID, materialID string) error {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return ErrNotFound
	}
	material, err := s.materials.GetMaterial(ctx, userID, materialID)
	if err != nil || material.LearningSpaceID != spaceID {
		return ErrNotFound
	}
	if material.MaterialKind != "past_exam" {
		references, referenceErr := s.materials.CountKnowledgeReferences(ctx, userID, spaceID, materialID)
		if referenceErr != nil {
			return referenceErr
		}
		if references > 0 {
			return ErrConflict
		}
	}
	if s.vectors != nil {
		if err = s.vectors.DeleteByMaterial(ctx, userID, spaceID, materialID); err != nil {
			return fmt.Errorf("delete material vectors: %w", err)
		}
	}
	if err = s.storage.Delete(ctx, material.ObjectKey); err != nil {
		return err
	}
	if material.MaterialKind == "past_exam" {
		err = s.materials.DeleteExamMaterial(ctx, userID, spaceID, materialID)
	} else {
		err = s.materials.DeleteMaterial(ctx, userID, spaceID, materialID)
	}
	if err != nil {
		if err == repository.ErrNotFound {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *MaterialService) Download(ctx context.Context, userID, spaceID, materialID string) (dto.MaterialDownload, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.MaterialDownload{}, ErrNotFound
	}
	material, err := s.materials.GetMaterial(ctx, userID, materialID)
	if err != nil || material.LearningSpaceID != spaceID {
		return dto.MaterialDownload{}, ErrNotFound
	}
	expires := 5 * time.Minute
	url, err := s.storage.PresignedDownload(ctx, material.ObjectKey, material.OriginalName, expires)
	if err != nil {
		return dto.MaterialDownload{}, err
	}
	return dto.MaterialDownload{URL: url, ExpiresAt: time.Now().UTC().Add(expires)}, nil
}
