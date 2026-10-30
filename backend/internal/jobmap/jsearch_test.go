package jobmap

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestJSearchProviderAuthenticationAndV2Results(t *testing.T) {
	for _, provider := range []string{"", "openwebninja", "rapidapi"} {
		t.Run(provider, func(t *testing.T) {
			client := NewClient(Config{JSearchProvider: provider, JSearchKey: "test-secret"})
			client.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				host, path := "api.openwebninja.com", "/jsearch/search-v2"
				if provider == "rapidapi" {
					host, path = "jsearch.p.rapidapi.com", "/search-v2"
					if r.Header.Get("X-RapidAPI-Key") != "test-secret" || r.Header.Get("X-RapidAPI-Host") != host || r.Header.Get("x-api-key") != "" {
						t.Fatal("RapidAPI authentication is missing or mixed with direct authentication")
					}
				} else if r.Header.Get("x-api-key") != "test-secret" || r.Header.Get("X-RapidAPI-Key") != "" {
					t.Fatal("direct authentication is missing or mixed with RapidAPI authentication")
				}
				if r.URL.Scheme != "https" || r.URL.Host != host || r.URL.Path != path {
					t.Fatal("incorrect provider endpoint")
				}
				q := r.URL.Query()
				if q.Get("query") != "software engineer in Bengaluru, India" || q.Get("country") != "in" || q.Get("num_pages") != "1" || strings.Contains(r.URL.String(), "test-secret") {
					t.Fatal("incorrect query, page budget, or credential placement")
				}
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"status":"OK","data":{"cursor":"next-page","jobs":[{"job_id":"job-1","job_title":"Software Engineer","employer_name":"Example","job_apply_link":"https://example.com/careers/1","job_is_remote":false,"job_location":"Bengaluru, Karnataka, India","job_country":"IN","job_employment_type":"Full-time"},{"job_id":"job-2","job_title":"Backend Engineer","employer_name":"Example","job_apply_link":"https://example.com/careers/2","job_is_remote":true,"job_location":"Remote, India"}]}}`))}, nil
			})
			jobs, err := client.FetchJSearch(context.Background(), "software engineer in Bengaluru, India", "IN")
			if err != nil || len(jobs) != 2 {
				t.Fatalf("expected two v2 listings: count=%d error=%v", len(jobs), err)
			}
			job, ok := Prepare(jobs[0], time.Now())
			if !ok || job.Source != "jsearch" || job.EmploymentType != "full_time" || len(job.Locations) != 1 || job.Locations[0].Latitude == nil || job.Locations[0].CountryCode != "IN" {
				t.Fatalf("job could not be prepared for map: %+v", job)
			}
			if jobs[1].Workplace != "remote" || len(jobs[1].Locations) != 0 || jobs[1].Remote.Scope != "restricted" {
				t.Fatalf("remote restrictions were lost: %+v", jobs[1])
			}
		})
	}
}

func TestJSearchRejectsProviderFailuresAndMalformedSuccess(t *testing.T) {
	for _, test := range []struct {
		name      string
		status    int
		body      string
		wantError bool
	}{
		{"unauthorized", 401, `{"message":"Unauthorized"}`, true},
		{"not subscribed", 403, `{"message":"You are not subscribed to this API."}`, true},
		{"quota", 429, `{"message":"Too many requests"}`, true},
		{"provider error", 200, `{"status":"ERROR","error":{"message":"upstream error"}}`, true},
		{"missing data", 200, `{"status":"OK"}`, true},
		{"missing jobs", 200, `{"status":"OK","data":{}}`, true},
		{"empty search", 200, `{"status":"OK","data":{"jobs":[]}}`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := NewClient(Config{JSearchProvider: "rapidapi", JSearchKey: "test-secret"})
			client.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})
			_, err := client.FetchJSearch(context.Background(), "software engineer in Bengaluru, India", "IN")
			if (err != nil) != test.wantError || calls != 1 {
				t.Fatalf("unexpected result or extra quota consumed: error=%v calls=%d", err, calls)
			}
		})
	}
}

func TestJSearchProviderConfig(t *testing.T) {
	t.Setenv("JOB_MAP_COMPANIES_FILE", "")
	t.Setenv("JOB_MAP_REFRESH_INTERVAL", "")
	t.Setenv("JSEARCH_MONTHLY_LIMIT", "")
	t.Setenv("GEOAPIFY_DAILY_LIMIT", "")
	for _, test := range []struct {
		value, want string
		invalid     bool
	}{
		{"", "openwebninja", false},
		{"openwebninja", "openwebninja", false},
		{" RapidAPI ", "rapidapi", false},
		{"typo", "", true},
	} {
		t.Run(test.value, func(t *testing.T) {
			t.Setenv("JSEARCH_PROVIDER", test.value)
			cfg, err := LoadConfig()
			if (err != nil) != test.invalid || (!test.invalid && cfg.JSearchProvider != test.want) {
				t.Fatalf("unexpected provider config: %q error=%v", cfg.JSearchProvider, err)
			}
		})
	}
}
