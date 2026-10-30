package jobmap

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed companies.json
var companyData []byte

type Config struct {
	Companies       []Company
	JSearchKey      string
	JSearchProvider string
	JSearchLimit    int
	HimalayasKey    string
	GeoapifyKey     string
	GeoapifyLimit   int
	RefreshInterval time.Duration
}

func LoadConfig() (Config, error) {
	c := Config{JSearchKey: os.Getenv("JSEARCH_API_KEY"), HimalayasKey: os.Getenv("HIMALAYAS_API_KEY"), GeoapifyKey: os.Getenv("GEOAPIFY_API_KEY"), JSearchLimit: 200, GeoapifyLimit: 1000, RefreshInterval: 6 * time.Hour}
	c.JSearchProvider = strings.ToLower(strings.TrimSpace(os.Getenv("JSEARCH_PROVIDER")))
	if c.JSearchProvider == "" {
		c.JSearchProvider = "openwebninja"
	}
	if c.JSearchProvider != "openwebninja" && c.JSearchProvider != "rapidapi" {
		return c, fmt.Errorf("JSEARCH_PROVIDER must be openwebninja or rapidapi")
	}
	data := companyData
	if path := os.Getenv("JOB_MAP_COMPANIES_FILE"); path != "" {
		var err error
		data, err = os.ReadFile(path)
		if err != nil {
			return c, fmt.Errorf("read job map company registry: %w", err)
		}
	}
	if err := json.Unmarshal(data, &c.Companies); err != nil {
		return c, fmt.Errorf("decode job map companies: %w", err)
	}
	if err := ValidateCompanies(c.Companies); err != nil {
		return c, err
	}
	for name, target := range map[string]*int{"JSEARCH_MONTHLY_LIMIT": &c.JSearchLimit, "GEOAPIFY_DAILY_LIMIT": &c.GeoapifyLimit} {
		if raw := os.Getenv(name); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 {
				return c, fmt.Errorf("%s must be a non-negative integer", name)
			}
			*target = n
		}
	}
	// This prototype never opts a free JSearch account into paid volume.
	if c.JSearchLimit > 200 {
		return c, fmt.Errorf("JSEARCH_MONTHLY_LIMIT cannot exceed the prototype's 200 request cap")
	}
	if raw := os.Getenv("JOB_MAP_REFRESH_INTERVAL"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d < time.Hour {
			return c, fmt.Errorf("JOB_MAP_REFRESH_INTERVAL must be a duration of at least 1h")
		}
		c.RefreshInterval = d
	}
	return c, nil
}

func ValidateCompanies(companies []Company) error {
	board := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,99}$`)
	seen := map[string]bool{}
	for _, c := range companies {
		if c.Name == "" || SafeURL(c.Website) == "" {
			return fmt.Errorf("job company needs a name and HTTP website")
		}
		if c.Provider == "career_page" {
			u, err := url.Parse(c.CareersURL)
			if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
				return fmt.Errorf("%s: career page must be an HTTPS URL", c.Name)
			}
		} else if (c.Provider != "ashby" && c.Provider != "greenhouse" && c.Provider != "lever") || !board.MatchString(c.Board) {
			return fmt.Errorf("%s: unsupported provider or invalid board identifier", c.Name)
		}
		key := c.Provider + ":" + c.Board
		if c.Provider == "career_page" {
			key = c.Provider + ":" + c.CareersURL
		}
		if seen[key] {
			return fmt.Errorf("duplicate job source %s", c.Name)
		}
		seen[key] = true
	}
	return nil
}

func CompanySource(c Company) string {
	if c.Provider == "career_page" {
		return "career_page:" + StableID(c.Website, c.CareersURL)
	}
	return c.Provider + ":" + c.Board
}
