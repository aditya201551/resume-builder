package jobmap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type himalayasJob struct {
	GUID        string    `json:"guid"`
	Title       string    `json:"title"`
	Company     string    `json:"companyName"`
	Description string    `json:"description"`
	Excerpt     string    `json:"excerpt"`
	ApplyURL    string    `json:"applicationLink"`
	Regions     *[]string `json:"locationRestrictions"`
	Timezones   []float64 `json:"timezoneRestrictions"`
	Employment  string    `json:"employmentType"`
	Seniority   []string  `json:"seniority"`
	Updated     int64     `json:"pubDate"`
	Expiry      int64     `json:"expiryDate"`
	Currency    string    `json:"currency"`
	Min         *float64  `json:"minSalary"`
	Max         *float64  `json:"maxSalary"`
	Period      string    `json:"salaryPeriod"`
}

// SearchHimalayasPage requests a filtered page rather than the bulk feed. A
// short page is not terminal; the provider documents an empty page as the end.
func (c *Client) SearchHimalayasPage(ctx context.Context, query, country string, page int) ([]Job, bool, error) {
	var body struct {
		Jobs []himalayasJob `json:"jobs"`
	}
	v := url.Values{"page": {strconv.Itoa(page)}, "sort": {"recent"}}
	if query != "" {
		v.Set("q", query)
	}
	if country != "" {
		v.Set("country", country)
	}
	if err := c.get(ctx, "https://himalayas.app/jobs/api/search?"+v.Encode(), c.himalayasHeaders(), &body); err != nil {
		return nil, false, err
	}
	if body.Jobs == nil || len(body.Jobs) > 100 {
		return nil, false, fmt.Errorf("invalid Himalayas search page")
	}
	jobs := []Job{}
	for _, j := range body.Jobs {
		jobs = append(jobs, j.job())
	}
	return jobs, len(body.Jobs) > 0, nil
}

func (c *Client) himalayasHeaders() map[string]string {
	return map[string]string{"Authorization": bearer(c.Config.HimalayasKey)}
}

func bearer(key string) string {
	if key == "" {
		return ""
	}
	return "Bearer " + key
}

// FetchHimalayasJob only requests the documented API path on the fixed provider
// host. HTTP 410 is authoritative closure; 404, outages and rate limits are not.
func (c *Client) FetchHimalayasJob(ctx context.Context, guid string) (Job, bool, error) {
	u, err := url.Parse(guid)
	if err != nil || u.Scheme != "https" || u.Host != "himalayas.app" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return Job{}, false, fmt.Errorf("invalid Himalayas job identity")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[0] != "companies" || parts[2] != "jobs" || parts[1] == "" || parts[3] == "" || parts[1] == "." || parts[1] == ".." || parts[3] == "." || parts[3] == ".." {
		return Job{}, false, fmt.Errorf("invalid Himalayas job path")
	}
	var body struct {
		Job *himalayasJob `json:"job"`
	}
	err = c.get(ctx, "https://himalayas.app/jobs/api"+u.EscapedPath(), c.himalayasHeaders(), &body)
	var status *FeedHTTPError
	if errors.As(err, &status) && status.StatusCode == http.StatusGone {
		return Job{}, true, nil
	}
	if err != nil {
		return Job{}, false, err
	}
	if body.Job == nil || body.Job.GUID != guid {
		return Job{}, false, fmt.Errorf("invalid Himalayas job response")
	}
	job := body.Job.job()
	return job, job.ExpiresAt != nil && !job.ExpiresAt.After(time.Now()), nil
}

func (j himalayasJob) job() Job {
	job := Job{ProviderID: j.GUID, Source: "himalayas", SourceName: "Himalayas", SourceKind: "aggregator", Company: j.Company, Title: j.Title, Description: j.Description, SourceURL: j.GUID, ApplyURL: j.ApplyURL, Workplace: "remote", EmploymentType: Employment(j.Employment), Seniority: Seniority(strings.Join(j.Seniority, " ")), Remote: RemotePolicy{Scope: "unspecified", Regions: []string{}, Timezones: j.Timezones}}
	if job.Description == "" {
		job.Description = j.Excerpt
	}
	if j.Regions != nil {
		job.Remote.Regions = *j.Regions
		job.Remote.Scope = "worldwide"
		if len(*j.Regions) > 0 {
			job.Remote.Scope = "restricted"
		}
		for _, r := range *j.Regions {
			if strings.EqualFold(r, "worldwide") || strings.EqualFold(r, "anywhere") {
				job.Remote.Scope = "worldwide"
			}
		}
	}
	// An absent restriction stays unknown; timezone offsets are always preserved.
	if j.Updated > 0 {
		t := time.Unix(j.Updated, 0)
		job.SourceUpdatedAt = &t
	}
	if j.Expiry > 0 {
		t := time.Unix(j.Expiry, 0)
		job.ExpiresAt = &t
	}
	if j.Min != nil || j.Max != nil {
		parts := []string{}
		if j.Min != nil {
			parts = append(parts, fmt.Sprintf("%.0f", *j.Min))
		}
		if j.Max != nil {
			parts = append(parts, fmt.Sprintf("%.0f", *j.Max))
		}
		job.Salary = strings.TrimSpace(j.Currency + " " + strings.Join(parts, " to ") + " " + j.Period)
	}
	return job
}
