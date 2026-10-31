package jobmap

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestRemoteRestrictionsAreIndependentOfMapArea(t *testing.T) {
	now := time.Now()
	city := ResolveLocation("Berlin, Germany", "DE")
	jobs := []Job{
		{ID: "berlin", Title: "Backend Engineer", ApplyURL: "https://example.com/berlin", Workplace: "hybrid", Locations: []Location{city}},
		{ID: "india", Title: "Backend Engineer", ApplyURL: "https://example.com/india", Workplace: "remote", Remote: RemotePolicy{Scope: "restricted", Regions: []string{"India"}}},
		{ID: "us", Title: "Backend Engineer", ApplyURL: "https://example.com/us", Workplace: "remote", Remote: RemotePolicy{Scope: "restricted", Regions: []string{"United States"}}},
		{ID: "world", Title: "Backend Engineer", ApplyURL: "https://example.com/world", Workplace: "remote", Remote: RemotePolicy{Scope: "worldwide"}},
		{ID: "unknown", Title: "Backend Engineer", ApplyURL: "https://example.com/unknown", Workplace: "remote", Remote: RemotePolicy{Scope: "unspecified"}},
	}
	search := Search{Country: "DE", RemoteCountry: "IN", Bounds: &[4]float64{12, 51, 14, 54}}
	result := FilterJobs(jobs, search, now)
	if result.Total != 3 || result.Mapped != 1 || result.RemoteCount != 2 {
		t.Fatalf("expected Berlin + India + worldwide, got %+v", result)
	}
	for _, j := range result.Jobs {
		if j.ID == "us" || j.ID == "unknown" {
			t.Fatalf("ineligible or unknown role leaked: %s", j.ID)
		}
	}
	search.IncludeUnknown = true
	if got := FilterJobs(jobs, search, now).Total; got != 4 {
		t.Fatalf("unknown opt-in: %d", got)
	}
	search.Workplace = "remote"
	if got := FilterJobs(jobs, search, now).Mapped; got != 0 {
		t.Fatalf("remote jobs must not be mapped")
	}
}

func TestSupplementalSearchFollowsAreaButKeepsRemotePreference(t *testing.T) {
	area := &[4]float64{13, 52, 14, 53}
	q, country := ExternalSearch(Search{Query: "backend", Bounds: area})
	if country != "DE" || !strings.Contains(q, "Germany") {
		t.Fatalf("focused map search was not scoped: %s %s", q, country)
	}
	q, country = ExternalSearch(Search{Query: "backend", Workplace: "remote", RemoteCountry: "IN", Country: "DE", Bounds: area})
	if country != "IN" || !strings.Contains(q, "India") || strings.Contains(q, "Germany") {
		t.Fatalf("remote search inherited workplace area: %s %s", q, country)
	}
	if q, _ := ExternalSearch(Search{Query: "backend"}); q != "" {
		t.Fatal("unscoped search silently used a country")
	}
}

func TestSupplementalSearchDiscoversTechJobsWithoutKeywords(t *testing.T) {
	for _, test := range []struct {
		name     string
		search   Search
		country  string
		contains string
	}{
		{"city", Search{Bounds: &[4]float64{77.3, 12.6, 77.9, 13.3}}, "IN", "technology jobs in Bengaluru"},
		{"regional map", Search{Bounds: &[4]float64{73, 8, 83, 18}}, "IN", "technology jobs in"},
		{"country", Search{Country: "IN"}, "IN", "technology jobs in India"},
		{"remote country", Search{Workplace: "remote", RemoteCountry: "IN"}, "IN", "technology jobs remote in India"},
		{"global", Search{}, "", ""},
		{"world map", Search{Bounds: &[4]float64{-180, -80, 180, 80}}, "", ""},
		{"remote without country", Search{Workplace: "remote", Bounds: &[4]float64{77.3, 12.6, 77.9, 13.3}}, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			query, country := ExternalSearch(test.search)
			if country != test.country || (test.contains == "" && query != "") || (test.contains != "" && !strings.Contains(query, test.contains)) {
				t.Fatalf("unexpected scoped query: %q country=%q", query, country)
			}
		})
	}
}

