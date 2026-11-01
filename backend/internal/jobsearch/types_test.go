package jobsearch

import (
	"resume-builder/backend/internal/jobmap"
	"testing"
	"time"
)

func TestLocationAlternativesDoNotRelaxKnownRemoteRestrictions(t *testing.T) {
	s := Intent{Version: 1, Branches: []Branch{{Workplace: "remote", Country: "IN"}, {Workplace: "hybrid", Country: "DE", City: "Berlin"}}, IncludeUnknown: true}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		job  jobmap.Job
		want bool
	}{
		{"India remote", jobmap.Job{Workplace: "remote", Remote: jobmap.RemotePolicy{Scope: "restricted", Regions: []string{"India"}, Timezones: []float64{5.5}}}, true},
		{"US remote", jobmap.Job{Workplace: "remote", Remote: jobmap.RemotePolicy{Scope: "restricted", Regions: []string{"United States"}}}, false},
		{"wrong timezone", jobmap.Job{Workplace: "remote", Remote: jobmap.RemotePolicy{Scope: "worldwide", Timezones: []float64{-5}}}, false},
		{"Berlin hybrid", jobmap.Job{Workplace: "hybrid", Locations: []jobmap.Location{{Name: "Berlin", CountryCode: "DE"}}}, true},
		{"Berlin onsite", jobmap.Job{Workplace: "onsite", Locations: []jobmap.Location{{Name: "Berlin", CountryCode: "DE"}}}, false},
		{"unknown remote region", jobmap.Job{Workplace: "remote", Remote: jobmap.RemotePolicy{Scope: "unspecified"}}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := s.Matches(c.job, time.Now(), false); got != c.want {
				t.Fatalf("match=%v want %v", got, c.want)
			}
		})
	}
	s.IncludeUnknown = false
	if s.Matches(cases[5].job, time.Now(), false) {
		t.Fatal("unknown regions must be opt-in")
	}
}

func TestOfficeCountryNeverBecomesRemoteEligibility(t *testing.T) {
	s := Intent{Version: 1, Branches: []Branch{{Workplace: "all", Country: "IN"}}}
	if s.Matches(jobmap.Job{Workplace: "remote", Remote: jobmap.RemotePolicy{Scope: "worldwide"}}, time.Now(), false) {
		t.Fatal("office country admitted an unrelated remote job")
	}
	if !s.Matches(jobmap.Job{Workplace: "onsite", Locations: []jobmap.Location{{CountryCode: "IN"}}}, time.Now(), false) {
		t.Fatal("office country did not match")
	}
}

func TestInvalidIntentAndClosedListings(t *testing.T) {
	for _, s := range []Intent{{Version: 2, Branches: []Branch{{Workplace: "remote"}}}, {Version: 1, Branches: []Branch{{Workplace: "remote", City: "Mumbai", Country: "IN"}}}, {Version: 1, Branches: []Branch{{Workplace: "hybrid", City: "Berlin"}}}, {Version: 1, Branches: []Branch{{Workplace: "remote", Country: "not-a-country"}}}} {
		if s.Validate() == nil {
			t.Fatal("invalid intent accepted", s)
		}
	}
	s := Intent{Version: 1, Branches: []Branch{{Workplace: "all"}}, ExcludeCompanies: []string{"Acme"}}
	if s.Matches(jobmap.Job{Company: "Acme Labs"}, time.Now(), false) {
		t.Fatal("excluded company accepted")
	}
	j := jobmap.Job{Closed: true}
	if s.Matches(j, time.Now(), false) || !s.Matches(j, time.Now(), true) {
		t.Fatal("closed jobs should only survive in saved results")
	}
}
