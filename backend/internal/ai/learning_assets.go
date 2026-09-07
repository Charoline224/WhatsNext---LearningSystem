package ai

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type AssetSource struct {
	ChunkID string `db:"chunk_id"`
	Content string `db:"content"`
}
type GeneratedNode struct {
	Name, Description, SourceChunkID string
	ExamWeight                       float64
	EstimatedMinutes                 int
}
type GeneratedEdge struct {
	From, To     int
	RelationType string
}
type GeneratedArticle struct {
	Node                       int
	Title, Body, SourceChunkID string
}
type GeneratedTask struct {
	Node             int
	TaskType, Title  string
	EstimatedMinutes int
}
type GeneratedRoadmapStage struct {
	Node                       int
	Title, Description, Status string
	EstimatedDays              int
}
type GeneratedPlan struct {
	Title, Rationale string
	Stages           []GeneratedRoadmapStage
	Tasks            []GeneratedTask
}
type GeneratedAssets struct {
	Nodes    []GeneratedNode
	Edges    []GeneratedEdge
	Articles []GeneratedArticle
	Tasks    []GeneratedTask
}
type PlanContext struct {
	Goal         string `json:"goal"`
	ExamDate     string `json:"examDate"`
	DailyMinutes int    `json:"dailyMinutes"`
}
type PlanSource struct {
	NodeID           string  `db:"node_id" json:"nodeId"`
	Name             string  `db:"name" json:"name"`
	Description      string  `db:"description" json:"description"`
	ArticleBody      string  `db:"article_body" json:"materialSummary"`
	MaterialName     string  `db:"material_name" json:"materialName"`
	EstimatedMinutes int     `db:"estimated_minutes" json:"estimatedMinutes"`
	ExamWeight       float64 `db:"exam_weight" json:"examWeight"`
	MasteryScore     float64 `db:"mastery_score" json:"masteryScore"`
	MasteryStatus    string  `db:"mastery_status" json:"masteryStatus"`
	EvidenceCount    int     `db:"evidence_count" json:"evidenceCount"`
	Confidence       float64 `db:"confidence" json:"confidence"`
	CorrectCount     int     `db:"correct_count" json:"correctCount"`
	WrongCount       int     `db:"wrong_count" json:"wrongCount"`
}
type PlanSignal struct {
	ID            string `db:"id"`
	Question      string `db:"question"`
	RelatedNodeID string `db:"related_node_id"`
	Weight        int    `db:"weight"`
}

type KnowledgeAssetGenerator interface {
	GenerateKnowledge(context.Context, string, []AssetSource) (GeneratedAssets, error)
}
type LearningPlanGenerator interface {
	GeneratePlan(context.Context, PlanContext, []PlanSource, []PlanSignal) (GeneratedPlan, error)
}

type MockAssetGenerator struct{}
type UnavailableAssetGenerator struct{}

func (UnavailableAssetGenerator) GenerateKnowledge(context.Context, string, []AssetSource) (GeneratedAssets, error) {
	return GeneratedAssets{}, fmt.Errorf("generation provider is not configured")
}
func (UnavailableAssetGenerator) GeneratePlan(context.Context, PlanContext, []PlanSource, []PlanSignal) (GeneratedPlan, error) {
	return GeneratedPlan{}, fmt.Errorf("generation provider is not configured")
}

func (MockAssetGenerator) GenerateKnowledge(ctx context.Context, goal string, sources []AssetSource) (GeneratedAssets, error) {
	if len(sources) == 0 {
		return GeneratedAssets{}, fmt.Errorf("no indexed sources")
	}
	if len(sources) > 12 {
		sources = sources[:12]
	}
	result := GeneratedAssets{}
	for i, source := range sources {
		if err := ctx.Err(); err != nil {
			return GeneratedAssets{}, err
		}
		content := strings.TrimSpace(source.Content)
		name := firstMeaningfulLine(content)
		name = truncateRunes(name, 60)
		if name == "" {
			name = fmt.Sprintf("知识点 %d", i+1)
		}
		description := truncateRunes(strings.Join(strings.Fields(content), " "), 220)
		minutes := 20
		result.Nodes = append(result.Nodes, GeneratedNode{Name: name, Description: description, SourceChunkID: source.ChunkID, ExamWeight: max(0.5, 0.95-float64(i)*0.04), EstimatedMinutes: minutes})
		result.Articles = append(result.Articles, GeneratedArticle{Node: i, Title: name, Body: fmt.Sprintf("学习目标：%s\n\n%s\n\n建议：结合原始资料完成理解、复述和自测。", goal, description), SourceChunkID: source.ChunkID})
	}
	// Build a small directed acyclic graph instead of a synthetic linear chain:
	// each concept can unlock two children, while siblings remain related.
	for i := 1; i < len(result.Nodes); i++ {
		parent := (i - 1) / 2
		result.Edges = append(result.Edges, GeneratedEdge{From: parent, To: i, RelationType: "prerequisite"})
		if i%2 == 0 {
			result.Edges = append(result.Edges, GeneratedEdge{From: i - 1, To: i, RelationType: "related"})
		}
	}
	if len(result.Nodes) > 3 {
		result.Edges = append(result.Edges, GeneratedEdge{From: 0, To: len(result.Nodes) - 1, RelationType: "contains"})
	}
	return result, nil
}