func TestMapJobsCoverAllPagesAndOnlyMatchingWorkplaces(t *testing.T) {
	berlin, tokyo := ResolveLocation("Berlin", "DE"), ResolveLocation("Tokyo", "JP")
	jobs := []Job{}
	for i := 0; i < 55; i++ {
		id := fmt.Sprintf("office-%02d", i)
		jobs = append(jobs, Job{ID: id, Title: "Software Engineer", Company: "Example", ApplyURL: "https://example.com/" + id, Workplace: "hybrid", Locations: []Location{berlin, tokyo}})
	}
	for i := 0; i < 60; i++ {
		id := fmt.Sprintf("remote-%02d", i)
		posted := time.Now()
		jobs = append(jobs, Job{ID: id, Title: "Software Engineer", ApplyURL: "https://example.com/" + id, Workplace: "remote", PostedAt: &posted, Locations: []Location{berlin}, Remote: RemotePolicy{Scope: "worldwide"}})
	}
	jobs = append(jobs, Job{ID: "closed", ApplyURL: "https://example.com/closed", Closed: true, Workplace: "hybrid", Locations: []Location{berlin}})
	jobs = append(jobs, Job{ID: "outside", ApplyURL: "https://example.com/outside", Workplace: "hybrid", Locations: []Location{tokyo}})
	search := Search{Bounds: &[4]float64{12, 51, 14, 54}, PageSize: 50}
	first := FilterJobs(jobs, search, time.Now())
	if len(first.Jobs) != 50 || first.Mapped != 55 || len(first.MapJobs) != 55 || first.RemoteCount != 60 || first.Total != 115 || !first.HasMore {
		t.Fatalf("page one should map every matching workplace: %+v", first)
	}
	for _, job := range first.Jobs {
		if job.Workplace == "remote" {
			t.Fatal("remote roles pushed nearby workplaces out of page one")
		}
	}
	for _, job := range first.MapJobs {
		if len(job.Locations) != 1 || job.Locations[0].CountryCode != "DE" {
			t.Fatal("outside secondary office leaked onto map")
		}
	}
	search.Page = 2
	second := FilterJobs(jobs, search, time.Now())
	if len(second.Jobs) != 50 || len(second.MapJobs) != 55 || second.MapJobs[0].ID != first.MapJobs[0].ID || second.RemoteCount != 60 {
		t.Fatal("list pagination changed map coverage or full-result totals")
	}
	search.SavedOnly = true
	saved := FilterJobs(jobs, search, time.Now())
	if saved.Mapped != 55 {
		t.Fatal("closed saved jobs should never gain map pins")
	}
	search.Workplace = "remote"
	if remote := FilterJobs(jobs, search, time.Now()); len(remote.MapJobs) != 0 || remote.Mapped != 0 || remote.RemoteCount != 60 {
		t.Fatal("remote roles acquired map pins")
	}
}

func TestRemotePolicyDoesNotInventWorldwide(t *testing.T) {
	for _, name := range []string{"Remote", "", "Distributed"} {
		if got := RemoteFromLocations([]string{name}); got.Scope != "unspecified" {
			t.Fatalf("%q: %+v", name, got)
		}
	}
	if p := RemoteFromLocations([]string{"Remote, Global"}); p.Scope != "worldwide" {
		t.Fatal(p)
	}
	if p := RemoteFromLocations([]string{"San Francisco"}); p.Scope != "unspecified" {
		t.Fatal("an office city cannot establish remote applicant restrictions", p)
	}
	if p := RemoteFromLocations([]string{"Remote, United States"}); p.Scope != "restricted" || RemoteMatches(p, "IN", false) {
		t.Fatal(p)
	}
	if !RemoteMatches(RemotePolicy{Scope: "restricted", Regions: []string{"European Union"}}, "DE", false) {
		t.Fatal("EU should include Germany")
	}
	if RemoteMatches(RemotePolicy{Scope: "restricted", Regions: []string{"European Union"}}, "GB", false) {
		t.Fatal("EU must not include UK")
	}
	policy := RemotePolicy{Scope: "worldwide", Timezones: []float64{5.5}}
	if !RemoteMatches(policy, "IN", false) || RemoteMatches(policy, "US", false) {
		t.Fatal("worldwide geography must still respect source timezone limits")
	}
}

