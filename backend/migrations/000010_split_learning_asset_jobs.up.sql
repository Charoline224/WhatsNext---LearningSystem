ALTER TABLE learning_asset_jobs
    ADD COLUMN job_type VARCHAR(30) NOT NULL DEFAULT 'knowledge_assets' AFTER learning_space_id,
    ADD KEY idx_asset_jobs_space_type (user_id, learning_space_id, job_type, created_at),
    ADD CONSTRAINT chk_asset_jobs_type CHECK (job_type IN ('knowledge_assets', 'learning_plan'));
