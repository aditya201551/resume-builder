package jobmap

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestHimalayasSearchPageAndSourceDate(t *testing.T) {
	c := NewClient(Config{HimalayasKey: "test-key"})
	c.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/jobs/api/search" || r.URL.Query().Get("country") != "IN" || r.URL.Query().Get("page") != "2" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected feed request: %s", r.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"jobs":[{"guid":"https://himalayas.app/companies/acme/jobs/a","title":"Software Engineer","companyName":"Acme","pubDate":1700000000,"locationRestrictions":[],"applicationLink":"https://example.com/apply?ref=himalayas"}]}`))}, nil
	})
	jobs, more, err := c.SearchHimalayasPage(context.Background(), "engineer", "IN", 2)
	if err != nil || len(jobs) != 1 || !more {
		t.Fatalf("page: %+v %v %v", jobs, more, err)
	}
	if jobs[0].PostedAt != nil || jobs[0].SourceUpdatedAt == nil || jobs[0].SourceUpdatedAt.Unix() != 1700000000 {
		t.Fatal("provider update was represented as a posting date")
	}
	if jobs[0].ApplyURL != "https://example.com/apply?ref=himalayas" {
		t.Fatal("application attribution changed")
	}
}

func TestHimalayasSearchRejectsMalformedPages(t *testing.T) {
	for _, body := range []string{`{}`, `{"jobs":null}`} {
		c := NewClient(Config{})
		c.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
		})
		if _, _, err := c.SearchHimalayasPage(context.Background(), "", "", 1); err == nil {
			t.Fatalf("accepted invalid feed: %s", body)
		}
	}
}

func TestHimalayasStatusClosureAndTransientErrors(t *testing.T) {
	guid := "https://himalayas.app/companies/acme/jobs/engineer"
	for _, code := range []int{200, 404, 410, 429, 503} {
		c := NewClient(Config{})
		c.HTTP.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://himalayas.app/jobs/api/companies/acme/jobs/engineer" {
				t.Fatal(r.URL)
			}
			return &http.Response{StatusCode: code, Header: http.Header{"Retry-After": []string{"45"}}, Body: io.NopCloser(strings.NewReader(`{"job":{"guid":"` + guid + `","companyName":"Acme","title":"Software Engineer"}}`))}, nil
		})
		_, closed, err := c.FetchHimalayasJob(context.Background(), guid)
		if closed != (code == 410) || (err != nil) != (code != 200 && code != 410) {
			t.Fatalf("HTTP %d: closed=%v err=%v", code, closed, err)
		}
		if code == 429 {
			var response *FeedHTTPError
			if !errors.As(err, &response) || response.RetryAfter != 45*time.Second {
				t.Fatal("lost retry delay")
			}
		}
	}
}

func TestHimalayasStatusNeverFetchesArbitraryURLs(t *testing.T) {
	c := NewClient(Config{})
	c.HTTP.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unsafe identity caused network request")
		return nil, nil
	})
	for _, guid := range []string{"http://himalayas.app/companies/acme/jobs/a", "https://example.com/companies/acme/jobs/a", "https://himalayas.app@localhost/companies/acme/jobs/a", "https://himalayas.app/companies/acme/jobs/../a", "https://himalayas.app/other", "https://himalayas.app/companies/acme/jobs/a?redirect=1"} {
		if _, _, err := c.FetchHimalayasJob(context.Background(), guid); err == nil {
			t.Fatal("accepted", guid)
		}
	}
}
