CREATE TABLE job_feed_progress (
    source TEXT PRIMARY KEY,
    cursor TEXT NOT NULL DEFAULT '',
    job_count INTEGER NOT NULL DEFAULT 0 CHECK (job_count >= 0),
    completed_at TIMESTAMPTZ,
    retry_at TIMESTAMPTZ
);

ALTER TABLE job_listings ADD COLUMN checked_at TIMESTAMPTZ;

-- Himalayas pubDate is a source update timestamp, not an original posting date.
UPDATE job_listings
SET payload = (payload - 'postedAt') || jsonb_build_object('sourceUpdatedAt', payload->'postedAt')
WHERE source = 'himalayas' AND payload ? 'postedAt';
