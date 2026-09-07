package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"whatsnext/backend/internal/model"
)

const spaceColumns = `id,user_id,name,mode,goal,DATE_FORMAT(exam_date,'%Y-%m-%d') AS exam_date,daily_minutes,status,created_at,updated_at,deleted_at,purge_after`

type LearningSpaceRepository struct{ db *sqlx.DB }

func newRepositoryID() string {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.NewString()
	}
	return id.String()
}

func NewLearningSpaceRepository(db *sqlx.DB) *LearningSpaceRepository {
	return &LearningSpaceRepository{db: db}
}
func (r *LearningSpaceRepository) Create(ctx context.Context, s model.LearningSpace, idempotencyKey string) (model.LearningSpace, error) {
	if idempotencyKey == "" {
		_, err := r.db.ExecContext(ctx, `INSERT INTO learning_spaces(id,user_id,name,mode,goal,exam_date,daily_minutes,status) VALUES(?,?,?,?,?,?,?,?)`, s.ID, s.UserID, s.Name, s.Mode, s.Goal, s.ExamDate, s.DailyMinutes, s.Status)
		if err != nil {
			return model.LearningSpace{}, fmt.Errorf("insert learning space: %w", err)
		}
		return r.Get(ctx, s.UserID, s.ID)
	}
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.LearningSpace{}, fmt.Errorf("begin create space transaction: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO learning_spaces(id,user_id,name,mode,goal,exam_date,daily_minutes,status) VALUES(?,?,?,?,?,?,?,?)`, s.ID, s.UserID, s.Name, s.Mode, s.Goal, s.ExamDate, s.DailyMinutes, s.Status)
	if err != nil {
		return model.LearningSpace{}, fmt.Errorf("insert learning space: %w", err)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO idempotency_keys(id,user_id,operation,idempotency_key,resource_id,expires_at) VALUES(?,?,'create_space',?,?,?)`, newRepositoryID(), s.UserID, idempotencyKey, s.ID, time.Now().UTC().Add(24*time.Hour))
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			_ = tx.Rollback()
			var existingID string
			if lookupErr := r.db.GetContext(ctx, &existingID, `SELECT resource_id FROM idempotency_keys WHERE user_id=? AND operation='create_space' AND idempotency_key=?`, s.UserID, idempotencyKey); lookupErr != nil {
				return model.LearningSpace{}, fmt.Errorf("lookup idempotent space: %w", lookupErr)
			}
			return r.Get(ctx, s.UserID, existingID)
		}
		return model.LearningSpace{}, fmt.Errorf("insert idempotency key: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return model.LearningSpace{}, fmt.Errorf("commit create space: %w", err)
	}
	return r.Get(ctx, s.UserID, s.ID)
}
func (r *LearningSpaceRepository) List(ctx context.Context, userID string) ([]model.LearningSpace, error) {
	items := make([]model.LearningSpace, 0)
	err := r.db.SelectContext(ctx, &items, `SELECT `+spaceColumns+` FROM learning_spaces WHERE user_id=? AND status<>'deleted' ORDER BY created_at DESC,id DESC LIMIT 100`, userID)
	if err != nil {
		return nil, fmt.Errorf("list learning spaces: %w", err)
	}
	return items, nil
}
func (r *LearningSpaceRepository) Get(ctx context.Context, userID, id string) (model.LearningSpace, error) {
	var s model.LearningSpace
	err := r.db.GetContext(ctx, &s, `SELECT `+spaceColumns+` FROM learning_spaces WHERE id=? AND user_id=? AND status<>'deleted'`, id, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return model.LearningSpace{}, ErrNotFound
	}
	if err != nil {
		return model.LearningSpace{}, fmt.Errorf("get learning space: %w", err)
	}
	return s, nil
}
func (r *LearningSpaceRepository) Update(ctx context.Context, s model.LearningSpace) error {
	res, err := r.db.ExecContext(ctx, `UPDATE learning_spaces SET name=?,goal=?,exam_date=?,daily_minutes=?,status=? WHERE id=? AND user_id=? AND status<>'deleted'`, s.Name, s.Goal, s.ExamDate, s.DailyMinutes, s.Status, s.ID, s.UserID)
	if err != nil {
		return fmt.Errorf("update learning space: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *LearningSpaceRepository) Delete(ctx context.Context, userID, id string, now, purgeAfter time.Time) error {
	res, err := r.db.ExecContext(ctx, `UPDATE learning_spaces SET status='deleted',deleted_at=?,purge_after=? WHERE id=? AND user_id=? AND status<>'deleted'`, now, purgeAfter, id, userID)
	if err != nil {
		return fmt.Errorf("delete learning space: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
func (r *LearningSpaceRepository) CountMaterials(ctx context.Context, userID, spaceID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM learning_materials WHERE user_id=? AND learning_space_id=?`, userID, spaceID)
	if err != nil {
		return 0, fmt.Errorf("count materials: %w", err)
	}
	return count, nil
}
func (r *LearningSpaceRepository) CountNodes(ctx context.Context, userID, spaceID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM learning_nodes WHERE user_id=? AND learning_space_id=?`, userID, spaceID)
	return count, err
}
func (r *LearningSpaceRepository) CountTodayTasks(ctx context.Context, userID, spaceID string) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM plan_nodes pn JOIN learning_plans lp ON lp.id=pn.plan_id WHERE pn.user_id=? AND pn.learning_space_id=? AND lp.plan_date=?`, userID, spaceID, time.Now().UTC().Format("2006-01-02"))
	return count, err
}