func TestOfflineCityResolutionAndUnknownLocations(t *testing.T) {
	if p := NearestPlace(12.9716, 77.5946); p == nil || p.CountryCode != "IN" || (p.Name != "Bengaluru" && p.Name != "Bangalore") {
		t.Fatalf("Bengaluru device coordinates resolved to %+v", p)
	}
	for _, test := range []struct{ name, hint, code string }{{"Bengaluru, India", "IN", "IN"}, {"Berlin", "DE", "DE"}, {"San Francisco, California", "USA", "US"}, {"São Paulo", "BR", "BR"}, {"Tokyo", "JP", "JP"}} {
		l := ResolveLocation(test.name, test.hint)
		if l.CountryCode != test.code || l.Latitude == nil || l.Precision != "city" {
			t.Fatalf("%s: %+v", test.name, l)
		}
	}
	if l := ResolveLocation("Remote, India", "IN"); l.Latitude != nil {
		t.Fatalf("remote office invented: %+v", l)
	}
	if l := ResolveLocation("Unspecified office", "US"); l.Latitude != nil || l.Precision != "unknown" {
		t.Fatalf("unknown location guessed: %+v", l)
	}
	if p := SearchPlaces("Berlin", 4); len(p) == 0 || p[0].CountryCode != "DE" {
		t.Fatal(p)
	}
}

