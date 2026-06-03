-- Throwaway table for the Test Lab medium. Stores generated image bytes
-- directly in the DB (no S3 dependency). Delete this whole feature once
-- we commit to a provider and wire up the real flow.

CREATE TABLE test_generations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mode               VARCHAR(16) NOT NULL CHECK (mode IN ('text', 'edit')),
    prompt             TEXT NOT NULL,
    input_image_bytes  BYTEA,
    input_image_mime   VARCHAR(64),
    output_image_bytes BYTEA,
    output_image_mime  VARCHAR(64),
    status             VARCHAR(16) NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'complete', 'failed')),
    error_message      TEXT,
    duration_ms        INT,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_test_generations_user_id ON test_generations(user_id, created_at DESC);
