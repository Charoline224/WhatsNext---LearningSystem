DELETE FROM generation_jobs WHERE job_type = 'embed_material';
ALTER TABLE generation_jobs DROP CHECK chk_jobs_type;
ALTER TABLE generation_jobs
    ADD CONSTRAINT chk_jobs_type CHECK (job_type IN ('process_material'));
