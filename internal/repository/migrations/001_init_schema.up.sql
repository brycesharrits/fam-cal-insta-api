CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    apple_user_id VARCHAR(255) UNIQUE NOT NULL,
    email         VARCHAR(255),
    token_balance INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE calendar_projects (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    year       INT NOT NULL,
    theme      VARCHAR(255) NOT NULL,
    status     VARCHAR(50) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calendar_projects_user_id ON calendar_projects(user_id);

CREATE TABLE calendar_months (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id               UUID NOT NULL REFERENCES calendar_projects(id) ON DELETE CASCADE,
    month                    INT NOT NULL CHECK (month BETWEEN 1 AND 12),
    reference_photo_asset_id VARCHAR(255),
    reference_image_url      TEXT,
    prompt                   TEXT,
    generated_image_url      TEXT,
    status                   VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, month)
);

CREATE INDEX idx_calendar_months_project_id ON calendar_months(project_id);

CREATE TABLE generation_jobs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 UUID NOT NULL REFERENCES users(id),
    calendar_id             UUID NOT NULL REFERENCES calendar_projects(id),
    month_id                UUID NOT NULL REFERENCES calendar_months(id),
    status                  VARCHAR(50) NOT NULL DEFAULT 'queued',
    replicate_prediction_id VARCHAR(255),
    result_image_url        TEXT,
    error_message           TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_generation_jobs_replicate_id ON generation_jobs(replicate_prediction_id);
CREATE INDEX idx_generation_jobs_calendar_id ON generation_jobs(calendar_id);

CREATE TABLE token_transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    amount      INT NOT NULL,
    type        VARCHAR(50) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_token_transactions_user_id ON token_transactions(user_id);

CREATE TABLE orders (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          UUID NOT NULL REFERENCES users(id),
    calendar_id      UUID NOT NULL REFERENCES calendar_projects(id),
    partner          VARCHAR(100),
    status           VARCHAR(50) NOT NULL DEFAULT 'pending',
    partner_order_id VARCHAR(255),
    tracking_url     TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
