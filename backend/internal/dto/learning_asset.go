package dto

import "whatsnext/backend/internal/model"

type KnowledgeMap struct {
	Nodes []model.LearningNode `json:"nodes"`
	Edges []model.LearningEdge `json:"edges"`
}
type KnowledgeHandbook struct {
	Articles []model.KnowledgeArticle `json:"articles"`
}
type TodayPlan struct {
	Plan   *model.LearningPlan `json:"plan"`
	Stages []model.PlanStage   `json:"stages"`
	Tasks  []model.PlanNode    `json:"tasks"`
}
type LearningAssets struct {
	KnowledgeJob *model.LearningAssetJob `json:"knowledge_job"`
	PlanJob      *model.LearningAssetJob `json:"plan_job"`
	KnowledgeMap KnowledgeMap            `json:"knowledge_map"`
	Handbook     KnowledgeHandbook       `json:"handbook"`
	TodayPlan    TodayPlan               `json:"today_plan"`
}
