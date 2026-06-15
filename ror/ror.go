// Package ror is the library behind the ror command line:
// the HTTP client, request shaping, and the typed data models for the
// Research Organization Registry (ROR) at api.ror.org.
//
// The Client here is the spine every command shares. It sets a real
// User-Agent, paces requests so a busy session stays polite, and retries the
// transient failures (429 and 5xx) that any public API throws under load.
package ror

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to ROR.
const DefaultUserAgent = "ror-cli/dev (+https://github.com/tamnd/ror-cli)"

// Host is the API hostname.
const Host = "api.ror.org"

// BaseURL is the root every request is built from.
const BaseURL = "https://" + Host

// Config holds the tunable knobs for a Client.
type Config struct {
	BaseURL string
	Rate    time.Duration
	Timeout time.Duration
	Retries int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL: "https://api.ror.org",
		Rate:    300 * time.Millisecond,
		Timeout: 15 * time.Second,
		Retries: 3,
	}
}

// Client talks to the ROR API over HTTP.
type Client struct {
	HTTP      *http.Client
	UserAgent string
	BaseURL   string
	// Rate is the minimum gap between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int

	mu   sync.Mutex
	last time.Time
}

// NewClient returns a Client built from DefaultConfig.
func NewClient() *Client {
	cfg := DefaultConfig()
	return &Client{
		HTTP:      &http.Client{Timeout: cfg.Timeout},
		UserAgent: DefaultUserAgent,
		BaseURL:   cfg.BaseURL,
		Rate:      cfg.Rate,
		Retries:   cfg.Retries,
	}
}

// Get fetches a URL and returns the response body. It paces and retries
// according to the client's settings.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) (body []byte, retry bool, err error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

// pace blocks until at least Rate has passed since the previous request.
func (c *Client) pace() {
	if c.Rate <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// --- wire types (unexported) ---

type wireResponse struct {
	NumberOfResults int       `json:"number_of_results"`
	Items           []wireOrg `json:"items"`
}

type wireOrg struct {
	ID          string          `json:"id"`
	Names       []wireName      `json:"names"`
	Types       []string        `json:"types"`
	Locations   []wireLocation  `json:"locations"`
	Established int             `json:"established"`
	Status      string          `json:"status"`
	Links       []wireLink      `json:"links"`
	ExternalIDs []wireExtID     `json:"external_ids"`
}

type wireName struct {
	Value string   `json:"value"`
	Types []string `json:"types"`
}

type wireLocation struct {
	GeonamesDetails struct {
		CountryName string `json:"country_name"`
		Name        string `json:"name"`
	} `json:"geonames_details"`
}

type wireLink struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

type wireExtID struct {
	Type string   `json:"type"`
	All  []string `json:"all"`
}

// --- public types ---

// Org is a Research Organization Registry record.
type Org struct {
	ID          string   `json:"id"          kit:"id"`
	Name        string   `json:"name"`
	ShortName   string   `json:"short_name,omitempty"`
	Country     string   `json:"country,omitempty"`
	City        string   `json:"city,omitempty"`
	Status      string   `json:"status,omitempty"`
	Types       []string `json:"types,omitempty"`
	Established int      `json:"established,omitempty"`
	Website     string   `json:"website,omitempty"`
	WikidataID  string   `json:"wikidata_id,omitempty"`
	GridID      string   `json:"grid_id,omitempty"`
}

// displayName returns the ror_display name from the names slice, falling
// back to the first name if none is tagged ror_display.
func displayName(names []wireName) string {
	for _, n := range names {
		for _, t := range n.Types {
			if t == "ror_display" {
				return n.Value
			}
		}
	}
	if len(names) > 0 {
		return names[0].Value
	}
	return ""
}

// shortName returns the first name tagged "acronym".
func shortName(names []wireName) string {
	for _, n := range names {
		for _, t := range n.Types {
			if t == "acronym" {
				return n.Value
			}
		}
	}
	return ""
}

func toOrg(w wireOrg) Org {
	o := Org{
		ID:          w.ID,
		Name:        displayName(w.Names),
		ShortName:   shortName(w.Names),
		Types:       w.Types,
		Established: w.Established,
		Status:      w.Status,
	}
	if len(w.Locations) > 0 {
		o.Country = w.Locations[0].GeonamesDetails.CountryName
		o.City = w.Locations[0].GeonamesDetails.Name
	}
	for _, l := range w.Links {
		if l.Type == "website" {
			o.Website = l.Value
			break
		}
	}
	for _, e := range w.ExternalIDs {
		switch e.Type {
		case "wikidata":
			if len(e.All) > 0 {
				o.WikidataID = e.All[0]
			}
		case "grid":
			if len(e.All) > 0 {
				o.GridID = e.All[0]
			}
		}
	}
	return o
}

// stripRORPrefix removes the "https://ror.org/" prefix if present, returning
// just the short ID (e.g. "00f54p054").
func stripRORPrefix(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "https://ror.org/")
	id = strings.TrimPrefix(id, "http://ror.org/")
	id = strings.Trim(id, "/")
	return id
}

// SearchOrgs searches ROR for organizations matching query.
// page is 1-based. Returns the matching orgs, total result count, and any error.
func (c *Client) SearchOrgs(ctx context.Context, query string, page int) ([]Org, int, error) {
	if page < 1 {
		page = 1
	}
	u := c.BaseURL + "/v2/organizations?query=" + url.QueryEscape(query) +
		"&page=" + fmt.Sprintf("%d", page)
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, 0, err
	}
	var wr wireResponse
	if err := json.Unmarshal(body, &wr); err != nil {
		return nil, 0, fmt.Errorf("parse response: %w", err)
	}
	orgs := make([]Org, len(wr.Items))
	for i, item := range wr.Items {
		orgs[i] = toOrg(item)
	}
	return orgs, wr.NumberOfResults, nil
}

// FilterOrgs returns organizations matching a ROR filter expression such as
// "types:Education" or "country.country_code:US". page is 1-based.
func (c *Client) FilterOrgs(ctx context.Context, filter string, page int) ([]Org, int, error) {
	if page < 1 {
		page = 1
	}
	u := c.BaseURL + "/v2/organizations?filter=" + url.QueryEscape(filter) +
		"&page=" + fmt.Sprintf("%d", page)
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, 0, err
	}
	var wr wireResponse
	if err := json.Unmarshal(body, &wr); err != nil {
		return nil, 0, fmt.Errorf("parse response: %w", err)
	}
	orgs := make([]Org, len(wr.Items))
	for i, item := range wr.Items {
		orgs[i] = toOrg(item)
	}
	return orgs, wr.NumberOfResults, nil
}

// GetOrg fetches a single organization by its ROR ID.
// id may be a full URL ("https://ror.org/00f54p054") or just the short ID
// ("00f54p054").
func (c *Client) GetOrg(ctx context.Context, id string) (*Org, error) {
	id = stripRORPrefix(id)
	if id == "" {
		return nil, fmt.Errorf("empty ROR id")
	}
	u := c.BaseURL + "/v2/organizations/" + id
	body, err := c.Get(ctx, u)
	if err != nil {
		return nil, err
	}
	var wo wireOrg
	if err := json.Unmarshal(body, &wo); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	org := toOrg(wo)
	return &org, nil
}
