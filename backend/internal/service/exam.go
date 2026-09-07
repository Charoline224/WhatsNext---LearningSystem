package service

import (
	"context"
	"errors"
	"strings"

	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/repository"
)

type ExamService struct {
	spaces  *repository.LearningSpaceRepository
	repo    *repository.ExamRepository
	planner *LearningAssetService
}

func NewExamService(spaces *repository.LearningSpaceRepository, repo *repository.ExamRepository, planner *LearningAssetService) *ExamService {
	return &ExamService{spaces: spaces, repo: repo, planner: planner}
}
func (s *ExamService) Get(ctx context.Context, userID, spaceID string) (dto.ExamAnalysis, error) {
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.ExamAnalysis{}, ErrNotFound
	}
	return s.repo.Analysis(ctx, userID, spaceID)
}
func (s *ExamService) Feedback(ctx context.Context, userID, spaceID, questionID string, input dto.ExamFeedbackInput) (dto.ExamFeedbackResult, error) {
	if input.IsCorrect == nil {
		return dto.ExamFeedbackResult{}, ValidationError{"is_correct", "请标记正确或错误"}
	}
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.ExamFeedbackResult{}, ErrNotFound
	}
	if err := s.repo.SaveFeedback(ctx, userID, spaceID, questionID, *input.IsCorrect, strings.TrimSpace(input.Note)); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dto.ExamFeedbackResult{}, ErrNotFound
		}
		return dto.ExamFeedbackResult{}, err
	}
	status := "pending_plan"
	if _, err := s.planner.GeneratePlan(ctx, userID, spaceID); err == nil {
		status = "replanning"
	}
	return dto.ExamFeedbackResult{QuestionID: questionID, IsCorrect: *input.IsCorrect, DecisionStatus: status}, nil
}
