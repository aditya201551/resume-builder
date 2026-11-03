package jobmap

import (
	"context"
	"os"
	"testing"
	"time"
)

// One optional, filtered request. Normal test runs never contact providers.
func TestLiveFilteredHimalayasSearch(t *testing.T) {
	if os.Getenv("JOB_SEARCH_LIVE_TEST") != "1" {
		t.Skip("set JOB_SEARCH_LIVE_TEST=1 to validate a public search page")
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	jobs, _, err := NewClient(cfg).SearchHimalayasPage(ctx, "software engineer", "IN", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) == 0 {
		t.Fatal("expected a usable search page")
	}
	for _, j := range jobs {
		if j.Workplace != "remote" || len(j.Locations) != 0 {
			t.Fatal("remote listing acquired an office location")
		}
	}
}
