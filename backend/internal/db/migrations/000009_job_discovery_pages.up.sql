ALTER TABLE job_listings ADD COLUMN discovery_cached boolean NOT NULL DEFAULT false;
CREATE INDEX job_listings_discovery_cached_idx ON job_listings (id) WHERE discovery_cached;

CREATE TABLE job_discovery_pages (
    cache_key text PRIMARY KEY,
    job_ids text[] NOT NULL,
    next_cursor text NOT NULL DEFAULT '',
    has_more boolean NOT NULL,
    fetched_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX job_discovery_pages_age_idx ON job_discovery_pages (fetched_at);
CREATE INDEX job_discovery_pages_jobs_idx ON job_discovery_pages USING gin (job_ids);
CREATE INDEX saved_jobs_recent_idx ON saved_jobs (user_id, saved_at DESC, job_id);
