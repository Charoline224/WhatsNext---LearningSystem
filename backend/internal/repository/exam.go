package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
	"whatsnext/backend/internal/dto"
	"whatsnext/backend/internal/model"
)

type ExamRepository struct{ db *sqlx.DB }

func NewExamRepository(db *sqlx.DB) *ExamRepository { return &ExamRepository{db: db} }
func (r *ExamRepository) Analysis(ctx context.Context, userID, spaceID string) (dto.ExamAnalysis, error) {
	out := dto.ExamAnalysis{Patterns: []model.ExamPattern{}, Questions: []model.ExamQuestion{}}
	if err := r.db.SelectContext(ctx, &out.Patterns, `SELECT id,pattern_key,title,description,tested_knowledge,common_mistakes,solving_strategy,occurrence_count,frequency_level FROM exam_question_patterns WHERE user_id=? AND learning_space_id=? ORDER BY occurrence_count DESC,title`, userID, spaceID); err != nil {
		return out, err
	}
	type patternLinkRow struct {
		PatternID string `db:"pattern_id"`
		model.ExamPatternNode
	}
	links := []patternLinkRow{}
	if err := r.db.SelectContext(ctx, &links, `SELECT l.pattern_id,l.node_id,n.name AS node_name,l.confidence,l.relation_reason FROM exam_pattern_node_links l JOIN exam_question_patterns p ON p.id=l.pattern_id JOIN learning_nodes n ON n.id=l.node_id WHERE p.user_id=? AND p.learning_space_id=? ORDER BY l.confidence DESC,n.name`, userID, spaceID); err != nil {
		return out, err
	}
	byPattern := map[string][]model.ExamPatternNode{}
	for _, link := range links {
		byPattern[link.PatternID] = append(byPattern[link.PatternID], link.ExamPatternNode)
	}
	for i := range out.Patterns {
		out.Patterns[i].RelatedNodes = byPattern[out.Patterns[i].ID]
		if out.Patterns[i].RelatedNodes == nil {
			out.Patterns[i].RelatedNodes = []model.ExamPatternNode{}
		}
	}
	if err := r.db.SelectContext(ctx, &out.Questions, `SELECT q.id,q.material_id,q.pattern_id,q.sequence_no,q.stem,q.question_type,q.source_type,q.source_start,f.is_correct,q.created_at FROM exam_questions q LEFT JOIN exam_question_feedback f ON f.question_id=q.id AND f.user_id=q.user_id WHERE q.user_id=? AND q.learning_space_id=? ORDER BY q.created_at,q.sequence_no`, userID, spaceID); err != nil {
		return out, err
	}
	if err := r.db.GetContext(ctx, &out.PaperCount, `SELECT COUNT(*) FROM learning_materials WHERE user_id=? AND learning_space_id=? AND material_kind='past_exam' AND status='ready'`, userID, spaceID); err != nil {
		return out, err
	}
	for _, q := range out.Questions {
		if q.IsCorrect != nil {
			out.AnsweredCount++
			if !*q.IsCorrect {
				out.WrongCount++
			}
		}
	}
	return out, nil
}
func (r *ExamRepository) SaveFeedback(ctx context.Context, userID, spaceID, questionID string, isCorrect bool, note string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var count int
	err = tx.GetContext(ctx, &count, `SELECT COUNT(*) FROM exam_questions WHERE id=? AND user_id=? AND learning_space_id=?`, questionID, userID, spaceID)
	if err != nil {
		return err
	}
	if count == 0 {
		return ErrNotFound
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO exam_question_feedback(id,user_id,learning_space_id,question_id,is_correct,note,incorporated_plan_id) VALUES(?,?,?,?,?,?,NULL) ON DUPLICATE KEY UPDATE is_correct=VALUES(is_correct),note=VALUES(note),incorporated_plan_id=NULL`, newRepositoryID(), userID, spaceID, questionID, isCorrect, note)
	if err != nil {
		return err
	}
	nodeIDs := []string{}
	if err = tx.SelectContext(ctx, &nodeIDs, `SELECT l.node_id FROM exam_questions q JOIN exam_pattern_node_links l ON l.pattern_id=q.pattern_id WHERE q.id=?`, questionID); err != nil {
		return err
	}
	for _, nodeID := range nodeIDs {
		var totals struct {
			Correct  int `db:"correct_count"`
			Wrong    int `db:"wrong_count"`
			Evidence int `db:"evidence_count"`
		}
		if err = tx.GetContext(ctx, &totals, `SELECT COALESCE(SUM(CASE WHEN f.is_correct THEN 1 ELSE 0 END),0) AS correct_count,COALESCE(SUM(CASE WHEN f.is_correct THEN 0 ELSE 1 END),0) AS wrong_count,COUNT(*) AS evidence_count FROM exam_question_feedback f JOIN exam_questions q ON q.id=f.question_id JOIN exam_pattern_node_links l ON l.pattern_id=q.pattern_id WHERE f.user_id=? AND f.learning_space_id=? AND l.node_id=?`, userID, spaceID, nodeID); err != nil {
			return err
		}
		score := 0.0
		if totals.Evidence > 0 {
			score = float64(totals.Correct) * 100 / float64(totals.Evidence)
		}
		status := "unassessed"
		if totals.Evidence > 0 {
			if score < 50 {
				status = "weak"
			} else if score >= 80 && totals.Evidence >= 2 {
				status = "mastered"
			} else {
				status = "learning"
			}
		}
		confidence := float64(totals.Evidence) / 5
		if confidence > 1 {
			confidence = 1
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO user_node_states(user_id,learning_space_id,node_id,mastery_score,mastery_status,correct_count,wrong_count,evidence_count,confidence,last_evaluated_at) VALUES(?,?,?,?,?,?,?,?,?,NOW(6)) ON DUPLICATE KEY UPDATE mastery_score=VALUES(mastery_score),mastery_status=VALUES(mastery_status),correct_count=VALUES(correct_count),wrong_count=VALUES(wrong_count),evidence_count=VALUES(evidence_count),confidence=VALUES(confidence),last_evaluated_at=VALUES(last_evaluated_at)`, userID, spaceID, nodeID, score, status, totals.Correct, totals.Wrong, totals.Evidence, confidence); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (r *ExamRepository) Question(ctx context.Context, userID, spaceID, questionID string) (model.ExamQuestion, error) {
	var q model.ExamQuestion
	err := r.db.GetContext(ctx, &q, `SELECT q.id,q.material_id,q.pattern_id,q.sequence_no,q.stem,q.question_type,q.source_type,q.source_start,f.is_correct,q.created_at FROM exam_questions q LEFT JOIN exam_question_feedback f ON f.question_id=q.id AND f.user_id=q.user_id WHERE q.id=? AND q.user_id=? AND q.learning_space_id=?`, questionID, userID, spaceID)
	if errors.Is(err, sql.ErrNoRows) {
		return q, ErrNotFound
	}
	return q, err
}
