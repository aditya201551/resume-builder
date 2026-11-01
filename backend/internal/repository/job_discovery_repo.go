package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"resume-builder/backend/internal/jobmap"
	"resume-builder/backend/internal/jobsearch"
)

func (r *JobRepository) DiscoveryPage(ctx context.Context, userID, key string) (jobsearch.Page, bool, error) {
	var p jobsearch.Page
	var ids []string
	err := r.pool.QueryRow(ctx, `SELECT job_ids,next_cursor,has_more,fetched_at FROM job_discovery_pages WHERE cache_key=$1 AND fetched_at>now()-interval '24 hours'`, key).Scan(&ids, &p.Cursor, &p.HasMore, &p.FetchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, false, nil
	}
	if err != nil {
		return p, false, err
	}
	p.Jobs, err = r.discoveryJobs(ctx, userID, ids)
	return p, true, err
}

func (r *JobRepository) discoveryJobs(ctx context.Context, userID string, ids []string) ([]jobmap.Job, error) {
	rows, err := r.pool.Query(ctx, `SELECT j.payload,j.checked_at,
	 (j.closed OR (j.expires_at IS NOT NULL AND j.expires_at<=now()) OR GREATEST(j.fetched_at,j.checked_at)<now()-interval '72 hours'),
	 EXISTS(SELECT 1 FROM saved_jobs s WHERE s.user_id=$1 AND s.job_id=j.id)
	 FROM unnest($2::text[]) WITH ORDINALITY ids(id,pos) JOIN job_listings j ON j.id=ids.id ORDER BY ids.pos`, userID, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []jobmap.Job{}
	for rows.Next() {
		var data []byte
		var j jobmap.Job
		if err := rows.Scan(&data, &j.CheckedAt, &j.Closed, &j.Saved); err != nil {
			return nil, err
		}
		checked, closed, saved := j.CheckedAt, j.Closed, j.Saved
		if err := json.Unmarshal(data, &j); err != nil {
			return nil, err
		}
		j.CheckedAt, j.Closed, j.Saved = checked, closed, saved
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

func (r *JobRepository) StoreDiscoveryPage(ctx context.Context, key string, p jobsearch.Page) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Serialize eviction across API replicas; at most 500 public pages survive.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(84392019)`); err != nil {
		return err
	}
	ids := []string{}
	for _, j := range p.Jobs {
		if err = storeJob(ctx, tx, j); err != nil {
			return err
		}
		ids = append(ids, j.ID)
	}
	if _, err = tx.Exec(ctx, `UPDATE job_listings SET discovery_cached=true WHERE id=ANY($1::text[])`, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO job_discovery_pages(cache_key,job_ids,next_cursor,has_more,fetched_at) VALUES($1,$2,$3,$4,$5)
	 ON CONFLICT(cache_key) DO UPDATE SET job_ids=EXCLUDED.job_ids,next_cursor=EXCLUDED.next_cursor,has_more=EXCLUDED.has_more,fetched_at=EXCLUDED.fetched_at`, key, ids, p.Cursor, p.HasMore, p.FetchedAt); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM job_discovery_pages WHERE fetched_at<now()-interval '24 hours' OR cache_key IN (SELECT cache_key FROM job_discovery_pages ORDER BY fetched_at DESC,cache_key OFFSET 500)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM job_listings j WHERE (j.discovery_cached OR GREATEST(j.fetched_at,j.checked_at)<now()-interval '14 days')
	 AND NOT EXISTS(SELECT 1 FROM saved_jobs s WHERE s.job_id=j.id)
	 AND NOT EXISTS(SELECT 1 FROM job_discovery_pages p WHERE p.job_ids @> ARRAY[j.id])`); err != nil {
		return err
	}
	// Search activity owns routine cache cleanup; there is no inventory worker.
	if _, err = tx.Exec(ctx, `DELETE FROM job_search_cache WHERE fetched_at<now()-interval '7 days'`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM job_external_search_cache
	 WHERE (lease_until IS NULL OR lease_until<=now()) AND (retry_at IS NULL OR retry_at<=now())
	 AND (refreshed_at IS NULL OR refreshed_at<now()-interval '7 days')`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *JobRepository) SavedDiscoveryPage(ctx context.Context, userID string, offset int) ([]jobmap.Job, bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT job_id FROM saved_jobs WHERE user_id=$1 ORDER BY saved_at DESC,job_id LIMIT 51 OFFSET $2`, userID, offset)
	if err != nil {
		return nil, false, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, false, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, false, err
	}
	more := len(ids) > 50
	if more {
		ids = ids[:50]
	}
	jobs, err := r.discoveryJobs(ctx, userID, ids)
	return jobs, more, err
}
