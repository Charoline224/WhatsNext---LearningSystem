package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/queue"
	"whatsnext/backend/internal/repository"
)

type LearningAssetService struct {
	spaces *repository.LearningSpaceRepository
	repo   *repository.LearningAssetRepository
	queue  *queue.RedisQueue
}

func NewLearningAssetService(spaces *repository.LearningSpaceRepository, repo *repository.LearningAssetRepository, queue *queue.RedisQueue) *LearningAssetService {
	return &LearningAssetService{spaces: spaces, repo: repo, queue: queue}
}
func (s *LearningAssetService) Generate(ctx context.Context, userID, spaceID string) (model.LearningAssetJob, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return model.LearningAssetJob{}, ErrNotFound
	}
	count, err := s.repo.CountIndexedSources(ctx, userID, spaceID)
	if err != nil {
		return model.LearningAssetJob{}, err
	}
	if count == 0 {
		return model.LearningAssetJob{}, ErrConflict
	}
	return s.createJob(ctx, userID, spaceID, "knowledge_assets")
}

func (s *LearningAssetService) UpdateArticle(ctx context.Context, userID, spaceID, articleID string, input dto.KnowledgeArticleInput) (model.KnowledgeArticle, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return model.KnowledgeArticle{}, ErrNotFound
	}
	input.Title = strings.TrimSpace(input.Title)
	input.Body = strings.TrimSpace(input.Body)
	if input.Title == "" || input.Body == "" {
		return model.KnowledgeArticle{}, ValidationError{"article", "title and body are required"}
	}
	out, err := s.repo.UpdateArticle(ctx, userID, spaceID, articleID, input)
	if errors.Is(err, repository.ErrNotFound) {
		return out, ErrNotFound
	}
	if err == nil {
		_, _ = s.GeneratePlan(ctx, userID, spaceID)
	}
	return out, err
}

func (s *LearningAssetService) CreateNode(ctx context.Context, userID, spaceID string, input dto.KnowledgeNodeInput) (model.LearningNode, error) {
	if err := s.validateNode(ctx, userID, spaceID, &input); err != nil {
		return model.LearningNode{}, err
	}
	return s.repo.CreateNode(ctx, userID, spaceID, input)
}
func (s *LearningAssetService) UpdateNode(ctx context.Context, userID, spaceID, nodeID string, input dto.KnowledgeNodeInput) (model.LearningNode, error) {
	if err := s.validateNode(ctx, userID, spaceID, &input); err != nil {
		return model.LearningNode{}, err
	}
	n, err := s.repo.UpdateNode(ctx, userID, spaceID, nodeID, input)
	if errors.Is(err, repository.ErrNotFound) {
		return n, ErrNotFound
	}
	return n, err
}
func (s *LearningAssetService) validateNode(ctx context.Context, userID, spaceID string, input *dto.KnowledgeNodeInput) error {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return ErrNotFound
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.NodeType == "" {
		input.NodeType = "knowledge"
	}
	valid := map[string]bool{"knowledge": true, "skill": true, "practice": true, "milestone": true, "project": true}
	if input.Name == "" || !valid[input.NodeType] || input.ExamWeight < 0 || input.ExamWeight > 1 || input.EstimatedMinutes < 1 || input.EstimatedMinutes > 1440 {
		return ValidationError{"node", "invalid knowledge node"}
	}
	return nil
}
func (s *LearningAssetService) UpdateNodePosition(ctx context.Context, userID, spaceID, nodeID string, input dto.KnowledgeNodePositionInput) error {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return ErrNotFound
	}
	err := s.repo.UpdateNodePosition(ctx, userID, spaceID, nodeID, input.PositionX, input.PositionY)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
func (s *LearningAssetService) DeleteNode(ctx context.Context, userID, spaceID, nodeID string) error {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return ErrNotFound
	}
	err := s.repo.DeleteNode(ctx, userID, spaceID, nodeID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}

func (s *LearningAssetService) CreateEdge(ctx context.Context, userID, spaceID string, input dto.KnowledgeEdgeInput) (model.LearningEdge, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return model.LearningEdge{}, ErrNotFound
	}
	valid := map[string]bool{"prerequisite": true, "related": true}
	if input.FromNodeID == "" || input.ToNodeID == "" || input.FromNodeID == input.ToNodeID || !valid[input.RelationType] {
		return model.LearningEdge{}, ValidationError{"edge", "invalid knowledge relation"}
	}
	e, err := s.repo.CreateEdge(ctx, userID, spaceID, input)
	if errors.Is(err, repository.ErrNotFound) {
		return e, ErrNotFound
	}
	return e, err
}
func (s *LearningAssetService) DeleteEdge(ctx context.Context, userID, spaceID, edgeID string) error {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return ErrNotFound
	}
	err := s.repo.DeleteEdge(ctx, userID, spaceID, edgeID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
func (s *LearningAssetService) GeneratePlan(ctx context.Context, userID, spaceID string) (model.LearningAssetJob, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return model.LearningAssetJob{}, ErrNotFound
	}
	sources, err := s.repo.PlanSources(ctx, userID, spaceID)
	if err != nil {
		return model.LearningAssetJob{}, err
	}
	if len(sources) == 0 {
		return model.LearningAssetJob{}, ErrConflict
	}
	return s.createJob(ctx, userID, spaceID, "learning_plan")
}
func (s *LearningAssetService) createJob(ctx context.Context, userID, spaceID, jobType string) (model.LearningAssetJob, error) {
	job := model.LearningAssetJob{ID: newID(), UserID: userID, LearningSpaceID: spaceID, JobType: jobType, Status: "queued", MaxAttempts: 3, AvailableAt: time.Now().UTC()}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return job, ErrConflict
		}
		return job, err
	}
	_ = s.queue.Enqueue(ctx, job.ID)
	return s.repo.GetJob(ctx, userID, spaceID, job.ID)
}
func (s *LearningAssetService) Get(ctx context.Context, userID, spaceID string) (dto.LearningAssets, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.LearningAssets{}, ErrNotFound
	}
	return s.repo.Snapshot(ctx, userID, spaceID)
}
func (s *LearningAssetService) GetJob(ctx context.Context, userID, spaceID, jobID string) (model.LearningAssetJob, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return model.LearningAssetJob{}, ErrNotFound
	}
	j, err := s.repo.GetJob(ctx, userID, spaceID, jobID)
	if errors.Is(err, repository.ErrNotFound) {
		return j, ErrNotFound
	}
	return j, err
}
