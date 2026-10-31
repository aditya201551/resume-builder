package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"resume-builder/backend/internal/jobmap"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type jobDB interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Begin(context.Context) (pgx.Tx, error)
}

type JobRepository struct{ pool jobDB }

func NewJobRepository(pool jobDB) *JobRepository { return &JobRepository{pool: pool} }

func (r *JobRepository) StoreSource(ctx context.Context, source string, jobs []jobmap.Job, complete bool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin job refresh: %w", err)
	}
	defer tx.Rollback(ctx)
	ids := []string{}
	for _, job := range jobs {
		if job.Source != source {
			return fmt.Errorf("job source mismatch")
		}
		if err := storeJob(ctx, tx, job); err != nil {
			return err
		}
		ids = append(ids, job.ID)
	}
	if complete {
		if _, err := tx.Exec(ctx, `UPDATE job_listings SET closed=true WHERE source=$1 AND NOT (id=ANY($2::text[]))`, source, ids); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func storeJob(ctx context.Context, tx pgx.Tx, job jobmap.Job) error {
	job.CheckFailed = false // transient request failures are never stored as job data
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO job_listings (id,source,provider_id,payload,expires_at,fetched_at,closed,checked_at)
	 VALUES ($1,$2,$3,$4,$5,$6,false,$7) ON CONFLICT (id) DO UPDATE SET
	 payload=EXCLUDED.payload, expires_at=EXCLUDED.expires_at, fetched_at=EXCLUDED.fetched_at,
	 closed=CASE WHEN EXCLUDED.checked_at IS NOT NULL THEN false
	   WHEN job_listings.source='himalayas' THEN job_listings.closed ELSE false END,
	 checked_at=COALESCE(EXCLUDED.checked_at,job_listings.checked_at)
	 WHERE EXCLUDED.checked_at IS NULL OR job_listings.checked_at IS NULL OR EXCLUDED.checked_at>=job_listings.checked_at`,
		job.ID, job.Source, job.ProviderID, payload, job.ExpiresAt, job.FetchedAt, job.CheckedAt)
	if err != nil {
		return fmt.Errorf("store job: %w", err)
	}
	return nil
}

func (r *JobRepository) MarkJobClosed(ctx context.Context, id string, checkedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE job_listings SET closed=true,checked_at=$2
	 WHERE id=$1 AND (checked_at IS NULL OR checked_at<=$2)`, id, checkedAt)
	return err
}

func (r *JobRepository) FeedProgress(ctx context.Context, source string) (jobmap.FeedProgress, error) {
	var p jobmap.FeedProgress
	err := r.pool.QueryRow(ctx, `SELECT cursor,job_count,completed_at,retry_at FROM job_feed_progress WHERE source=$1`, source).
		Scan(&p.Cursor, &p.JobCount, &p.CompletedAt, &p.RetryAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, nil
	}
	return p, err
}

// Page data and its cursor commit together; crashes can safely retry the page.
// Cursor feeds are not atomic snapshots: missing jobs are never closed here.
func (r *JobRepository) StoreFeedPage(ctx context.Context, source string, jobs []jobmap.Job, p jobmap.FeedProgress) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, job := range jobs {
		if job.Source != source {
			return fmt.Errorf("feed source mismatch")
		}
		if err := storeJob(ctx, tx, job); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO job_feed_progress(source,cursor,job_count,completed_at)
	 VALUES($1,$2,$3,$4) ON CONFLICT(source) DO UPDATE SET
	 cursor=EXCLUDED.cursor,job_count=EXCLUDED.job_count,completed_at=EXCLUDED.completed_at`, source, p.Cursor, p.JobCount, p.CompletedAt)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *JobRepository) DelayFeed(ctx context.Context, source string, until time.Time) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO job_feed_progress(source,retry_at) VALUES($1,$2)
	 ON CONFLICT(source) DO UPDATE SET retry_at=GREATEST(job_feed_progress.retry_at,EXCLUDED.retry_at)`, source, until)
	return err
}

