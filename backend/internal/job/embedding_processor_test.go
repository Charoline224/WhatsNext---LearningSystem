package job

import (
	"context"
	"testing"

	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/rag"
	"whatsnext/backend/internal/repository"
)

func TestEmbeddingProcessorIndexesInBatches(t *testing.T) {
	jobStore := &fakeEmbeddingJobs{job: model.GenerationJob{ID: "job", MaterialID: "material", JobType: "embed_material", Status: "queued", Attempts: 1, MaxAttempts: 3}}
	recordStore := &fakeEmbeddingRecords{items: []repository.PendingChunkEmbedding{
		pendingEmbedding("one", 0), pendingEmbedding("two", 1), pendingEmbedding("three", 2),
	}}
	embedder := &fakeEmbedder{}
	vectors := &fakeVectorStore{}
	processor := NewEmbeddingProcessor(jobStore, recordStore, embedder, vectors, "model", 2)

	if err := processor.Process(context.Background(), "job"); err != nil {
		t.Fatal(err)
	}
	if embedder.calls != 2 || len(vectors.batches) != 2 {
		t.Fatalf("embed calls=%d vector batches=%d", embedder.calls, len(vectors.batches))
	}
	if len(recordStore.indexedBatches) != 2 || len(recordStore.indexedBatches[0]) != 2 || len(recordStore.indexedBatches[1]) != 1 {
		t.Fatalf("indexed batches=%v", recordStore.indexedBatches)
	}
	if !jobStore.completed {
		t.Fatal("embedding job was not completed")
	}
}

func pendingEmbedding(id string, index int) repository.PendingChunkEmbedding {
	return repository.PendingChunkEmbedding{
		ChunkEmbedding: model.ChunkEmbedding{ID: id, UserID: "user", LearningSpaceID: "space", MaterialID: "material", ChunkID: "chunk-" + id, QdrantPointID: "point-" + id},
		Content:        id, ChunkIndex: index,
	}
}

type fakeEmbeddingJobs struct {
	job       model.GenerationJob
	completed bool
}

func (f *fakeEmbeddingJobs) ClaimJob(context.Context, string) (model.GenerationJob, bool, error) {
	return f.job, true, nil
}
func (*fakeEmbeddingJobs) UpdateJobProgress(context.Context, string, int) error { return nil }
func (*fakeEmbeddingJobs) FailJob(context.Context, model.GenerationJob, string, string) error {
	return nil
}
func (f *fakeEmbeddingJobs) CompleteEmbeddingJob(context.Context, string) error {
	f.completed = true
	return nil
}

type fakeEmbeddingRecords struct {
	items          []repository.PendingChunkEmbedding
	indexedBatches [][]string
}

func (f *fakeEmbeddingRecords) ListPendingForMaterial(context.Context, string, string) ([]repository.PendingChunkEmbedding, error) {
	return f.items, nil
}
func (f *fakeEmbeddingRecords) MarkBatchIndexed(_ context.Context, ids []string) error {
	f.indexedBatches = append(f.indexedBatches, append([]string(nil), ids...))
	return nil
}
func (*fakeEmbeddingRecords) MarkMaterialFailed(context.Context, string, string, string) error {
	return nil
}

type fakeEmbedder struct{ calls int }

func (f *fakeEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	f.calls++
	result := make([][]float32, len(texts))
	for i := range result {
		result[i] = []float32{float32(i), 1}
	}
	return result, nil
}

type fakeVectorStore struct{ batches [][]rag.ChunkPoint }

func (f *fakeVectorStore) Upsert(_ context.Context, points []rag.ChunkPoint) error {
	f.batches = append(f.batches, append([]rag.ChunkPoint(nil), points...))
	return nil
}