func (MockAssetGenerator) GeneratePlan(ctx context.Context, planContext PlanContext, sources []PlanSource, signals []PlanSignal) (GeneratedPlan, error) {
	if len(sources) == 0 {
		return GeneratedPlan{}, fmt.Errorf("knowledge map and handbook are required")
	}
	tasks := make([]GeneratedTask, 0)
	order := make([]int, len(sources))
	weights := make(map[string]int)
	for _, signal := range signals {
		weights[signal.RelatedNodeID] += signal.Weight
	}
	if len(sources) > 0 {
		weights[sources[0].NodeID] += weights[""]
	}
	for i := range sources {
		order[i] = i
	}
	priority := func(node PlanSource) float64 {
		weakness := max(0, 100-node.MasteryScore) / 100
		return float64(weights[node.NodeID])*2 + node.ExamWeight*weakness + float64(node.WrongCount)*0.25
	}
	sort.SliceStable(order, func(i, j int) bool { return priority(sources[order[i]]) > priority(sources[order[j]]) })
	remaining := planContext.DailyMinutes
	for position, i := range order {
		node := sources[i]
		if err := ctx.Err(); err != nil {
			return GeneratedPlan{}, err
		}
		if remaining < 15 {
			break
		}
		minutes := min(maxInt(15, node.EstimatedMinutes), remaining)
		title := "学习：" + node.Name
		if weights[node.NodeID] > 0 || (position == 0 && len(signals) > 0) {
			title = "问答重点：" + node.Name
		}
		tasks = append(tasks, GeneratedTask{Node: i, TaskType: "learn", Title: title, EstimatedMinutes: minutes})
		remaining -= minutes
	}
	stages := make([]GeneratedRoadmapStage, 0, min(4, len(order)))
	activeAssigned := false
	for _, i := range order {
		if len(stages) == cap(stages) {
			break
		}
		node := sources[i]
		status := "pending"
		if node.MasteryStatus == "mastered" {
			status = "complete"
		} else if !activeAssigned {
			status = "active"
			activeAssigned = true
		}
		material := node.MaterialName
		if strings.TrimSpace(material) == "" {
			material = "已上传资料"
		}
		description := fmt.Sprintf("结合《%s》的相关内容，当前掌握度 %.0f%%；按考试权重 %.0f%% 安排理解、练习与复盘。", material, node.MasteryScore, node.ExamWeight*100)
		stages = append(stages, GeneratedRoadmapStage{Node: i, Title: truncateRunes("攻克："+node.Name, 254), Description: description, Status: status, EstimatedDays: maxInt(1, (node.EstimatedMinutes+maxInt(planContext.DailyMinutes, 1)-1)/maxInt(planContext.DailyMinutes, 1))})
	}
	rationale := truncateRunes(fmt.Sprintf("AI 已结合 %d 个资料知识点、当前掌握度和 %d 条学习反馈，按目标“%s”动态排序。", len(sources), len(signals), planContext.Goal), 499)
	return GeneratedPlan{Title: truncateRunes(planContext.Goal+" · 个性化学习计划", 254), Rationale: rationale, Stages: stages, Tasks: tasks}, nil
}

func firstMeaningfulLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(strings.TrimLeft(line, "#*- ")); line != "" {
			return line
		}
	}
	return ""
}
func truncateRunes(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	return string([]rune(value)[:limit]) + "…"
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
