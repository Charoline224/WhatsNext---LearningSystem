package dto

import "whatsnext/backend/internal/model"

type ExamAnalysis struct {
	Patterns      []model.ExamPattern  `json:"patterns"`
	Questions     []model.ExamQuestion `json:"questions"`
	PaperCount    int                  `json:"paper_count"`
	AnsweredCount int                  `json:"answered_count"`
	WrongCount    int                  `json:"wrong_count"`
}
type ExamFeedbackInput struct {
	IsCorrect *bool  `json:"is_correct"`
	Note      string `json:"note"`
}
type ExamFeedbackResult struct {
	QuestionID     string `json:"question_id"`
	IsCorrect      bool   `json:"is_correct"`
	DecisionStatus string `json:"decision_status"`
}
