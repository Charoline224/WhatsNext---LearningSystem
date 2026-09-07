package model

import "time"

type LearningMaterial struct {
	ID              string    `db:"id" json:"id"`
	UserID          string    `db:"user_id" json:"-"`
	LearningSpaceID string    `db:"learning_space_id" json:"learning_space_id"`
	MaterialKind    string    `db:"material_kind" json:"material_kind"`
	OriginalName    string    `db:"original_name" json:"original_name"`
	ObjectKey       string    `db:"object_key" json:"-"`
	MIMEType        string    `db:"mime_type" json:"mime_type"`
	SizeBytes       int64     `db:"size_bytes" json:"size_bytes"`
	ContentSHA256   []byte    `db:"content_sha256" json:"-"`
	Status          string    `db:"status" json:"status"`
	FailureReason   *string   `db:"failure_reason" json:"failure_reason"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type GenerationJob struct {
	ID              string     `db:"id" json:"id"`
	UserID          string     `db:"user_id" json:"-"`
	LearningSpaceID string     `db:"learning_space_id" json:"learning_space_id"`
	MaterialID      string     `db:"material_id" json:"material_id"`
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
