ALTER TABLE generation_jobs
    ADD COLUMN provider        VARCHAR(64),
    ADD COLUMN provider_job_id VARCHAR(255);

UPDATE generation_jobs
   SET provider = 'replicate/flux',
       provider_job_id = replicate_prediction_id
 WHERE replicate_prediction_id IS NOT NULL;

DROP INDEX IF EXISTS idx_generation_jobs_replicate_id;
ALTER TABLE generation_jobs DROP COLUMN replicate_prediction_id;

CREATE INDEX idx_generation_jobs_provider_job_id
    ON generation_jobs(provider, provider_job_id);
