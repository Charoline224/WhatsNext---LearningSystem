package service

import (
	"context"
	"errors"
	"testing"

	"whatsnext/backend/internal/model"
	"whatsnext/backend/internal/repository"
)

func TestIndexStatusReportsPartialProgress(t *testing.T) {
	materials := &fakeIndexMaterials{
		material: model.LearningMaterial{ID: "material", LearningSpaceID: "space"},
		job:      model.GenerationJob{ID: "job", Status: "processing", JobType: "embed_material"},
	}
	records := &fakeIndexRecords{counts: repository.MaterialIndexCounts{Total: 4, Pending: 2, Indexed: 2}}
	svc := NewIndexStatusService(&fakeRetrievalSpaces{}, materials, records, "text-embedding-v4")

	result, err := svc.Get(context.Background(), "user", "space", "material")
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "indexing" || result.Progress != 50 || result.Job == nil || result.Job.ID != "job" {
		t.Fatalf("unexpected index status: %+v", result)
	}
	if records.userID != "user" || records.spaceID != "space" || records.model != "text-embedding-v4" {
		t.Fatalf("unsafe index query: %+v", records)
	}
}

func TestIndexStatusHidesMaterialFromAnotherSpace(t *testing.T) {
	materials := &fakeIndexMaterials{material: model.LearningMaterial{ID: "material", LearningSpaceID: "other"}}
	svc := NewIndexStatusService(&fakeRetrievalSpaces{}, materials, &fakeIndexRecords{}, "model")
	_, err := svc.Get(context.Background(), "user", "space", "material")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("error=%v", err)
	}
}

func TestIndexStatusTerminalStates(t *testing.T) {
	tests := []struct {
		name   string
		counts repository.MaterialIndexCounts
		want   string
	}{
		{"not started", repository.MaterialIndexCounts{}, "not_started"},
		{"pending", repository.MaterialIndexCounts{Total: 2, Pending: 2}, "pending"},
		{"indexed", repository.MaterialIndexCounts{Total: 2, Indexed: 2}, "indexed"},
		{"failed", repository.MaterialIndexCounts{Total: 2, Failed: 2}, "failed"},
		{"partial", repository.MaterialIndexCounts{Total: 3, Indexed: 1, Failed: 2}, "partial"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexStatus(tt.counts, "succeeded"); got != tt.want {
				t.Fatalf("status=%s want=%s", got, tt.want)
			}
		})
	}
}

type fakeIndexMaterials struct {
	material model.LearningMaterial
	job      model.GenerationJob
	jobErr   error
}

func (f *fakeIndexMaterials) GetMaterial(context.Context, string, string) (model.LearningMaterial, error) {
	return f.material, nil
}
func (f *fakeIndexMaterials) GetLatestEmbeddingJobForMaterial(context.Context, string, string, string) (model.GenerationJob, error) {
	if f.jobErr != nil {
		return model.GenerationJob{}, f.jobErr
	}
	if f.job.ID == "" {
		return model.GenerationJob{}, repository.ErrNotFound
	}
	return f.job, nil
}

type fakeIndexRecords struct {
	counts            repository.MaterialIndexCounts
	userID, spaceID   string
	materialID, model string
}

func (f *fakeIndexRecords) CountMaterialStatus(_ context.Context, userID, spaceID, materialID, modelName string) (repository.MaterialIndexCounts, error) {
	f.userID, f.spaceID, f.materialID, f.model = userID, spaceID, materialID, modelName
	return f.counts, nil
}
