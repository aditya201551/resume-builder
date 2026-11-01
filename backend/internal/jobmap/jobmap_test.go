package jobmap

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

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

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestHimalayasExplicitEmptyRestrictionsDifferFromMissing(t *testing.T) {
	c := NewClient(Config{})
	c.HTTP = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		body := `{"jobs":[{"guid":"https://himalayas.app/companies/example/jobs/a","title":"Software Engineer","companyName":"Example","applicationLink":"https://himalayas.app/companies/example/jobs/a?utm_source=api","locationRestrictions":[],"timezoneRestrictions":[5.5]},{"guid":"https://himalayas.app/companies/example/jobs/b","title":"Software Engineer","companyName":"Example","applicationLink":"https://himalayas.app/companies/example/jobs/b?utm_source=api"}]}`
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	jobs, _, err := c.SearchHimalayasPage(context.Background(), "", "", 1)
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
