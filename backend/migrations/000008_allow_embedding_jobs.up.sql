ALTER TABLE generation_jobs DROP CHECK chk_jobs_type;
ALTER TABLE generation_jobs
    ADD CONSTRAINT chk_jobs_type CHECK (job_type IN ('process_material', 'embed_material'));
