package model

import "time"

type ExamPattern struct {
	ID              string            `db:"id" json:"id"`
	PatternKey      string            `db:"pattern_key" json:"pattern_key"`
	Title           string            `db:"title" json:"title"`
	Description     string            `db:"description" json:"description"`
	TestedKnowledge string            `db:"tested_knowledge" json:"tested_knowledge"`
	CommonMistakes  string            `db:"common_mistakes" json:"common_mistakes"`
	SolvingStrategy string            `db:"solving_strategy" json:"solving_strategy"`
	OccurrenceCount int               `db:"occurrence_count" json:"occurrence_count"`
	FrequencyLevel  string            `db:"frequency_level" json:"frequency_level"`
	RelatedNodes    []ExamPatternNode `db:"-" json:"related_nodes"`
}
type ExamPatternNode struct {
	NodeID         string  `db:"node_id" json:"node_id"`
	NodeName       string  `db:"node_name" json:"node_name"`
	Confidence     float64 `db:"confidence" json:"confidence"`
	RelationReason string  `db:"relation_reason" json:"relation_reason"`
}
type ExamQuestion struct {
	ID           string    `db:"id" json:"id"`
	MaterialID   string    `db:"material_id" json:"material_id"`
	PatternID    string    `db:"pattern_id" json:"pattern_id"`
	SequenceNo   int       `db:"sequence_no" json:"sequence_no"`
	Stem         string    `db:"stem" json:"stem"`
	QuestionType string    `db:"question_type" json:"question_type"`
	SourceType   string    `db:"source_type" json:"source_type"`
	SourceStart  *int      `db:"source_start" json:"source_start"`
	IsCorrect    *bool     `db:"is_correct" json:"is_correct"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}
