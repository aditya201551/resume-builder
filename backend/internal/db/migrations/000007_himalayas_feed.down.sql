UPDATE job_listings
SET payload = (payload - 'sourceUpdatedAt') || jsonb_build_object('postedAt', payload->'sourceUpdatedAt')
WHERE source = 'himalayas' AND payload ? 'sourceUpdatedAt';

ALTER TABLE job_listings DROP COLUMN checked_at;
DROP TABLE job_feed_progress;
