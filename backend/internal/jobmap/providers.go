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
	req.Header.Set("User-Agent", "ResumeBuilder-JobMap/1.0 (public job feeds)")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
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

type ashbyAddress struct {
	PostalAddress struct {
		Country  string `json:"addressCountry"`
		Locality string `json:"addressLocality"`
	} `json:"postalAddress"`
}
type ashbyLocation struct {
	Location string       `json:"location"`
	Address  ashbyAddress `json:"address"`
}

func (c *Client) FetchCompany(ctx context.Context, company Company) ([]Job, error) {
	switch company.Provider {
	case "ashby":
		return c.ashby(ctx, company)
	case "greenhouse":
		return c.greenhouse(ctx, company)
	case "lever":
		return c.lever(ctx, company)
	case "career_page":
		return c.careerPage(ctx, company)
	default:
		return nil, fmt.Errorf("unsupported provider")
	}
}

func (c *Client) ashby(ctx context.Context, company Company) ([]Job, error) {
	var body struct {
		Jobs []struct {
			ID              string          `json:"id"`
			Title           string          `json:"title"`
			Location        string          `json:"location"`
			Description     string          `json:"descriptionPlain"`
			DescriptionHTML string          `json:"descriptionHtml"`
			Workplace       string          `json:"workplaceType"`
			Remote          *bool           `json:"isRemote"`
			Listed          bool            `json:"isListed"`
			Employment      string          `json:"employmentType"`
			Published       string          `json:"publishedAt"`
			JobURL          string          `json:"jobUrl"`
			ApplyURL        string          `json:"applyUrl"`
			Address         ashbyAddress    `json:"address"`
			Secondary       []ashbyLocation `json:"secondaryLocations"`
			Compensation    struct {
				Summary string `json:"scrapeableCompensationSalarySummary"`
			} `json:"compensation"`
		} `json:"jobs"`
	}
	if err := c.get(ctx, "https://api.ashbyhq.com/posting-api/job-board/"+company.Board+"?includeCompensation=true", nil, &body); err != nil {
		return nil, err
	}
	jobs := []Job{}
	if body.Jobs == nil {
		return nil, fmt.Errorf("company feed omitted its jobs list")
	}
	for _, j := range body.Jobs {
		if !j.Listed || !IsTech(j.Title) {
			continue
		}
		job := companyJob(company, j.ID, j.Title, j.JobURL, j.ApplyURL)
		job.Description, job.Salary = j.Description, j.Compensation.Summary
		if job.Description == "" {
			job.Description = j.DescriptionHTML
		}
		job.Workplace = Workplace(j.Workplace, j.Remote, j.Location)
		job.EmploymentType, job.PostedAt = Employment(j.Employment), parseTime(j.Published)
		all := append([]ashbyLocation{{Location: j.Location, Address: j.Address}}, j.Secondary...)
		remoteNames := []string{}
		for _, l := range all {
			if l.Location == "" {
				continue
			}
			if job.Workplace == "remote" || strings.Contains(strings.ToLower(l.Location), "remote") {
				// A postal country can describe an office, not accepted remote
				// applicants. Use the advertised remote location text only.
				remoteNames = append(remoteNames, l.Location)
			} else {
				job.Locations = append(job.Locations, ResolveLocation(l.Location, l.Address.PostalAddress.Country))
			}
		}
		job.Remote = RemoteFromLocations(remoteNames)
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (c *Client) greenhouse(ctx context.Context, company Company) ([]Job, error) {
	var body struct {
		Jobs []struct {
			ID        int64   `json:"id"`
			Title     string  `json:"title"`
			Content   string  `json:"content"`
			URL       string  `json:"absolute_url"`
			Published string  `json:"first_published"`
			Language  string  `json:"language"`
			Deadline  *string `json:"application_deadline"`
			Location  struct {
				Name string `json:"name"`
			} `json:"location"`
		} `json:"jobs"`
	}
	if err := c.get(ctx, "https://boards-api.greenhouse.io/v1/boards/"+company.Board+"/jobs?content=true", nil, &body); err != nil {
		return nil, err
	}
	jobs := []Job{}
	if body.Jobs == nil {
		return nil, fmt.Errorf("company feed omitted its jobs list")
	}
	for _, j := range body.Jobs {
		if !IsTech(j.Title) {
			continue
		}
		job := companyJob(company, strconv.FormatInt(j.ID, 10), j.Title, j.URL, j.URL)
		job.Description, job.Language, job.PostedAt = j.Content, j.Language, parseTime(j.Published)
		if j.Deadline != nil {
			job.ExpiresAt = parseTime(*j.Deadline)
		}
		job.Workplace = Workplace("", nil, j.Location.Name)
		if job.Workplace == "remote" {
			job.Remote = RemoteFromLocations([]string{j.Location.Name})
		} else {
			// One posting may name several offices separated by bullets or semicolons.
			for _, name := range strings.FieldsFunc(j.Location.Name, func(r rune) bool { return r == '•' || r == ';' }) {
				job.Locations = append(job.Locations, ResolveLocation(strings.TrimSpace(name), ""))
			}
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (c *Client) lever(ctx context.Context, company Company) ([]Job, error) {
	jobs := []Job{}
	for page := 0; page < 20; page++ {
		var body []struct {
			ID          string `json:"id"`
			Text        string `json:"text"`
			Description string `json:"descriptionPlain"`
			Additional  string `json:"additionalPlain"`
			URL         string `json:"hostedUrl"`
			ApplyURL    string `json:"applyUrl"`
			Workplace   string `json:"workplaceType"`
			Created     int64  `json:"createdAt"`
			Categories  struct {
				Location   string `json:"location"`
				Commitment string `json:"commitment"`
			} `json:"categories"`
			Lists []struct {
				Text    string `json:"text"`
				Content string `json:"content"`
			} `json:"lists"`
		}
		endpoint := fmt.Sprintf("https://api.lever.co/v0/postings/%s?mode=json&limit=100&skip=%d", company.Board, page*100)
		if err := c.get(ctx, endpoint, nil, &body); err != nil {
			return nil, err
		}
		for _, j := range body {
			if !IsTech(j.Text) {
				continue
			}
			job := companyJob(company, j.ID, j.Text, j.URL, j.ApplyURL)
			job.Description = j.Description
			for _, list := range j.Lists {
				job.Description += "\n\n" + list.Text + "\n" + list.Content
			}
			job.Description += "\n\n" + j.Additional
			job.Workplace, job.EmploymentType = Workplace(j.Workplace, nil, j.Categories.Location), Employment(j.Categories.Commitment)
			if j.Created > 0 {
				t := time.UnixMilli(j.Created)
				job.PostedAt = &t
			}
			if job.Workplace == "remote" {
				job.Remote = RemoteFromLocations([]string{j.Categories.Location})
			} else {
				job.Locations = []Location{ResolveLocation(j.Categories.Location, "")}
			}
			jobs = append(jobs, job)
		}
		if len(body) < 100 {
			return jobs, nil
		}
	}
	return nil, fmt.Errorf("company feed exceeded the safe page limit; retained previous snapshot")
}

func companyJob(c Company, id, title, sourceURL, applyURL string) Job {
	if applyURL == "" {
		applyURL = sourceURL
	}
	kind := "company"
	if c.Provider == "career_page" {
		kind = "career_page"
	}
	return Job{ProviderID: id, Source: CompanySource(c), SourceName: c.Name + " careers", SourceKind: kind, Company: c.Name, CompanyURL: c.Website, Title: title, SourceURL: sourceURL, ApplyURL: applyURL}
}

func parseTime(s string) *time.Time {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

func (c *Client) FetchJSearch(ctx context.Context, query, country string) ([]Job, error) {
	var body struct {
		Status string `json:"status"`
		Data   *struct {
			Jobs []struct {
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
	endpoint := "https://api.openwebninja.com/jsearch/search-v2"
	headers := map[string]string{"x-api-key": c.Config.JSearchKey}
	switch c.Config.JSearchProvider {
	case "rapidapi":
		endpoint = "https://jsearch.p.rapidapi.com/search-v2"
		headers = map[string]string{"x-rapidapi-key": c.Config.JSearchKey, "x-rapidapi-host": "jsearch.p.rapidapi.com"}
	case "", "openwebninja":
	default:
		return nil, fmt.Errorf("unsupported JSearch provider")
	}
	if err := c.get(ctx, endpoint+"?"+values.Encode(), headers, &body); err != nil {
		return nil, err
	}
	if body.Status != "OK" || body.Data == nil || body.Data.Jobs == nil {
		return nil, fmt.Errorf("search provider did not return usable results")
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
	return jobs, nil
}
