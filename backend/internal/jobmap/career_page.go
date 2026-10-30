package jobmap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	xhtml "golang.org/x/net/html"
)

// PublicCareerClient is used only for operator-configured career pages. DNS
// checks apply to the actual connection, and redirects cannot change hosts.
func PublicCareerClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil || port != "443" {
			return nil, fmt.Errorf("only public HTTPS career pages are supported")
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !publicIP(ip.IP) {
				return nil, fmt.Errorf("private career-page destination refused")
			}
		}
		var dial net.Dialer
		for _, ip := range ips {
			if conn, err := dial.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port)); err == nil {
				return conn, nil
			}
		}
		return nil, fmt.Errorf("career page is unavailable")
	}
	return &http.Client{Transport: transport, Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 || req.URL.Scheme != "https" || req.URL.Hostname() != via[0].URL.Hostname() {
			return fmt.Errorf("career page redirect refused")
		}
		return nil
	}}
}

func publicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() && !ip.IsUnspecified()
}

func (c *Client) careerPage(ctx context.Context, company Company) ([]Job, error) {
	u, _ := url.Parse(company.CareersURL)
	client := PublicCareerClient()
	// The operator explicitly selects each page; we do not crawl arbitrary links.
	get := func(raw string) ([]byte, int, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", "ResumeBuilder-JobMap/1.0")
		resp, err := client.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("career page request failed")
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		return data, resp.StatusCode, err
	}
	robots, status, err := get(u.Scheme + "://" + u.Host + "/robots.txt")
	if err != nil || (status != 200 && status != 404) {
		return nil, fmt.Errorf("could not check career page robots policy")
	}
	if status == 200 && !RobotsAllows(string(robots), u.EscapedPath()) {
		return nil, fmt.Errorf("career page disallowed by robots policy")
	}
	data, status, err := get(company.CareersURL)
	if err != nil || status != 200 {
		return nil, fmt.Errorf("career page is unavailable")
	}
	jobs := ParseCareerPage(company, string(data))
	if len(jobs) == 0 {
		return nil, fmt.Errorf("career page contains no usable JobPosting structured data")
	}
	return jobs, nil
}

// RobotsAllows deliberately errs on the side of skipping a page: a matching
// Disallow in any user-agent group blocks collection. No bypass or login flow.
func RobotsAllows(robots, path string) bool {
	for _, line := range strings.Split(robots, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		key, value, ok := strings.Cut(line, ":")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "disallow") {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		pattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(strings.TrimSuffix(value, "$")), `\*`, ".*")
		if strings.HasSuffix(value, "$") {
			pattern += "$"
		}
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return false
		}
	}
	return true
}

func ParseCareerPage(company Company, document string) []Job {
	z := xhtml.NewTokenizer(strings.NewReader(document))
	jobs := []Job{}
	for {
		tt := z.Next()
		if tt == xhtml.ErrorToken {
			break
		}
		if tt != xhtml.StartTagToken {
			continue
		}
		token := z.Token()
		if token.Data != "script" {
			continue
		}
		isLD := false
		for _, a := range token.Attr {
			if a.Key == "type" && strings.EqualFold(a.Val, "application/ld+json") {
				isLD = true
			}
		}
		if !isLD {
			continue
		}
		if z.Next() != xhtml.TextToken {
			continue
		}
		var data any
		if json.Unmarshal(z.Text(), &data) != nil {
			continue
		}
		var walk func(any)
		walk = func(value any) {
			switch v := value.(type) {
			case []any:
				for _, item := range v {
					walk(item)
				}
			case map[string]any:
				types, _ := json.Marshal(v["@type"])
				if strings.Contains(string(types), `"JobPosting"`) {
					if job, ok := schemaJob(company, v); ok {
						jobs = append(jobs, job)
					}
				}
				if graph, ok := v["@graph"]; ok {
					walk(graph)
				}
			}
		}
		walk(data)
	}
	return jobs
}

func schemaJob(company Company, v map[string]any) (Job, bool) {
	text := func(key string) string { s, _ := v[key].(string); return s }
	rawURL := text("url")
	if rawURL == "" {
		return Job{}, false
	}
	base, _ := url.Parse(company.CareersURL)
	if parsed, err := url.Parse(rawURL); err == nil {
		rawURL = base.ResolveReference(parsed).String()
	}
	if rawURL == "" {
		return Job{}, false
	}
	id := rawURL
	if ident, ok := v["identifier"].(map[string]any); ok {
		if s, ok := ident["value"].(string); ok && s != "" {
			id = s
		}
	}
	job := companyJob(company, id, text("title"), rawURL, rawURL)
	job.Description, job.PostedAt, job.ExpiresAt = text("description"), parseTime(text("datePosted")), parseTime(text("validThrough"))
	job.EmploymentType = Employment(text("employmentType"))
	job.Workplace = Workplace(text("jobLocationType"), nil, "")
	if strings.EqualFold(text("jobLocationType"), "TELECOMMUTE") {
		job.Workplace = "remote"
	}
	list := func(a any) []any {
		if xs, ok := a.([]any); ok {
			return xs
		}
		if a != nil {
			return []any{a}
		}
		return nil
	}
	for _, raw := range list(v["jobLocation"]) {
		loc, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		address, _ := loc["address"].(map[string]any)
		country, _ := address["addressCountry"].(string)
		if x, ok := address["addressCountry"].(map[string]any); ok {
			country, _ = x["name"].(string)
		}
		city, _ := address["addressLocality"].(string)
		region, _ := address["addressRegion"].(string)
		if job.Workplace != "remote" {
			job.Locations = append(job.Locations, ResolveLocation(strings.Trim(strings.Join([]string{city, region, country}, ", "), ", "), country))
		}
	}
	job.Remote = RemotePolicy{Scope: "unspecified", Regions: []string{}, Timezones: []float64{}}
	for _, raw := range list(v["applicantLocationRequirements"]) {
		if loc, ok := raw.(map[string]any); ok {
			if name, ok := loc["name"].(string); ok && name != "" {
				job.Remote.Scope = "restricted"
				job.Remote.Regions = appendUnique(job.Remote.Regions, name)
			}
		}
	}
	return job, true
}
