DROP TABLE job_discovery_pages;
DROP INDEX saved_jobs_recent_idx;
DROP INDEX job_listings_discovery_cached_idx;
ALTER TABLE job_listings DROP COLUMN discovery_cached;
