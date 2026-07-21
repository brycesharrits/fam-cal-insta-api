-- Best-effort reversal. Google-provider rows cannot round-trip and will
-- cause this migration to fail once google users exist. Dev-only rollback.
DROP INDEX IF EXISTS idx_users_provider_identity;

ALTER TABLE users ADD COLUMN apple_user_id VARCHAR(255);
UPDATE users SET apple_user_id = provider_user_id WHERE provider = 'apple';
ALTER TABLE users ALTER COLUMN apple_user_id SET NOT NULL;
ALTER TABLE users ADD CONSTRAINT users_apple_user_id_key UNIQUE (apple_user_id);

ALTER TABLE users DROP COLUMN provider;
ALTER TABLE users DROP COLUMN provider_user_id;
