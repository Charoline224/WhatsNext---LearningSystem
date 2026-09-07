package dto

import "whatsnext/backend/internal/model"

type CreateSpaceInput struct {
	Name         string `json:"name"`
	Mode         string `json:"mode"`
	Goal         string `json:"goal"`
	ExamDate     string `json:"exam_date"`
	DailyMinutes int    `json:"daily_minutes"`
}
type UpdateSpaceInput struct {
	Name         *string `json:"name"`
	Goal         *string `json:"goal"`
	ExamDate     *string `json:"exam_date"`
	DailyMinutes *int    `json:"daily_minutes"`
	Status       *string `json:"status"`
}
type SpacePage struct {
	Items      []model.LearningSpace `json:"items"`
	NextCursor *string               `json:"next_cursor"`
}
type SpaceSummary struct {
	MaterialCount     int `json:"material_count"`
	NodeCount         int `json:"node_count"`
	MasteredNodeCount int `json:"mastered_node_count"`
	TodayTaskCount    int `json:"today_task_count"`
}
type SpaceDetail struct {
	Space   model.LearningSpace `json:"space"`
	Summary SpaceSummary        `json:"summary"`
}
