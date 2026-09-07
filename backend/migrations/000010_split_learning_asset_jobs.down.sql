ALTER TABLE learning_asset_jobs
    DROP CHECK chk_asset_jobs_type,
    DROP KEY idx_asset_jobs_space_type,
    DROP COLUMN job_type;
