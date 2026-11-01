// Package jobsearch defines the shared, provider-independent contract used by
// manual discovery and the job assistant. It has no HTTP or persistence concerns.
package jobsearch

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"resume-builder/backend/internal/jobmap"
)

type Branch struct {
	Workplace string `json:"workplace"` // all, remote, onsite, hybrid, unknown
	Country   string `json:"country"`   // remote work country OR office country, by arrangement
	City      string `json:"city"`      // office only
}

type Intent struct {
	Version          int      `json:"version"`
	Query            string   `json:"query"`
	Branches         []Branch `json:"branches"` // alternatives (OR), never cumulative requirements
	EmploymentType   string   `json:"employmentType"`
	Seniority        string   `json:"seniority"`
	PostedDays       int      `json:"postedDays"`
	ExcludeCompanies []string `json:"excludeCompanies"`
	IncludeUnknown   bool     `json:"includeUnknown"`
}

func (s *Intent) Validate() error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported search version")
	}
	s.Query = strings.TrimSpace(s.Query)
	if len([]rune(s.Query)) > 160 || len(s.Branches) < 1 || len(s.Branches) > 4 {
		return fmt.Errorf("use up to four location alternatives and 160 search characters")
	}
	for i := range s.Branches {
		b := &s.Branches[i]
		if !oneOf(b.Workplace, "all", "remote", "onsite", "hybrid", "unknown") {
			return fmt.Errorf("invalid work arrangement")
		}
		b.City = strings.TrimSpace(b.City)
		if len([]rune(b.City)) > 100 || b.Workplace == "remote" && b.City != "" {
			return fmt.Errorf("cities apply to office locations, not remote eligibility")
		}
		if b.Country != "" {
			c, ok := jobmap.FindCountry(b.Country)
			if !ok {
				return fmt.Errorf("unknown country")
			}
			b.Country = c.Code
		}
		if b.City != "" && b.Country == "" {
			return fmt.Errorf("choose an office country with the city")
		}
	}
	if !oneOf(s.EmploymentType, "", "full_time", "part_time", "contract", "internship", "unknown") || !oneOf(s.Seniority, "", "junior", "mid", "senior", "lead", "unknown") || s.PostedDays < 0 || s.PostedDays > 365 {
		return fmt.Errorf("invalid job filters")
	}
	if len(s.ExcludeCompanies) > 10 {
		return fmt.Errorf("exclude up to ten companies")
	}
	for i := range s.ExcludeCompanies {
		s.ExcludeCompanies[i] = strings.TrimSpace(s.ExcludeCompanies[i])
		if s.ExcludeCompanies[i] == "" || len([]rune(s.ExcludeCompanies[i])) > 100 {
			return fmt.Errorf("invalid excluded company")
		}
	}
	if s.ExcludeCompanies == nil {
		s.ExcludeCompanies = []string{}
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

func (s Intent) Key() string {
	b, _ := json.Marshal(s)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:16])
}

func (s Intent) Matches(j jobmap.Job, now time.Time, saved bool) bool {
	if !saved && (j.Closed || j.ExpiresAt != nil && !j.ExpiresAt.After(now)) {
		return false
	}
	for _, company := range s.ExcludeCompanies {
		if strings.Contains(strings.ToLower(j.Company), strings.ToLower(company)) {
			return false
		}
	}
	if s.EmploymentType != "" && j.EmploymentType != s.EmploymentType || s.Seniority != "" && j.Seniority != s.Seniority {
		return false
	}
	if s.PostedDays > 0 {
		d := j.ListingDate()
		if d == nil || d.Before(now.Add(-time.Duration(s.PostedDays)*24*time.Hour)) {
			return false
		}
	}
	// Providers interpret keywords; applying an AND of every word here would
	// discard valid semantic matches. Saved jobs use a local keyword filter.
	if saved {
		for _, word := range strings.Fields(strings.ToLower(s.Query)) {
			if !strings.Contains(strings.ToLower(j.Title+" "+j.Company+" "+j.Description), word) {
				return false
			}
		}
	}
	for _, b := range s.Branches {
		if b.Workplace != "all" && b.Workplace != j.Workplace {
			continue
		}
		if j.Workplace == "remote" {
			// An all-arrangements office location must not silently constrain remote jobs.
			if b.Workplace == "all" && b.Country != "" {
				continue
			}
			if jobmap.RemoteMatches(j.Remote, b.Country, s.IncludeUnknown) {
				return true
			}
			continue
		}
		if b.Country == "" && b.City == "" {
			return true
		}
		for _, l := range j.Locations {
			if (b.Country == "" || l.CountryCode == b.Country) && (b.City == "" || strings.Contains(strings.ToLower(l.Name), strings.ToLower(b.City))) {
				return true
			}
		}
	}
	return false
}

type Request struct {
	Intent    Intent `json:"intent"`
	NextToken string `json:"nextToken,omitempty"`
	SavedOnly bool   `json:"savedOnly"`
}

type Source struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	Message   string     `json:"message,omitempty"`
	FetchedAt *time.Time `json:"fetchedAt,omitempty"`
}

type Result struct {
	Jobs      []jobmap.Job `json:"jobs"`
	Sources   []Source     `json:"sources"`
	NextToken string       `json:"nextToken,omitempty"`
	Retry     bool         `json:"retry"`
}

// Page holds only public provider results. User bookmarks are joined on read.
type Page struct {
	Jobs      []jobmap.Job
	Cursor    string
	HasMore   bool
	FetchedAt time.Time
}

type Refinement struct {
	Intent  Intent `json:"intent"`
	Message string `json:"message"`
	Apply   bool   `json:"apply"`
}

type Turn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}
