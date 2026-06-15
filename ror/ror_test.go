package ror

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockResponse is the JSON returned by mock search/filter endpoints.
const mockSearchResponse = `{
  "number_of_results": 1,
  "time_taken": 5,
  "items": [
    {
      "id": "https://ror.org/00f54p054",
      "names": [
        {"value": "Massachusetts Institute of Technology", "types": ["ror_display", "label"]},
        {"value": "MIT", "types": ["acronym"]}
      ],
      "types": ["Education"],
      "locations": [
        {"geonames_details": {"country_name": "United States", "name": "Cambridge"}}
      ],
      "established": 1861,
      "status": "active",
      "links": [
        {"value": "https://www.mit.edu", "type": "website"}
      ],
      "external_ids": [
        {"type": "wikidata", "all": ["Q49108"]},
        {"type": "grid", "all": ["grid.116068.8"]}
      ]
    }
  ]
}`

const mockOrgResponse = `{
  "id": "https://ror.org/00f54p054",
  "names": [
    {"value": "Massachusetts Institute of Technology", "types": ["ror_display", "label"]},
    {"value": "MIT", "types": ["acronym"]}
  ],
  "types": ["Education"],
  "locations": [
    {"geonames_details": {"country_name": "United States", "name": "Cambridge"}}
  ],
  "established": 1861,
  "status": "active",
  "links": [
    {"value": "https://www.mit.edu", "type": "website"}
  ],
  "external_ids": [
    {"type": "wikidata", "all": ["Q49108"]},
    {"type": "grid", "all": ["grid.116068.8"]}
  ]
}`

func newTestClient(srv *httptest.Server) *Client {
	c := NewClient()
	c.BaseURL = srv.URL
	c.Rate = 0 // no pacing in tests
	return c
}

func TestSearchOrgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/organizations" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query().Get("query")
		if q == "" {
			t.Error("missing query param")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockSearchResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	orgs, total, err := c.SearchOrgs(context.Background(), "MIT", 1)
	if err != nil {
		t.Fatalf("SearchOrgs: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(orgs) != 1 {
		t.Fatalf("len(orgs) = %d, want 1", len(orgs))
	}
	o := orgs[0]
	if o.ID != "https://ror.org/00f54p054" {
		t.Errorf("ID = %q", o.ID)
	}
	if o.Name != "Massachusetts Institute of Technology" {
		t.Errorf("Name = %q", o.Name)
	}
	if o.ShortName != "MIT" {
		t.Errorf("ShortName = %q", o.ShortName)
	}
	if o.Country != "United States" {
		t.Errorf("Country = %q", o.Country)
	}
	if o.City != "Cambridge" {
		t.Errorf("City = %q", o.City)
	}
	if o.Established != 1861 {
		t.Errorf("Established = %d", o.Established)
	}
	if o.Status != "active" {
		t.Errorf("Status = %q", o.Status)
	}
	if o.Website != "https://www.mit.edu" {
		t.Errorf("Website = %q", o.Website)
	}
	if o.WikidataID != "Q49108" {
		t.Errorf("WikidataID = %q", o.WikidataID)
	}
	if o.GridID != "grid.116068.8" {
		t.Errorf("GridID = %q", o.GridID)
	}
}

func TestFilterOrgs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := r.URL.Query().Get("filter")
		if f != "types:Education" {
			t.Errorf("filter = %q, want types:Education", f)
		}
		pg := r.URL.Query().Get("page")
		if pg != "2" {
			t.Errorf("page = %q, want 2", pg)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockSearchResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	orgs, total, err := c.FilterOrgs(context.Background(), "types:Education", 2)
	if err != nil {
		t.Fatalf("FilterOrgs: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(orgs) != 1 {
		t.Fatalf("len(orgs) = %d, want 1", len(orgs))
	}
	if orgs[0].Name != "Massachusetts Institute of Technology" {
		t.Errorf("Name = %q", orgs[0].Name)
	}
}

func TestGetOrg(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/organizations/00f54p054" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockOrgResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)

	// test with short ID
	org, err := c.GetOrg(context.Background(), "00f54p054")
	if err != nil {
		t.Fatalf("GetOrg: %v", err)
	}
	if org.Name != "Massachusetts Institute of Technology" {
		t.Errorf("Name = %q", org.Name)
	}
}

func TestGetOrgStripsPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/organizations/00f54p054" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockOrgResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)

	// test with full URL
	org, err := c.GetOrg(context.Background(), "https://ror.org/00f54p054")
	if err != nil {
		t.Fatalf("GetOrg with full URL: %v", err)
	}
	if org.ID != "https://ror.org/00f54p054" {
		t.Errorf("ID = %q", org.ID)
	}
}

func TestRetryOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(mockOrgResponse))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	c.Retries = 5

	start := time.Now()
	org, err := c.GetOrg(context.Background(), "00f54p054")
	if err != nil {
		t.Fatalf("GetOrg after retries: %v", err)
	}
	if org.Name != "Massachusetts Institute of Technology" {
		t.Errorf("Name = %q after retries", org.Name)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestDisplayName(t *testing.T) {
	names := []wireName{
		{Value: "Massachusetts Institute of Technology", Types: []string{"ror_display", "label"}},
		{Value: "MIT", Types: []string{"acronym"}},
	}
	got := displayName(names)
	if got != "Massachusetts Institute of Technology" {
		t.Errorf("displayName = %q", got)
	}
}

func TestDisplayNameFallback(t *testing.T) {
	names := []wireName{
		{Value: "Some Org", Types: []string{"label"}},
	}
	got := displayName(names)
	if got != "Some Org" {
		t.Errorf("displayName fallback = %q", got)
	}
}

func TestStripRORPrefix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"00f54p054", "00f54p054"},
		{"https://ror.org/00f54p054", "00f54p054"},
		{"http://ror.org/00f54p054", "00f54p054"},
		{"/00f54p054/", "00f54p054"},
		{"  00f54p054  ", "00f54p054"},
	}
	for _, tc := range cases {
		got := stripRORPrefix(tc.in)
		if got != tc.want {
			t.Errorf("stripRORPrefix(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
