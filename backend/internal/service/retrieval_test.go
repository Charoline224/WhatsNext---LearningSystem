package service

import (
	"context"
	"errors"
	"testing"

	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/rag"
	"whatsnext/backend/internal/repository"
)

func TestRetrievalSearchRefillsChunksInVectorOrder(t *testing.T) {
	spaces := &fakeRetrievalSpaces{}
	materials := &fakeRetrievalMaterials{chunks: []repository.RetrievalChunk{
		{MaterialChunk: model.MaterialChunk{ID: "c1", MaterialID: "m1", Content: "first", SourceType: "page"}, MaterialName: "book.pdf"},
		{MaterialChunk: model.MaterialChunk{ID: "c2", MaterialID: "m1", Content: "second", SourceType: "page"}, MaterialName: "book.pdf"},
	}}
	vectors := &fakeRetrievalVectors{results: []rag.SearchResult{
		{Score: 0.95, Payload: map[string]any{"chunk_id": "c2"}},
		{Score: 0.80, Payload: map[string]any{"chunk_id": "stale"}},
		{Score: 0.70, Payload: map[string]any{"chunk_id": "c1"}},
	}}
	service := NewRetrievalService(spaces, materials, fakeQueryEmbedder{}, vectors)
	result, err := service.Search(context.Background(), "user", "space", dto.RetrievalSearchInput{Query: " TCP ", MaterialIDs: []string{"m1"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 2 || result.Items[0].ChunkID != "c2" || result.Items[1].ChunkID != "c1" {
		t.Fatalf("unexpected results: %+v", result.Items)
	}
	if vectors.userID != "user" || vectors.spaceID != "space" || vectors.limit != defaultRetrievalTopK {
		t.Fatalf("unsafe vector request: %+v", vectors)
	}
	if len(materials.validated) != 1 || materials.validated[0] != "m1" {
		t.Fatalf("materials were not validated: %v", materials.validated)
	}
}

func TestRetrievalSearchRejectsUnknownMaterial(t *testing.T) {
	materials := &fakeRetrievalMaterials{validateErr: repository.ErrNotFound}
	service := NewRetrievalService(&fakeRetrievalSpaces{}, materials, fakeQueryEmbedder{}, &fakeRetrievalVectors{})
	_, err := service.Search(context.Background(), "user", "space", dto.RetrievalSearchInput{Query: "query", MaterialIDs: []string{"other"}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error=%v", err)
	}
}

func TestRetrievalSearchUnavailableWithoutProvider(t *testing.T) {
	service := NewRetrievalService(&fakeRetrievalSpaces{}, &fakeRetrievalMaterials{}, nil, nil)
	_, err := service.Search(context.Background(), "user", "space", dto.RetrievalSearchInput{Query: "query"})
	if !errors.Is(err, ErrAIUnavailable) {
		t.Fatalf("error=%v", err)
	}
}

type fakeRetrievalSpaces struct{ err error }

func (f *fakeRetrievalSpaces) Get(context.Context, string, string) (model.LearningSpace, error) {
	return model.LearningSpace{}, f.err
}

type fakeRetrievalMaterials struct {
	chunks      []repository.RetrievalChunk
	validated   []string
	validateErr error
}

func (f *fakeRetrievalMaterials) ValidateMaterials(_ context.Context, _, _ string, ids []string) error {
	f.validated = append([]string(nil), ids...)
	return f.validateErr
}
func (f *fakeRetrievalMaterials) GetRetrievalChunks(context.Context, string, string, []string) ([]repository.RetrievalChunk, error) {
	return f.chunks, nil
}

type fakeQueryEmbedder struct{}

func (fakeQueryEmbedder) Embed(context.Context, []string) ([][]float32, error) {
	return [][]float32{{1, 2}}, nil
}

type fakeRetrievalVectors struct {
	results []rag.SearchResult
	userID  string
	spaceID string
	limit   int
}

func (f *fakeRetrievalVectors) Search(_ context.Context, _ []float32, userID, spaceID string, _ []string, limit int) ([]rag.SearchResult, error) {
	f.userID, f.spaceID, f.limit = userID, spaceID, limit
	return f.results, nil
}
