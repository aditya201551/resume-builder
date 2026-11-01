// Package jobmap owns provider-neutral job data and public-feed adapters.
// HTTP handlers, user ownership and persistence belong to the app's existing layers.
package jobmap

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"time"
)

type Location struct {
	Name        string   `json:"name"`
	Country     string   `json:"country"`
	CountryCode string   `json:"countryCode"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	Precision   string   `json:"precision"`
	Provenance  string   `json:"provenance,omitempty"`
}

type RemotePolicy struct {
	Scope     string    `json:"scope"` // worldwide, restricted, unspecified
	Regions   []string  `json:"regions"`
	Timezones []float64 `json:"timezones"` // source-provided UTC offsets, not inferred work hours
}

type Job struct {
	ID              string       `json:"id"`
	ProviderID      string       `json:"providerId"`
	Source          string       `json:"source"`
	SourceName      string       `json:"sourceName"`
	SourceKind      string       `json:"sourceKind"` // company, aggregator, career_page
	SourceURL       string       `json:"sourceUrl"`
	ApplyURL        string       `json:"applyUrl"`
	Company         string       `json:"company"`
	CompanyURL      string       `json:"companyUrl,omitempty"`
	Title           string       `json:"title"`
	Description     string       `json:"description"`
	Workplace       string       `json:"workplace"` // onsite, hybrid, remote, unknown
	Locations       []Location   `json:"locations"`
	Remote          RemotePolicy `json:"remote"`
	EmploymentType  string       `json:"employmentType"`
	Seniority       string       `json:"seniority"`
	Salary          string       `json:"salary,omitempty"`
	Language        string       `json:"language,omitempty"`
	PostedAt        *time.Time   `json:"postedAt,omitempty"`
	SourceUpdatedAt *time.Time   `json:"sourceUpdatedAt,omitempty"`
	ExpiresAt       *time.Time   `json:"expiresAt,omitempty"`
	FetchedAt       time.Time    `json:"fetchedAt"`
	CheckedAt       *time.Time   `json:"checkedAt,omitempty"`
	CheckFailed     bool         `json:"checkFailed,omitempty"`
	Closed          bool         `json:"closed"`
	Saved           bool         `json:"saved"`
}

// ListingDate keeps source updates distinct from original publication dates.
func (j Job) ListingDate() *time.Time {
	if j.PostedAt != nil {
		return j.PostedAt
	}
	return j.SourceUpdatedAt
}

func StableID(source, id string) string {
	h := sha256.Sum256([]byte(source + "\x00" + id))
	return "job_" + hex.EncodeToString(h[:16])
}

// SafeURL permits navigable HTTP links only; provider text is always rendered as text.
func SafeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	return u.String()
}
