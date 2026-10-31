-- Separate in-flight ownership from the freshness of successfully stored results.
-- Legacy job_search_cache rows may represent canceled searches and are not reused.
CREATE TABLE job_external_search_cache (
    cache_key TEXT PRIMARY KEY,
    lease_token TEXT,
    lease_until TIMESTAMPTZ,
    refreshed_at TIMESTAMPTZ,
    retry_at TIMESTAMPTZ
);
