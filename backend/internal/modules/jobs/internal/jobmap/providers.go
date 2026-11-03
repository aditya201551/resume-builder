package jobmap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	HTTP   *http.Client
	Config Config
}

type FeedHTTPError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e *FeedHTTPError) Error() string { return fmt.Sprintf("feed returned HTTP %d", e.StatusCode) }

func NewClient(cfg Config) *Client {
	return &Client{Config: cfg, HTTP: &http.Client{Timeout: 25 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || req.URL.Scheme != "https" || (len(via) > 0 && req.URL.Hostname() != via[0].URL.Hostname()) {
			return fmt.Errorf("cross-host redirect refused")
		}
		return nil
	}}}
}

func (c *Client) get(ctx context.Context, rawURL string, headers map[string]string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create feed request")
	}
	req.Header.Set("User-Agent", "ResumeBuilder-JobSearch/1.0")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("feed request failed: %w", ctx.Err())
		}
		return fmt.Errorf("feed request failed")
	} // do not leak API keys in request URLs
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		delay := time.Duration(0)
		if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 && seconds <= 7*24*60*60 {
			delay = time.Duration(seconds) * time.Second
		} else if at, err := http.ParseTime(resp.Header.Get("Retry-After")); err == nil {
			delay = max(0, time.Until(at))
		}
		return &FeedHTTPError{StatusCode: resp.StatusCode, RetryAfter: delay}
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 24<<20)).Decode(out); err != nil {
		return fmt.Errorf("invalid feed response")
	}
	return nil
}

func parseTime(s string) *time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

// FetchJSearchPage preserves the provider's opaque cursor; it never walks the
// next page automatically. country is required to avoid a hidden US default.
func (c *Client) FetchJSearchPage(ctx context.Context, query, country, cursor string) ([]Job, string, error) {
	var body struct {
		Status string `json:"status"`
		Cursor string `json:"cursor"`
		Data   *struct {
			Cursor     string `json:"cursor"`
			NextCursor string `json:"next_cursor"`
			Jobs       []struct {
				ID          string `json:"job_id"`
				Title       string `json:"job_title"`
				Company     string `json:"employer_name"`
				Website     string `json:"employer_website"`
				Description string `json:"job_description"`
				ApplyURL    string `json:"job_apply_link"`
				Remote      *bool  `json:"job_is_remote"`
				Location    string `json:"job_location"`
				City        string `json:"job_city"`
				Country     string `json:"job_country"`
				Workplace   string `json:"work_arrangement"`
				Employment  string `json:"job_employment_type"`
				Seniority   string `json:"seniority_level"`
				Salary      string `json:"job_salary_string"`
				Published   string `json:"job_posted_at_datetime_utc"`
			} `json:"jobs"`
		} `json:"data"`
	}
	values := url.Values{"query": {query}, "country": {strings.ToLower(country)}, "num_pages": {"1"}}
	if country == "" {
		return nil, "", fmt.Errorf("choose a search country")
	}
	if cursor != "" {
		values.Set("cursor", cursor)
	}
	endpoint := "https://api.openwebninja.com/jsearch/search-v2"
	headers := map[string]string{"x-api-key": c.Config.JSearchKey}
	switch c.Config.JSearchProvider {
	case "rapidapi":
		endpoint = "https://jsearch.p.rapidapi.com/search-v2"
		headers = map[string]string{"x-rapidapi-key": c.Config.JSearchKey, "x-rapidapi-host": "jsearch.p.rapidapi.com"}
	case "", "openwebninja":
	default:
		return nil, "", fmt.Errorf("unsupported JSearch provider")
	}
	if err := c.get(ctx, endpoint+"?"+values.Encode(), headers, &body); err != nil {
		return nil, "", err
	}
	if body.Status != "OK" || body.Data == nil || body.Data.Jobs == nil {
		return nil, "", fmt.Errorf("search provider did not return usable results")
	}
	jobs := []Job{}
	for _, j := range body.Data.Jobs {
		job := Job{ProviderID: j.ID, Source: "jsearch", SourceName: "JSearch", SourceKind: "aggregator", Company: j.Company, CompanyURL: j.Website, Title: j.Title, Description: j.Description, SourceURL: j.ApplyURL, ApplyURL: j.ApplyURL, Workplace: Workplace(j.Workplace, j.Remote, j.Location), Salary: j.Salary, PostedAt: parseTime(j.Published), EmploymentType: Employment(j.Employment), Seniority: Seniority(j.Seniority)}
		if job.Workplace == "remote" {
			job.Remote = RemoteFromLocations([]string{j.Location})
		} else {
			name := j.Location
			if name == "" {
				name = j.City
			}
			job.Locations = []Location{ResolveLocation(name, j.Country)}
		}
		jobs = append(jobs, job)
	}
	next := body.Data.Cursor
	if next == "" {
		next = body.Data.NextCursor
	}
	if next == "" {
		next = body.Cursor
	}
	if cursor != "" && next == cursor {
		return nil, "", fmt.Errorf("search cursor did not advance")
	}
	return jobs, next, nil
}
