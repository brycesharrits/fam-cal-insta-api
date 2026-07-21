ALTER TABLE users ADD COLUMN provider VARCHAR(20);
ALTER TABLE users ADD COLUMN provider_user_id VARCHAR(255);

UPDATE users SET provider = 'apple', provider_user_id = apple_user_id;

ALTER TABLE users ALTER COLUMN provider SET NOT NULL;
ALTER TABLE users ALTER COLUMN provider_user_id SET NOT NULL;

ALTER TABLE users DROP COLUMN apple_user_id;

CREATE UNIQUE INDEX idx_users_provider_identity ON users(provider, provider_user_id);
