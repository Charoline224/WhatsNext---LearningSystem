package service

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/repository"
)

type LearningSpaceService struct {
	repo *repository.LearningSpaceRepository
	now  func() time.Time
}

func NewLearningSpaceService(repo *repository.LearningSpaceRepository) *LearningSpaceService {
	return &LearningSpaceService{repo: repo, now: time.Now}
}
func (s *LearningSpaceService) Create(ctx context.Context, userID, idempotencyKey string, input dto.CreateSpaceInput) (model.LearningSpace, error) {
	if err := validateSpace(input); err != nil {
		return model.LearningSpace{}, err
	}
	if len(idempotencyKey) > 128 {
		return model.LearningSpace{}, ValidationError{"idempotency_key", "Idempotency-Key 不能超过 128 个字符"}
	}
	space := model.LearningSpace{ID: newID(), UserID: userID, Name: strings.TrimSpace(input.Name), Mode: input.Mode, Goal: strings.TrimSpace(input.Goal), ExamDate: input.ExamDate, DailyMinutes: input.DailyMinutes, Status: "active"}
	return s.repo.Create(ctx, space, idempotencyKey)
}
func (s *LearningSpaceService) List(ctx context.Context, userID string) (dto.SpacePage, error) {
	items, err := s.repo.List(ctx, userID)
	if err != nil {
		return dto.SpacePage{}, err
	}
	return dto.SpacePage{Items: items, NextCursor: nil}, nil
}
func (s *LearningSpaceService) Get(ctx context.Context, userID, id string) (dto.SpaceDetail, error) {
	space, err := s.repo.Get(ctx, userID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.SpaceDetail{}, ErrNotFound
	}
	if err != nil {
		return dto.SpaceDetail{}, err
	}
	materialCount, err := s.repo.CountMaterials(ctx, userID, id)
	if err != nil {
		return dto.SpaceDetail{}, err
	}
	nodeCount, err := s.repo.CountNodes(ctx, userID, id)
	if err != nil {
		return dto.SpaceDetail{}, err
	}
	taskCount, err := s.repo.CountTodayTasks(ctx, userID, id)
	if err != nil {
		return dto.SpaceDetail{}, err
	}
	return dto.SpaceDetail{Space: space, Summary: dto.SpaceSummary{MaterialCount: materialCount, NodeCount: nodeCount, TodayTaskCount: taskCount}}, nil
}
func (s *LearningSpaceService) Update(ctx context.Context, userID, id string, input dto.UpdateSpaceInput) (model.LearningSpace, error) {
	space, err := s.repo.Get(ctx, userID, id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.LearningSpace{}, ErrNotFound
	}
	if err != nil {
		return model.LearningSpace{}, err
	}
	if input.Name != nil {
		space.Name = strings.TrimSpace(*input.Name)
	}
	if input.Goal != nil {
		space.Goal = strings.TrimSpace(*input.Goal)
	}
	if input.ExamDate != nil {
		space.ExamDate = *input.ExamDate
	}
	if input.DailyMinutes != nil {
		space.DailyMinutes = *input.DailyMinutes
	}
	if input.Status != nil {
		space.Status = *input.Status
	}
	if err = validateSpace(dto.CreateSpaceInput{Name: space.Name, Mode: space.Mode, Goal: space.Goal, ExamDate: space.ExamDate, DailyMinutes: space.DailyMinutes}); err != nil {
		return model.LearningSpace{}, err
	}
	if space.Status != "active" && space.Status != "archived" {
		return model.LearningSpace{}, ValidationError{"status", "状态只能是 active 或 archived"}
	}
	if err = s.repo.Update(ctx, space); err != nil {
		return model.LearningSpace{}, err
	}
	return s.repo.Get(ctx, userID, id)
}
func (s *LearningSpaceService) Delete(ctx context.Context, userID, id string) error {
	now := s.now().UTC()
	err := s.repo.Delete(ctx, userID, id, now, now.Add(7*24*time.Hour))
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotFound
	}
	return err
}
func validateSpace(in dto.CreateSpaceInput) error {
	if utf8.RuneCountInString(strings.TrimSpace(in.Name)) < 1 || utf8.RuneCountInString(strings.TrimSpace(in.Name)) > 120 {
		return ValidationError{"name", "名称长度应为 1–120 个字符"}
	}
	if in.Mode != "exam" {
		return ValidationError{"mode", "MVP 仅支持 exam 模式"}
	}
	if strings.TrimSpace(in.Goal) == "" || utf8.RuneCountInString(strings.TrimSpace(in.Goal)) > 2000 {
		return ValidationError{"goal", "目标不能为空且不能超过 2000 个字符"}
	}
	date, err := time.Parse("2006-01-02", in.ExamDate)
	if err != nil {
		return ValidationError{"exam_date", "考试日期格式应为 YYYY-MM-DD"}
	}
	today, _ := time.ParseInLocation("2006-01-02", time.Now().In(time.Local).Format("2006-01-02"), time.Local)
	if date.Before(today) {
		return ValidationError{"exam_date", "考试日期不能早于今天"}
	}
	if in.DailyMinutes < 15 || in.DailyMinutes > 720 {
		return ValidationError{"daily_minutes", "每日学习时间应为 15–720 分钟"}
	}
	return nil
}
