CREATE TABLE learning_spaces (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    name VARCHAR(120) NOT NULL,
    mode VARCHAR(20) NOT NULL,
    goal TEXT NOT NULL,
    exam_date DATE NOT NULL,
    daily_minutes SMALLINT UNSIGNED NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    deleted_at DATETIME(6) NULL,
    purge_after DATETIME(6) NULL,
    PRIMARY KEY (id),
    KEY idx_learning_spaces_user_status (user_id, status, created_at),
    KEY idx_learning_spaces_purge (purge_after),
    CONSTRAINT fk_learning_spaces_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT chk_learning_spaces_mode CHECK (mode IN ('exam', 'growth')),
    CONSTRAINT chk_learning_spaces_status CHECK (status IN ('active', 'archived', 'deleted')),
    CONSTRAINT chk_learning_spaces_daily_minutes CHECK (daily_minutes BETWEEN 15 AND 720),
    CONSTRAINT chk_learning_spaces_deletion CHECK (
        (status = 'deleted' AND deleted_at IS NOT NULL AND purge_after IS NOT NULL)
        OR (status <> 'deleted' AND deleted_at IS NULL AND purge_after IS NULL)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
