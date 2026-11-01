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

func (r *JobRepository) StoreSource(ctx context.Context, source string, jobs []jobmap.Job) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin job refresh: %w", err)
	}
	defer tx.Rollback(ctx)
	for _, job := range jobs {
		if job.Source != source {
			return fmt.Errorf("job source mismatch")
		}
		if err := storeJob(ctx, tx, job); err != nil {
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

func (r *JobRepository) ProviderRetryAt(ctx context.Context, source string) (*time.Time, error) {
	var retryAt *time.Time
	err := r.pool.QueryRow(ctx, `SELECT retry_at FROM job_feed_progress WHERE source=$1`, source).Scan(&retryAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return retryAt, err
}

func (r *JobRepository) DelayFeed(ctx context.Context, source string, until time.Time) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO job_feed_progress(source,retry_at) VALUES($1,$2)
	 ON CONFLICT(source) DO UPDATE SET retry_at=GREATEST(job_feed_progress.retry_at,EXCLUDED.retry_at)`, source, until)
	return err
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

func (r *JobRepository) ClaimExternalSearch(ctx context.Context, key, token string, lease, cache time.Duration) (bool, error) {
	var claimed string
	err := r.pool.QueryRow(ctx, `INSERT INTO job_external_search_cache(cache_key,lease_token,lease_until)
	 VALUES($1,$2,now()+($3*interval '1 second')) ON CONFLICT(cache_key) DO UPDATE
	 SET lease_token=EXCLUDED.lease_token,lease_until=EXCLUDED.lease_until
	 WHERE (job_external_search_cache.lease_until IS NULL OR job_external_search_cache.lease_until<=now())
	 AND (job_external_search_cache.retry_at IS NULL OR job_external_search_cache.retry_at<=now())
	 AND (job_external_search_cache.refreshed_at IS NULL OR job_external_search_cache.refreshed_at<=now()-($4*interval '1 second'))
	 RETURNING lease_token`, key, token, lease.Seconds(), cache.Seconds()).Scan(&claimed)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func (r *JobRepository) CompleteExternalSearch(ctx context.Context, key, token string) error {
	result, err := r.pool.Exec(ctx, `UPDATE job_external_search_cache SET refreshed_at=now(),lease_token=NULL,lease_until=NULL,retry_at=NULL
	 WHERE cache_key=$1 AND lease_token=$2 AND lease_until>now()`, key, token)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return fmt.Errorf("external search lease expired or changed owner")
	}
	return nil
}

func (r *JobRepository) ReleaseExternalSearch(ctx context.Context, key, token string) error {
	_, err := r.pool.Exec(ctx, `UPDATE job_external_search_cache SET lease_token=NULL,lease_until=NULL WHERE cache_key=$1 AND lease_token=$2`, key, token)
	return err
}

func (r *JobRepository) FailExternalSearch(ctx context.Context, key, token string, retryAt time.Time) error {
	_, err := r.pool.Exec(ctx, `UPDATE job_external_search_cache SET retry_at=$3,lease_token=NULL,lease_until=NULL
	 WHERE cache_key=$1 AND lease_token=$2`, key, token, retryAt)
	return err
}