func (r *JobRepository) List(ctx context.Context, userID string, savedOnly bool) ([]jobmap.Job, error) {
	query := `SELECT j.payload, j.checked_at, (j.closed OR (j.expires_at IS NOT NULL AND j.expires_at<=now()) OR GREATEST(j.fetched_at,j.checked_at)<now()-interval '72 hours') AS unavailable,
	 EXISTS(SELECT 1 FROM saved_jobs s WHERE s.job_id=j.id AND s.user_id=$1) AS saved FROM job_listings j WHERE `
	if savedOnly {
		query += `EXISTS(SELECT 1 FROM saved_jobs s WHERE s.job_id=j.id AND s.user_id=$1)`
	} else {
		query += `NOT j.closed AND (j.expires_at IS NULL OR j.expires_at>now()) AND GREATEST(j.fetched_at,j.checked_at)>=now()-interval '72 hours'`
	}
	query += ` ORDER BY j.fetched_at DESC,j.id`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []jobmap.Job{}
	for rows.Next() {
		var data []byte
		var checkedAt *time.Time
		var closed, saved bool
		if err := rows.Scan(&data, &checkedAt, &closed, &saved); err != nil {
			return nil, err
		}
		var job jobmap.Job
		if err := json.Unmarshal(data, &job); err != nil {
			return nil, err
		}
		job.Closed, job.Saved = closed, saved
		job.CheckedAt = checkedAt
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *JobRepository) Get(ctx context.Context, userID, id string) (jobmap.Job, error) {
	var data []byte
	var checkedAt *time.Time
	var closed, saved bool
	err := r.pool.QueryRow(ctx, `SELECT j.payload, j.checked_at, (j.closed OR (j.expires_at IS NOT NULL AND j.expires_at<=now()) OR GREATEST(j.fetched_at,j.checked_at)<now()-interval '72 hours'),
	 EXISTS(SELECT 1 FROM saved_jobs s WHERE s.job_id=j.id AND s.user_id=$1) FROM job_listings j WHERE j.id=$2`, userID, id).Scan(&data, &checkedAt, &closed, &saved)
	if errors.Is(err, pgx.ErrNoRows) {
		return jobmap.Job{}, ErrNotFound
	}
	if err != nil {
		return jobmap.Job{}, err
	}
	var job jobmap.Job
	err = json.Unmarshal(data, &job)
	job.Closed, job.Saved = closed, saved
	job.CheckedAt = checkedAt
	return job, err
}

func (r *JobRepository) Save(ctx context.Context, userID, id string) error {
	tag, err := r.pool.Exec(ctx, `INSERT INTO saved_jobs (user_id,job_id) SELECT $1,id FROM job_listings WHERE id=$2 ON CONFLICT DO NOTHING`, userID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM job_listings WHERE id=$1)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrNotFound
		}
	}
	return nil
}

func (r *JobRepository) Unsave(ctx context.Context, userID, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM saved_jobs WHERE user_id=$1 AND job_id=$2`, userID, id)
	return err
}

func (r *JobRepository) StoreStatus(ctx context.Context, status jobmap.SourceStatus) error {
	data, err := json.Marshal(status)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `INSERT INTO job_source_status(source,payload) VALUES($1,$2) ON CONFLICT(source) DO UPDATE SET payload=EXCLUDED.payload`, status.ID, data)
	return err
}

func (r *JobRepository) Statuses(ctx context.Context) ([]jobmap.SourceStatus, error) {
	rows, err := r.pool.Query(ctx, `SELECT payload FROM job_source_status ORDER BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []jobmap.SourceStatus{}
	for rows.Next() {
		var data []byte
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var s jobmap.SourceStatus
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
		items = append(items, s)
	}
	return items, rows.Err()
}

func (r *JobRepository) ReserveRequest(ctx context.Context, provider, period string, limit int) (bool, error) {
	if limit <= 0 {
		return false, nil
	}
	var n int
	err := r.pool.QueryRow(ctx, `INSERT INTO job_api_usage(provider,period,requests) VALUES($1,$2,1)
	 ON CONFLICT(provider,period) DO UPDATE SET requests=job_api_usage.requests+1
	 WHERE job_api_usage.requests<$3 RETURNING requests`, provider, period, limit).Scan(&n)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// ClaimSearch atomically grants one process a refresh lease; repeated requests
// share a cache key even after the Go process restarts. The lease is released on failure.
func (r *JobRepository) ClaimSearch(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	var claimed string
	err := r.pool.QueryRow(ctx, `INSERT INTO job_search_cache(cache_key,fetched_at) VALUES($1,now()) ON CONFLICT(cache_key)
	 DO UPDATE SET fetched_at=now() WHERE job_search_cache.fetched_at<now()-($2*interval '1 second') RETURNING cache_key`, key, ttl.Seconds()).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *JobRepository) ReleaseSearch(ctx context.Context, key string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM job_search_cache WHERE cache_key=$1`, key)
	return err
}

func (r *JobRepository) Prune(ctx context.Context) error {
	// Keep unavailable saved records to explain closed bookmarks; purge other stale data.
	_, err := r.pool.Exec(ctx, `DELETE FROM job_listings j WHERE GREATEST(j.fetched_at,j.checked_at)<now()-interval '14 days'
	 AND NOT EXISTS(SELECT 1 FROM saved_jobs s WHERE s.job_id=j.id)`)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `DELETE FROM job_search_cache WHERE fetched_at<now()-interval '7 days'`)
	return err
}
