package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestJobSearchRejectsInvalidFiltersAndBounds(t *testing.T) {
	for _, query := range []string{"bounds=NaN,0,1,2", "bounds=0,4,1,2", "bounds=0,0,181,2", "pageSize=0", "pageSize=101", "page=-1", "days=999999", "country=not-a-country", "workplace=teleport", "source=internal"} {
		if _, err := ParseJobSearch(httptest.NewRequest("GET", "/api/jobs?"+query, nil)); err == nil {
			t.Fatalf("accepted %s", query)
		}
	}
	s, err := ParseJobSearch(httptest.NewRequest("GET", "/api/jobs?country=DE&remoteCountry=IN&bounds=170,-10,-170,10&saved=true", nil))
	if err != nil || s.Country != "DE" || s.RemoteCountry != "IN" || !s.SavedOnly || s.Bounds == nil {
		t.Fatalf("%+v %v", s, err)
	}
}
