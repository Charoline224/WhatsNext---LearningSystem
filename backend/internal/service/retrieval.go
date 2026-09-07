package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"whatsnext/backend/internal/ai"
	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/rag"
	"whatsnext/backend/internal/repository"
)

const (
	defaultRetrievalTopK   = 8
	maxRetrievalTopK       = 20
	maxRetrievalQueryRunes = 2000
	maxRetrievalMaterials  = 20
)

type retrievalSpaceStore interface {
	Get(context.Context, string, string) (model.LearningSpace, error)
}

type retrievalMaterialStore interface {
	ValidateMaterials(context.Context, string, string, []string) error
	GetRetrievalChunks(context.Context, string, string, []string) ([]repository.RetrievalChunk, error)
}

type retrievalVectorStore interface {
	Search(context.Context, []float32, string, string, []string, int) ([]rag.SearchResult, error)
}

type RetrievalService struct {
	spaces    retrievalSpaceStore
	materials retrievalMaterialStore
	embedder  ai.Embedder
	vectors   retrievalVectorStore
}

func NewRetrievalService(spaces retrievalSpaceStore, materials retrievalMaterialStore, embedder ai.Embedder, vectors retrievalVectorStore) *RetrievalService {
	return &RetrievalService{spaces: spaces, materials: materials, embedder: embedder, vectors: vectors}
}

func (s *RetrievalService) Search(ctx context.Context, userID, spaceID string, input dto.RetrievalSearchInput) (dto.RetrievalSearchResult, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" {
		return dto.RetrievalSearchResult{}, ValidationError{"query", "不能为空"}
	}
	if utf8.RuneCountInString(query) > maxRetrievalQueryRunes {
		return dto.RetrievalSearchResult{}, ValidationError{"query", "不能超过 2000 个字符"}
	}
	if input.TopK == 0 {
		input.TopK = defaultRetrievalTopK
	}
	if input.TopK < 1 || input.TopK > maxRetrievalTopK {
		return dto.RetrievalSearchResult{}, ValidationError{"top_k", "必须在 1 到 20 之间"}
	}
	materialIDs, valid := uniqueNonEmpty(input.MaterialIDs)
	if !valid || len(materialIDs) > maxRetrievalMaterials {
		return dto.RetrievalSearchResult{}, ValidationError{"material_ids", "最多包含 20 个非空资料 ID"}
	}
	if _, err := s.spaces.Get(ctx, userID, spaceID); err != nil {
		return dto.RetrievalSearchResult{}, ErrNotFound
	}
	if err := s.materials.ValidateMaterials(ctx, userID, spaceID, materialIDs); err != nil {
		if err == repository.ErrNotFound {
			return dto.RetrievalSearchResult{}, ErrNotFound
		}
		return dto.RetrievalSearchResult{}, err
	}
	if s.embedder == nil || s.vectors == nil {
		return dto.RetrievalSearchResult{}, ErrAIUnavailable
	}
	vectors, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return dto.RetrievalSearchResult{}, fmt.Errorf("%w: %v", ErrAIProvider, err)
	}
	if len(vectors) != 1 {
		return dto.RetrievalSearchResult{}, ErrAIUnavailable
	}
	matches, err := s.vectors.Search(ctx, vectors[0], userID, spaceID, materialIDs, input.TopK)
	if err != nil {
		return dto.RetrievalSearchResult{}, fmt.Errorf("%w: %v", ErrRetrievalUnavailable, err)
	}
	chunkIDs := make([]string, 0, len(matches))
	scores := make(map[string]float32, len(matches))
	for _, match := range matches {
		chunkID, ok := match.Payload["chunk_id"].(string)
		if !ok || chunkID == "" {
			continue
		}
		if _, exists := scores[chunkID]; !exists {
			chunkIDs = append(chunkIDs, chunkID)
			scores[chunkID] = match.Score
		}
	}
	chunks, err := s.materials.GetRetrievalChunks(ctx, userID, spaceID, chunkIDs)
	if err != nil {
		return dto.RetrievalSearchResult{}, err
	}
	byID := make(map[string]repository.RetrievalChunk, len(chunks))
	for _, chunk := range chunks {
		byID[chunk.ID] = chunk
	}
	items := make([]dto.RetrievalResult, 0, len(chunkIDs))
	for _, id := range chunkIDs {
		chunk, ok := byID[id]
		if !ok {
			continue
		}
		items = append(items, dto.RetrievalResult{
			ChunkID: chunk.ID, MaterialID: chunk.MaterialID, MaterialName: chunk.MaterialName,
			ChunkIndex: chunk.ChunkIndex, Content: chunk.Content, SourceType: chunk.SourceType,
			SourceStart: chunk.SourceStart, SourceEnd: chunk.SourceEnd, Score: scores[id],
		})
	}
	return dto.RetrievalSearchResult{Items: items}, nil
}

func uniqueNonEmpty(values []string) ([]string, bool) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, false
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, true
}
