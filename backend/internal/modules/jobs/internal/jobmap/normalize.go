package jobmap

import (
	"html"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	xhtml "golang.org/x/net/html"
)

var techTitle = regexp.MustCompile(`(?i)\b(engineer|engineering|developer|software|programmer|devops|sre|platform|architect|data scientist|data analyst|machine learning|ai researcher|research scientist|research engineer|cyber|security|qa|quality assurance|test automation|product manager|product designer|ux|ui designer|technical|technology|it support|systems administrator|database|frontend|backend|full.?stack|mobile|ios|android|embedded|firmware|computer vision|ml ops|mlops|analytics engineer)\b`)
var nonTechTitle = regexp.MustCompile(`(?i)\b(account executive|sales development|recruiter|recruiting|talent acquisition|payroll|accountant|medical writer|civil engineer|mechanical engineer|electrical engineer|manufacturing engineer|construction|facilities|warehouse)\b`)
var whitespace = regexp.MustCompile(`[\t\r ]+`)
var blankLines = regexp.MustCompile(`\n\s*\n\s*\n+`)

func IsTech(title string) bool {
	return techTitle.MatchString(title) && !nonTechTitle.MatchString(title)
}

func PlainText(input string) string {
	// Some ATS feeds HTML-encode the entire document before JSON encoding it.
	input = html.UnescapeString(input)
	z := xhtml.NewTokenizer(strings.NewReader(input))
	var b strings.Builder
	skip := 0
	for {
		switch z.Next() {
		case xhtml.ErrorToken:
			if z.Err() == io.EOF {
				return strings.TrimSpace(blankLines.ReplaceAllString(whitespace.ReplaceAllString(b.String(), " "), "\n\n"))
			}
			return strings.TrimSpace(b.String())
		case xhtml.StartTagToken, xhtml.EndTagToken:
			t := z.Token()
			if t.Data == "script" || t.Data == "style" {
				if t.Type == xhtml.StartTagToken {
					skip++
				} else if skip > 0 {
					skip--
				}
			}
			if skip == 0 {
				switch t.Data {
				case "p", "div", "li", "br", "h1", "h2", "h3", "h4", "ul", "ol":
					b.WriteByte('\n')
				}
			}
		case xhtml.SelfClosingTagToken:
			b.WriteByte('\n')
		case xhtml.TextToken:
			if skip == 0 {
				b.Write(z.Text())
			}
		}
	}
}

func Workplace(explicit string, remote *bool, location string) string {
	text := strings.ToLower(explicit)
	switch {
	case strings.Contains(text, "hybrid"):
		return "hybrid"
	case strings.Contains(text, "remote"):
		return "remote"
	case strings.Contains(text, "on-site") || strings.Contains(text, "onsite") || strings.Contains(text, "on site"):
		return "onsite"
	}
	text = strings.ToLower(location)
	switch {
	case strings.Contains(text, "hybrid"):
		return "hybrid"
	case strings.Contains(text, "remote"):
		return "remote"
	case remote != nil && *remote:
		return "remote"
	case remote != nil && !*remote:
		return "onsite"
	default:
		return "unknown" // an office/city name alone does not establish attendance policy
	}
}

func Employment(raw string) string {
	s := strings.NewReplacer(" ", "", "_", "", "-", "").Replace(strings.ToLower(raw))
	switch {
	case strings.Contains(s, "intern"):
		return "internship"
	case strings.Contains(s, "parttime"):
		return "part_time"
	case strings.Contains(s, "fulltime"):
		return "full_time"
	case strings.Contains(s, "contract") || strings.Contains(s, "freelance"):
		return "contract"
	default:
		return "unknown"
	}
}

func Seniority(raw string) string {
	s := strings.ToLower(raw)
	switch {
	case strings.Contains(s, "staff") || strings.Contains(s, "principal") || strings.Contains(s, "director") || strings.Contains(s, "lead") || strings.Contains(s, "head of"):
		return "lead"
	case strings.Contains(s, "senior") || strings.Contains(s, "sr."):
		return "senior"
	case strings.Contains(s, "junior") || strings.Contains(s, "intern") || strings.Contains(s, "graduate") || strings.Contains(s, "entry"):
		return "junior"
	case strings.Contains(s, "mid"):
		return "mid"
	default:
		return "unknown"
	}
}

func RemoteFromLocations(locations []string) RemotePolicy {
	p := RemotePolicy{Scope: "unspecified", Regions: []string{}, Timezones: []float64{}}
	for _, s := range locations {
		lower := strings.ToLower(s)
		if country, ok := FindCountry(s); ok {
			p.Regions = appendUnique(p.Regions, country.Name)
		}
		if regexp.MustCompile(`\b(worldwide|anywhere|global)\b`).MatchString(lower) {
			p.Scope = "worldwide"
		}
		for _, country := range CountriesInText(s) {
			p.Regions = appendUnique(p.Regions, country.Name)
		}
		// Broad regions remain restrictions; never silently expand Europe to the entire world.
		for _, region := range []string{"Europe", "European Union", "EMEA", "APAC", "LATAM", "North America", "South America", "Asia", "Africa", "Americas"} {
			if containsWord(lower, strings.ToLower(region)) {
				p.Regions = appendUnique(p.Regions, region)
			}
		}
	}
	if len(p.Regions) > 0 {
		p.Scope = "restricted"
	}
	return p
}

func Prepare(job Job, now time.Time) (Job, bool) {
	job.Title = strings.TrimSpace(job.Title)
	job.ApplyURL, job.SourceURL, job.CompanyURL = SafeURL(job.ApplyURL), SafeURL(job.SourceURL), SafeURL(job.CompanyURL)
	if job.ProviderID == "" || job.Source == "" || job.Company == "" || job.ApplyURL == "" || !IsTech(job.Title) {
		return job, false
	}
	if job.ExpiresAt != nil && !job.ExpiresAt.After(now) {
		return job, false
	}
	job.ID = StableID(job.Source, job.ProviderID)
	job.FetchedAt = now
	job.Description = PlainText(job.Description)
	if len(job.Description) > 40000 {
		job.Description = string([]rune(job.Description)[:min(30000, len([]rune(job.Description)))])
	}
	if job.Locations == nil {
		job.Locations = []Location{}
	}
	if job.Remote.Regions == nil {
		job.Remote.Regions = []string{}
	}
	if job.Remote.Timezones == nil {
		job.Remote.Timezones = []float64{}
	}
	if job.Remote.Scope == "" {
		job.Remote.Scope = "unspecified"
	}
	if job.Workplace == "" {
		job.Workplace = "unknown"
	}
	if job.EmploymentType == "" {
		job.EmploymentType = "unknown"
	}
	if job.Seniority == "" {
		job.Seniority = Seniority(job.Title)
	}
	return job, true
}

func appendUnique(xs []string, s string) []string {
	for _, x := range xs {
		if strings.EqualFold(x, s) {
			return xs
		}
	}
	return append(xs, s)
}

var wordPatterns sync.Map

func containsWord(text, word string) bool {
	v, ok := wordPatterns.Load(word)
	if !ok {
		v, _ = wordPatterns.LoadOrStore(word, regexp.MustCompile(`(?i)(^|[^\pL\pN])`+regexp.QuoteMeta(word)+`([^\pL\pN]|$)`))
	}
	return v.(*regexp.Regexp).MatchString(text)
}
