ALTER TABLE generation_jobs ADD COLUMN replicate_prediction_id VARCHAR(255);

UPDATE generation_jobs
   SET replicate_prediction_id = provider_job_id
 WHERE provider_job_id IS NOT NULL;

DROP INDEX IF EXISTS idx_generation_jobs_provider_job_id;
ALTER TABLE generation_jobs
    DROP COLUMN provider,
    DROP COLUMN provider_job_id;

CREATE INDEX idx_generation_jobs_replicate_id
    ON generation_jobs(replicate_prediction_id);
