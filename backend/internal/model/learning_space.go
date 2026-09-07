package model

import "time"

type LearningSpace struct {
	ID           string     `db:"id" json:"id"`
	UserID       string     `db:"user_id" json:"-"`
	Name         string     `db:"name" json:"name"`
	Mode         string     `db:"mode" json:"mode"`
	Goal         string     `db:"goal" json:"goal"`
	ExamDate     string     `db:"exam_date" json:"exam_date"`
	DailyMinutes int        `db:"daily_minutes" json:"daily_minutes"`
	Status       string     `db:"status" json:"status"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at" json:"-"`
	PurgeAfter   *time.Time `db:"purge_after" json:"-"`
}
