package service

import (
	"context"
	"fmt"
	"strings"

	"whatsnext/backend/internal/ai"
	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/repository"
)

type ChatService struct {
	retrieval *RetrievalService
	generator ai.ChatGenerator
	repo      *repository.ChatRepository
	planner   *LearningAssetService
}

func NewChatService(retrieval *RetrievalService, generator ai.ChatGenerator, repo *repository.ChatRepository, planner *LearningAssetService) *ChatService {
	return &ChatService{retrieval: retrieval, generator: generator, repo: repo, planner: planner}
}
func (s *ChatService) Chat(ctx context.Context, userID, spaceID string, input dto.ChatInput) (dto.ChatResult, error) {
	message := strings.TrimSpace(input.Message)
	if message == "" {
		return dto.ChatResult{}, ValidationError{"message", "不能为空"}
	}
	result, err := s.retrieval.Search(ctx, userID, spaceID, dto.RetrievalSearchInput{Query: message, TopK: 6, MaterialIDs: input.MaterialIDs})
	if err != nil {
		return dto.ChatResult{}, err
	}
	sources := make([]ai.ChatSource, len(result.Items))
	for i, item := range result.Items {
		sources[i] = ai.ChatSource{MaterialName: item.MaterialName, Content: item.Content, SourceStart: item.SourceStart}
	}
	answer, err := s.generator.Answer(ctx, message, sources)
	if err != nil {
		return dto.ChatResult{}, fmt.Errorf("%w: %v", ErrAIProvider, err)
	}
	signal, err := s.repo.SaveSignal(ctx, userID, spaceID, message, answer, result.Items)
	if err != nil {
		return dto.ChatResult{}, fmt.Errorf("save chat learning signal: %w", err)
	}
	status := "pending_plan"
	if _, planErr := s.planner.GeneratePlan(ctx, userID, spaceID); planErr == nil {
		status = "replanning"
	}
	return dto.ChatResult{Answer: answer, Sources: result.Items, DecisionSignal: dto.DecisionSignal{ID: signal.ID, RelatedNodeID: signal.RelatedNodeID, Status: status}}, nil
}
