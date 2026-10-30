-- Source payloads keep provenance and multiple/approximate locations without
-- coupling jobs to resume drafts. Internal publishing is intentionally deferred.
CREATE TABLE job_listings (
    id TEXT PRIMARY KEY,
    source TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    payload JSONB NOT NULL,
    expires_at TIMESTAMPTZ,
    fetched_at TIMESTAMPTZ NOT NULL,
    closed BOOLEAN NOT NULL DEFAULT false,
    UNIQUE (source, provider_id)
);
CREATE INDEX job_listings_active ON job_listings (source, fetched_at DESC) WHERE NOT closed;

CREATE TABLE saved_jobs (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_id TEXT NOT NULL REFERENCES job_listings(id) ON DELETE CASCADE,
    saved_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, job_id)
);

CREATE TABLE job_source_status (
    source TEXT PRIMARY KEY,
    payload JSONB NOT NULL
);

-- Shared counters survive restarts and atomically cap optional paid-provider
-- APIs at configured free-tier budgets, across users and API processes.
CREATE TABLE job_api_usage (
    provider TEXT NOT NULL,
    period TEXT NOT NULL,
    requests INTEGER NOT NULL DEFAULT 0 CHECK (requests >= 0),
    PRIMARY KEY (provider, period)
);

CREATE TABLE job_search_cache (
    cache_key TEXT PRIMARY KEY,
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
