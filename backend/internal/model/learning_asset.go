package model

import "time"

type LearningAssetJob struct {
	ID              string     `db:"id" json:"id"`
	UserID          string     `db:"user_id" json:"-"`
	LearningSpaceID string     `db:"learning_space_id" json:"learning_space_id"`
	JobType         string     `db:"job_type" json:"job_type"`
	Status          string     `db:"status" json:"status"`
	Progress        int        `db:"progress" json:"progress"`
	Attempts        int        `db:"attempts" json:"attempts"`
	MaxAttempts     int        `db:"max_attempts" json:"max_attempts"`
	AvailableAt     time.Time  `db:"available_at" json:"-"`
	StartedAt       *time.Time `db:"started_at" json:"started_at"`
	CompletedAt     *time.Time `db:"completed_at" json:"completed_at"`
	ErrorCode       *string    `db:"error_code" json:"error_code"`
	ErrorMessage    *string    `db:"error_message" json:"error_message"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at" json:"updated_at"`
}

type LearningNode struct {
	ID                string   `db:"id" json:"id"`
	LearningSpaceID   string   `db:"learning_space_id" json:"learning_space_id"`
	Name              string   `db:"name" json:"name"`
	NodeType          string   `db:"node_type" json:"node_type"`
	Description       string   `db:"description" json:"description"`
	ExamWeight        float64  `db:"exam_weight" json:"exam_weight"`
	EstimatedMinutes  int      `db:"estimated_minutes" json:"estimated_minutes"`
	SourceChunkID     *string  `db:"source_chunk_id" json:"source_chunk_id"`
	SortOrder         int      `db:"sort_order" json:"sort_order"`
	PositionX         *float64 `db:"position_x" json:"position_x"`
	PositionY         *float64 `db:"position_y" json:"position_y"`
	UserEdited        bool     `db:"user_edited" json:"user_edited"`
	MasteryScore      float64  `db:"mastery_score" json:"mastery_score"`
	MasteryStatus     string   `db:"mastery_status" json:"mastery_status"`
	EvidenceCount     int      `db:"evidence_count" json:"evidence_count"`
	MasteryConfidence float64  `db:"mastery_confidence" json:"mastery_confidence"`
}
type LearningEdge struct {
	ID           string `db:"id" json:"id"`
	FromNodeID   string `db:"from_node_id" json:"from_node_id"`
	ToNodeID     string `db:"to_node_id" json:"to_node_id"`
	RelationType string `db:"relation_type" json:"relation_type"`
	UserEdited   bool   `db:"user_edited" json:"user_edited"`
}
type KnowledgeArticle struct {
	ID            string  `db:"id" json:"id"`
	NodeID        string  `db:"node_id" json:"node_id"`
	Title         string  `db:"title" json:"title"`
	Body          string  `db:"body" json:"body"`
	SourceChunkID *string `db:"source_chunk_id" json:"source_chunk_id"`
	SortOrder     int     `db:"sort_order" json:"sort_order"`
	UserEdited    bool    `db:"user_edited" json:"user_edited"`
}
type LearningPlan struct {
	ID               string    `db:"id" json:"id"`
	PlanDate         string    `db:"plan_date" json:"plan_date"`
	Title            string    `db:"title" json:"title"`
	TotalMinutes     int       `db:"total_minutes" json:"total_minutes"`
	GenerationReason string    `db:"generation_reason" json:"generation_reason"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}
type PlanStage struct {
	ID            string `db:"id" json:"id"`
	PlanID        string `db:"plan_id" json:"plan_id"`
	FocusNodeID   string `db:"focus_node_id" json:"focus_node_id"`
	Title         string `db:"title" json:"title"`
	Description   string `db:"description" json:"description"`
	Status        string `db:"stage_status" json:"status"`
	EstimatedDays int    `db:"estimated_days" json:"estimated_days"`
	SortOrder     int    `db:"sort_order" json:"sort_order"`
}
type PlanNode struct {
	ID               string `db:"id" json:"id"`
	PlanID           string `db:"plan_id" json:"plan_id"`
	NodeID           string `db:"node_id" json:"node_id"`
	TaskType         string `db:"task_type" json:"task_type"`
	Title            string `db:"title" json:"title"`
	EstimatedMinutes int    `db:"estimated_minutes" json:"estimated_minutes"`
	SortOrder        int    `db:"sort_order" json:"sort_order"`
}
