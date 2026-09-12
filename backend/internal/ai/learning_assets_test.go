package ai

import (
	"context"
	"testing"
)

func TestMockAssetGeneratorBuildsAllThreeAssets(t *testing.T) {
	assets, err := (MockAssetGenerator{}).GenerateKnowledge(context.Background(), "通过考试", []AssetSource{{ChunkID: "chunk", Content: "# TCP 可靠传输\n序号、确认和重传保证可靠性。"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets.Nodes) != 1 || len(assets.Articles) != 1 || len(assets.Tasks) != 0 {
		t.Fatalf("incomplete assets: %+v", assets)
	}
	if assets.Nodes[0].SourceChunkID != "chunk" {
		t.Fatalf("invalid evidence or plan: %+v", assets)
	}
}

func TestMockAssetGeneratorBuildsBranchingGraph(t *testing.T) {
	sources := []AssetSource{{ChunkID: "a", Content: "A"}, {ChunkID: "b", Content: "B"}, {ChunkID: "c", Content: "C"}}
	assets, err := (MockAssetGenerator{}).GenerateKnowledge(context.Background(), "goal", sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets.Edges) != 3 {
		t.Fatalf("edges=%+v", assets.Edges)
	}
	if assets.Edges[0].From != 0 || assets.Edges[1].From != 0 || assets.Edges[2].RelationType != "related" {
		t.Fatalf("graph is not branching: %+v", assets.Edges)
	}
}

func TestMockPlanConsumesKnowledgeSources(t *testing.T) {
	plan, err := (MockAssetGenerator{}).GeneratePlan(context.Background(), PlanContext{Goal: "goal", ExamDate: "2026-09-10", DailyMinutes: 30}, []PlanSource{{NodeID: "node", Name: "TCP", ArticleBody: "summary", MaterialName: "网络课件.pdf", EstimatedMinutes: 20, ExamWeight: 0.9, MasteryScore: 25, MasteryStatus: "weak"}}, nil)
	if err != nil || len(plan.Tasks) != 1 || plan.Tasks[0].EstimatedMinutes != 20 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	if len(plan.Stages) != 1 || plan.Stages[0].Node != 0 || plan.Stages[0].Title != "攻克：TCP" {
		t.Fatalf("roadmap is not tied to source nodes: %+v", plan.Stages)
	}
	if plan.Rationale == "" {
		t.Fatal("expected a learning-state rationale")
	}
}

func TestMockPlanPrioritizesWeakHighWeightMaterialNode(t *testing.T) {
	sources := []PlanSource{
		{NodeID: "mastered", Name: "绪论", MaterialName: "教材.pdf", EstimatedMinutes: 20, ExamWeight: 0.3, MasteryScore: 95, MasteryStatus: "mastered"},
		{NodeID: "weak", Name: "TCP 拥塞控制", MaterialName: "重点课件.pptx", EstimatedMinutes: 30, ExamWeight: 0.95, MasteryScore: 20, MasteryStatus: "weak", WrongCount: 2},
	}
	plan, err := (MockAssetGenerator{}).GeneratePlan(context.Background(), PlanContext{Goal: "通过考试", DailyMinutes: 30}, sources, nil)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Stages[0].Node != 1 || plan.Stages[0].Status != "active" || plan.Tasks[0].Node != 1 {
		t.Fatalf("weak high-weight node was not prioritized: %+v", plan)
	}
}

func TestValidateKnowledgeGraphRejectsDisconnectedNodes(t *testing.T) {
	err := validateKnowledgeGraph(4, []GeneratedEdge{{From: 0, To: 1, RelationType: "related"}, {From: 2, To: 3, RelationType: "related"}})
	if err == nil {
		t.Fatal("expected disconnected graph to be rejected")
	}
}

func TestValidateKnowledgeGraphAcceptsConnectedPeerGraph(t *testing.T) {
	err := validateKnowledgeGraph(4, []GeneratedEdge{{From: 0, To: 1, RelationType: "related"}, {From: 0, To: 2, RelationType: "prerequisite"}, {From: 2, To: 3, RelationType: "related"}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestValidateKnowledgeGraphRejectsContainsRelationship(t *testing.T) {
	err := validateKnowledgeGraph(2, []GeneratedEdge{{From: 0, To: 1, RelationType: "contains"}})
	if err == nil {
		t.Fatal("expected hierarchical relationship to be rejected")
	}
}
