CREATE TABLE learning_asset_jobs (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    learning_space_id CHAR(36) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    progress TINYINT UNSIGNED NOT NULL DEFAULT 0,
    attempts TINYINT UNSIGNED NOT NULL DEFAULT 0,
    max_attempts TINYINT UNSIGNED NOT NULL DEFAULT 3,
    available_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    started_at DATETIME(6) NULL,
    completed_at DATETIME(6) NULL,
    error_code VARCHAR(80) NULL,
    error_message VARCHAR(1000) NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_asset_jobs_dispatch (status, available_at, created_at),
    KEY idx_asset_jobs_space_created (user_id, learning_space_id, created_at),
    CONSTRAINT fk_asset_jobs_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_asset_jobs_space FOREIGN KEY (learning_space_id) REFERENCES learning_spaces (id) ON DELETE CASCADE,
    CONSTRAINT chk_asset_jobs_status CHECK (status IN ('queued', 'processing', 'succeeded', 'failed')),
    CONSTRAINT chk_asset_jobs_progress CHECK (progress BETWEEN 0 AND 100),
    CONSTRAINT chk_asset_jobs_attempts CHECK (attempts <= max_attempts)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE learning_nodes (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    learning_space_id CHAR(36) NOT NULL,
    name VARCHAR(200) NOT NULL,
    node_type VARCHAR(30) NOT NULL DEFAULT 'knowledge',
    description TEXT NOT NULL,
    exam_weight DECIMAL(4,3) NOT NULL DEFAULT 0.500,
    estimated_minutes SMALLINT UNSIGNED NOT NULL DEFAULT 30,
    source VARCHAR(20) NOT NULL DEFAULT 'ai',
    source_chunk_id CHAR(36) NULL,
    sort_order SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_nodes_space_order (user_id, learning_space_id, sort_order),
    CONSTRAINT fk_nodes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_nodes_space FOREIGN KEY (learning_space_id) REFERENCES learning_spaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_nodes_chunk FOREIGN KEY (source_chunk_id) REFERENCES material_chunks (id) ON DELETE SET NULL,
    CONSTRAINT chk_nodes_type CHECK (node_type IN ('knowledge', 'skill', 'practice', 'milestone', 'project'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE learning_edges (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    learning_space_id CHAR(36) NOT NULL,
    from_node_id CHAR(36) NOT NULL,
    to_node_id CHAR(36) NOT NULL,
    relation_type VARCHAR(30) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_edges_relation (from_node_id, to_node_id, relation_type),
    KEY idx_edges_space (user_id, learning_space_id),
    CONSTRAINT fk_edges_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_edges_space FOREIGN KEY (learning_space_id) REFERENCES learning_spaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_edges_from FOREIGN KEY (from_node_id) REFERENCES learning_nodes (id) ON DELETE CASCADE,
    CONSTRAINT fk_edges_to FOREIGN KEY (to_node_id) REFERENCES learning_nodes (id) ON DELETE CASCADE,
    CONSTRAINT chk_edges_type CHECK (relation_type IN ('prerequisite', 'related', 'contains'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE knowledge_articles (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    learning_space_id CHAR(36) NOT NULL,
    node_id CHAR(36) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body MEDIUMTEXT NOT NULL,
    source_chunk_id CHAR(36) NULL,
    sort_order SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    UNIQUE KEY uk_articles_node (node_id),
    KEY idx_articles_space_order (user_id, learning_space_id, sort_order),
    CONSTRAINT fk_articles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_articles_space FOREIGN KEY (learning_space_id) REFERENCES learning_spaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_articles_node FOREIGN KEY (node_id) REFERENCES learning_nodes (id) ON DELETE CASCADE,
    CONSTRAINT fk_articles_chunk FOREIGN KEY (source_chunk_id) REFERENCES material_chunks (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE learning_plans (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    learning_space_id CHAR(36) NOT NULL,
    plan_date DATE NOT NULL,
    title VARCHAR(255) NOT NULL,
    total_minutes SMALLINT UNSIGNED NOT NULL,
    generation_reason VARCHAR(500) NOT NULL,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_plans_space_date (user_id, learning_space_id, plan_date, created_at),
    CONSTRAINT fk_plans_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_plans_space FOREIGN KEY (learning_space_id) REFERENCES learning_spaces (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

CREATE TABLE plan_nodes (
    id CHAR(36) NOT NULL,
    user_id CHAR(36) NOT NULL,
    learning_space_id CHAR(36) NOT NULL,
    plan_id CHAR(36) NOT NULL,
    node_id CHAR(36) NOT NULL,
    task_type VARCHAR(30) NOT NULL,
    title VARCHAR(255) NOT NULL,
    estimated_minutes SMALLINT UNSIGNED NOT NULL,
    sort_order SMALLINT UNSIGNED NOT NULL DEFAULT 0,
    created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    PRIMARY KEY (id),
    KEY idx_plan_nodes_order (plan_id, sort_order),
    CONSTRAINT fk_plan_nodes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_plan_nodes_space FOREIGN KEY (learning_space_id) REFERENCES learning_spaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_plan_nodes_plan FOREIGN KEY (plan_id) REFERENCES learning_plans (id) ON DELETE CASCADE,
    CONSTRAINT fk_plan_nodes_node FOREIGN KEY (node_id) REFERENCES learning_nodes (id) ON DELETE CASCADE,
    CONSTRAINT chk_plan_nodes_type CHECK (task_type IN ('learn', 'review', 'practice', 'test'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
