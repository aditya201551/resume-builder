package jobmap

import (
	"context"
	"os"
	"testing"
	"time"
)

// Opt-in integration check against public feeds; normal test runs stay offline.
func TestLivePublicFeeds(t *testing.T) {
	if os.Getenv("JOB_MAP_LIVE_TEST") != "1" {
		t.Skip("set JOB_MAP_LIVE_TEST=1 to validate real public feeds")
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(cfg)
	for _, company := range cfg.Companies {
		t.Run(company.Name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			defer cancel()
			jobs, err := client.FetchCompany(ctx, company)
			if err != nil {
				t.Fatal(err)
			}
			valid, mapped, remote := 0, 0, 0
			for _, j := range jobs {
				job, ok := Prepare(j, time.Now())
				if !ok {
					continue
				}
				valid++
				if job.Workplace == "remote" {
					remote++
				}
				for _, l := range job.Locations {
					if l.Latitude != nil {
						mapped++
						break
					}
				}
			}
			t.Logf("%d tech listings, %d with city coordinates, %d remote", valid, mapped, remote)
			if valid == 0 {
				t.Fatal("expected usable tech jobs from the seed company")
			}
		})
	}
	t.Run("Himalayas", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		jobs, err := client.FetchHimalayas(ctx)
		if err != nil {
			t.Fatal(err)
		}
		valid := 0
		for _, j := range jobs {
			if job, ok := Prepare(j, time.Now()); ok {
				valid++
				if job.Workplace != "remote" || len(job.Locations) != 0 {
					t.Fatal("remote job acquired office coordinates")
				}
			}
		}
		t.Logf("%d usable remote tech listings", valid)
		if valid == 0 {
			t.Fatal("expected remote tech listings")
		}
	})
}