func TestDatelineAndPaginationCounts(t *testing.T) {
	lat, lng := 0.0, 179.0
	l := Location{Latitude: &lat, Longitude: &lng}
	if !InBounds(l, [4]float64{170, -10, -170, 10}) {
		t.Fatal("dateline search missed east side")
	}
	lng = -179
	if !InBounds(l, [4]float64{170, -10, -170, 10}) {
		t.Fatal("dateline search missed west side")
	}
	lng = 0
	if InBounds(l, [4]float64{170, -10, -170, 10}) {
		t.Fatal("dateline search included Greenwich")
	}
	jobs := []Job{{ID: "a", ApplyURL: "https://example.com/a?utm_source=foo"}, {ID: "b", ApplyURL: "https://example.com/a?utm_source=bar", Saved: true}, {ID: "c", ApplyURL: "https://example.com/c"}}
	r := FilterJobs(jobs, Search{PageSize: 1}, time.Now())
	if r.Total != 2 || len(r.Jobs) != 1 || !r.HasMore || r.Jobs[0].ID != "b" {
		t.Fatalf("dedup/page totals: %+v", r)
	}
	r = FilterJobs(jobs, Search{Page: 2, PageSize: 1}, time.Now())
	if len(r.Jobs) != 1 || r.HasMore {
		t.Fatal(r)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAshbyHybridOverridesRemoteFlagAndKeepsSecondaryLocations(t *testing.T) {
	c := NewClient(Config{})
	c.HTTP = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"jobs":[{"id":"role","title":"Software Engineer","isListed":true,"isRemote":true,"workplaceType":"Hybrid","location":"San Francisco","address":{"postalAddress":{"addressCountry":"USA"}},"secondaryLocations":[{"location":"New York","address":{"postalAddress":{"addressCountry":"USA"}}}],"jobUrl":"https://jobs.ashbyhq.com/example/role","applyUrl":"https://jobs.ashbyhq.com/example/role/application"}]}`))}, nil
	})}
	jobs, err := c.FetchCompany(context.Background(), Company{Name: "Example", Provider: "ashby", Board: "example", Website: "https://example.com"})
	if err != nil || len(jobs) != 1 {
		t.Fatalf("%v %+v", err, jobs)
	}
	if jobs[0].Workplace != "hybrid" || len(jobs[0].Locations) != 2 {
		t.Fatalf("%+v", jobs[0])
	}
	c.HTTP = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}
	if _, err := c.FetchCompany(context.Background(), Company{Provider: "ashby", Board: "example"}); err == nil {
		t.Fatal("invalid empty object would close the entire source")
	}
}

func TestHimalayasExplicitEmptyRestrictionsDifferFromMissing(t *testing.T) {
	c := NewClient(Config{})
	c.HTTP = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := `{"jobs":[{"guid":"https://himalayas.app/companies/example/jobs/a","title":"Software Engineer","companyName":"Example","applicationLink":"https://himalayas.app/companies/example/jobs/a?utm_source=api","locationRestrictions":[],"timezoneRestrictions":[5.5]},{"guid":"https://himalayas.app/companies/example/jobs/b","title":"Software Engineer","companyName":"Example","applicationLink":"https://himalayas.app/companies/example/jobs/b?utm_source=api"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	jobs, _, err := c.FetchHimalayasPage(context.Background(), "")
	if err != nil || len(jobs) != 2 {
		t.Fatalf("%v %+v", err, jobs)
	}
	if jobs[0].Remote.Scope != "worldwide" || jobs[1].Remote.Scope != "unspecified" {
		t.Fatal("provider's explicit empty and missing restrictions were conflated")
	}
	if !RemoteMatches(jobs[0].Remote, "IN", false) || RemoteMatches(jobs[0].Remote, "US", false) {
		t.Fatal("source timezone restriction was lost")
	}
	if jobs[0].ApplyURL != "https://himalayas.app/companies/example/jobs/a?utm_source=api" {
		t.Fatal("original application URL changed")
	}
}

func TestSafeProviderContentAndTechScope(t *testing.T) {
	if s := PlainText(`<p>Build <b>software</b></p><script>alert(1)</script><p>Next &amp; more</p>`); strings.Contains(s, "alert") || !strings.Contains(s, "Next & more") {
		t.Fatal(s)
	}
	for _, s := range []string{"javascript:alert(1)", "data:text/html,test", "https://user:pass@example.com"} {
		if SafeURL(s) != "" {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"Senior Software Engineer", "Engineering Manager", "AI Platform Engineer", "Product Designer", "QA Engineer"} {
		if !IsTech(s) {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"Account Executive", "Mechanical Engineer", "Medical Writer"} {
		if IsTech(s) {
			t.Fatal(s)
		}
	}
}

func TestCareerPageJSONLDAndRobots(t *testing.T) {
	company := Company{Name: "Example", Provider: "career_page", Website: "https://example.com", CareersURL: "https://example.com/careers"}
	document := `<script type="application/ld+json">{"@graph":[{"@type":"JobPosting","title":"Software Engineer","url":"/careers/123","jobLocationType":"TELECOMMUTE","description":"&lt;p&gt;Build things&lt;/p&gt;","applicantLocationRequirements":{"@type":"Country","name":"India"}}]}</script>`
	jobs := ParseCareerPage(company, document)
	if len(jobs) != 1 || jobs[0].Workplace != "remote" || jobs[0].ApplyURL != "https://example.com/careers/123" || jobs[0].Remote.Scope != "restricted" {
		t.Fatal(jobs)
	}
	if RobotsAllows("User-agent: *\nDisallow: /careers", "/careers/123") {
		t.Fatal("robots disallow ignored")
	}
	if !RobotsAllows("User-agent: *\nDisallow: /admin", "/careers/123") {
		t.Fatal("public careers blocked")
	}
	for _, ip := range []string{"127.0.0.1", "10.0.0.1", "169.254.169.254", "::1", "fc00::1"} {
		if publicIP(net.ParseIP(ip)) {
			t.Fatal("private address allowed", ip)
		}
	}
}
