package jobmap

import (
	"net/url"
	"sort"
	"strings"
	"time"
)

// ExternalSearch scopes supplemental searches to a chosen country or focused
// map area. Remote searches use the independent remote-work preference.
func ExternalSearch(s Search) (string, string) {
	query, code := strings.TrimSpace(s.Query), s.Country
	if query == "" {
		query = "technology jobs"
	}
	if s.Workplace == "remote" {
		code = s.RemoteCountry
		if country, ok := FindCountry(code); ok {
			return query + " remote in " + country.Name, country.Code
		}
		return "", ""
	}
	if s.Bounds != nil {
		b := *s.Bounds
		if b[2] > b[0] && b[2]-b[0] < 20 && b[3]-b[1] < 20 {
			if p := NearestPlace((b[1]+b[3])/2, (b[0]+b[2])/2); p != nil && (code == "" || p.CountryCode == code) {
				return query + " in " + p.Name + ", " + p.Country, p.CountryCode
			}
		}
	}
	if country, ok := FindCountry(code); ok {
		return query + " in " + country.Name, country.Code
	}
	return "", "" // never silently query the provider's default US inventory
}

func InBounds(l Location, b [4]float64) bool {
	if l.Latitude == nil || l.Longitude == nil {
		return false
	}
	lat, lng := *l.Latitude, *l.Longitude
	inLongitude := lng >= b[0] && lng <= b[2]
	if b[0] > b[2] {
		inLongitude = lng >= b[0] || lng <= b[2]
	}
	return lat >= b[1] && lat <= b[3] && inLongitude
}

func FilterJobs(jobs []Job, s Search, now time.Time) SearchResult {
	filtered := []Job{}
	query := strings.ToLower(strings.TrimSpace(s.Query))
	for _, j := range jobs {
		if j.Closed && !s.SavedOnly {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(j.Title + " " + j.Company + " " + j.Description)
			matched := true
			for _, word := range strings.Fields(query) {
				if !strings.Contains(haystack, word) {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
		}
		if s.Workplace != "" && s.Workplace != "all" && j.Workplace != s.Workplace {
			continue
		}
		if s.EmploymentType != "" && j.EmploymentType != s.EmploymentType {
			continue
		}
		if s.Seniority != "" && j.Seniority != s.Seniority {
			continue
		}
		if s.SourceKind != "" && j.SourceKind != s.SourceKind {
			continue
		}
		if date := j.ListingDate(); s.PostedDays > 0 && (date == nil || date.Before(now.Add(-time.Duration(s.PostedDays)*24*time.Hour))) {
			continue
		}
		if j.Workplace == "remote" {
			if !RemoteMatches(j.Remote, s.RemoteCountry, s.IncludeUnknown) {
				continue
			}
		} else {
			if s.Country != "" || s.Bounds != nil {
				matched := false
				for _, l := range j.Locations {
					if s.Country != "" && l.CountryCode != s.Country {
						continue
					}
					if s.Bounds != nil && !InBounds(l, *s.Bounds) {
						continue
					}
					matched = true
					break
				}
				if !matched {
					continue
				}
			}
		}
		filtered = append(filtered, j)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		// Show nearby workplaces before remote roles in an area search. Remote
		// eligibility still uses its own country preference and never the viewport.
		if s.Bounds != nil || s.Country != "" {
			aRemote, bRemote := filtered[i].Workplace == "remote", filtered[j].Workplace == "remote"
			if aRemote != bRemote {
				return !aRemote
			}
		}
		a, b := filtered[i].ListingDate(), filtered[j].ListingDate()
		if a == nil && b != nil {
			return false
		}
		if b == nil && a != nil {
			return true
		}
		if a != nil && b != nil && !a.Equal(*b) {
			return a.After(*b)
		}
		return filtered[i].ID < filtered[j].ID
	})
	// Deduplicate only identical original application URLs, never company+title.
	// A source's unique ID remains stable; an already saved version wins its group.
	seen := map[string]int{}
	dedup := []Job{}
	for _, j := range filtered {
		key := ApplicationKey(j.ApplyURL)
		if index, ok := seen[key]; ok && key != "" {
			if j.Saved && !dedup[index].Saved {
				dedup[index] = j
			}
			continue
		}
		if key != "" {
			seen[key] = len(dedup)
		}
		dedup = append(dedup, j)
	}
	pageSize := s.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	pageSize = min(pageSize, 100)
	page := max(1, s.Page)
	start := min((page-1)*pageSize, len(dedup))
	end := min(start+pageSize, len(dedup))
	result := SearchResult{Jobs: dedup[start:end], MapJobs: []MapJob{}, Total: len(dedup), Page: page, PageSize: pageSize, HasMore: end < len(dedup), Sources: []SourceStatus{}}
	for _, j := range dedup {
		if j.Workplace == "remote" {
			result.RemoteCount++
			continue
		}
		if j.Closed {
			continue
		}
		locations := []Location{}
		for _, l := range j.Locations {
			if l.Latitude == nil || l.Longitude == nil || (s.Country != "" && l.CountryCode != s.Country) || (s.Bounds != nil && !InBounds(l, *s.Bounds)) {
				continue
			}
			locations = append(locations, l)
		}
		if len(locations) > 0 {
			result.MapJobs = append(result.MapJobs, MapJob{ID: j.ID, Title: j.Title, Company: j.Company, Workplace: j.Workplace, Locations: locations})
		}
	}
	result.Mapped = len(result.MapJobs)
	return result
}

func ApplicationKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Fragment = ""
	query := u.Query()
	for k := range query {
		if strings.HasPrefix(strings.ToLower(k), "utm_") {
			query.Del(k)
		}
	}
	u.RawQuery = query.Encode()
	return u.String()
}
