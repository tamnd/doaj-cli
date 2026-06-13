// Package doaj is the library behind the doaj command line:
// the HTTP client, request shaping, wire decoding, and typed data models
// for the DOAJ (Directory of Open Access Journals) REST API v3.
//
// The API is open and requires no authentication key for read-only access.
// Base URL: https://doaj.org/api/v3
package doaj

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultBaseURL = "https://doaj.org/api/v3"

// DefaultUserAgent identifies the client to DOAJ.
const DefaultUserAgent = "doaj-cli/dev (+https://github.com/tamnd/doaj-cli)"

// ErrNotFound is returned when the API returns a 404 or a null body.
var ErrNotFound = errors.New("not found")

// Config holds constructor parameters for the DOAJ client.
type Config struct {
	BaseURL   string // default: "https://doaj.org/api/v3"
	UserAgent string
	Rate      time.Duration // default: 200ms
	Retries   int           // default: 3
	Timeout   time.Duration // default: 30s
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   defaultBaseURL,
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the DOAJ REST API v3.
type Client struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client built from cfg.
func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = defaultBaseURL
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		baseURL:    base,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// get fetches a URL with pacing and retries.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
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

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
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

// getJSON fetches and JSON-decodes into v.
// Returns ErrNotFound when the response is 404 or the body trims to "null".
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "null" {
		return ErrNotFound
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// normaliseISSN inserts a hyphen at position 4 when the user omits it.
func normaliseISSN(issn string) string {
	issn = strings.TrimSpace(issn)
	if len(issn) == 8 && !strings.Contains(issn, "-") {
		return issn[:4] + "-" + issn[4:]
	}
	return issn
}

// ─── public methods ───────────────────────────────────────────────────────────

// SearchArticles searches the /search/articles endpoint. It pages internally
// until limit results are accumulated or the API has no more pages.
func (c *Client) SearchArticles(ctx context.Context, query string, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 10
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}

	var out []Article
	page := 1
	for {
		params := url.Values{}
		params.Set("q", query)
		params.Set("pageSize", strconv.Itoa(pageSize))
		params.Set("page", strconv.Itoa(page))
		rawURL := c.baseURL + "/search/articles?" + params.Encode()

		var resp articleSearchResp
		if err := c.getJSON(ctx, rawURL, &resp); err != nil {
			return out, err
		}
		for _, r := range resp.Results {
			out = append(out, wireArticleToArticle(r, len(out)+1))
			if len(out) >= limit {
				return out, nil
			}
		}
		if len(resp.Results) == 0 || len(out) >= resp.Total {
			break
		}
		page++
	}
	return out, nil
}

// SearchJournals searches the /search/journals endpoint. It pages internally
// until limit results are accumulated or the API has no more pages.
func (c *Client) SearchJournals(ctx context.Context, query string, limit int) ([]Journal, error) {
	if limit <= 0 {
		limit = 10
	}
	pageSize := limit
	if pageSize > 100 {
		pageSize = 100
	}

	var out []Journal
	page := 1
	for {
		params := url.Values{}
		params.Set("q", query)
		params.Set("pageSize", strconv.Itoa(pageSize))
		params.Set("page", strconv.Itoa(page))
		rawURL := c.baseURL + "/search/journals?" + params.Encode()

		var resp journalSearchResp
		if err := c.getJSON(ctx, rawURL, &resp); err != nil {
			return out, err
		}
		for _, r := range resp.Results {
			out = append(out, wireJournalToJournal(r, len(out)+1))
			if len(out) >= limit {
				return out, nil
			}
		}
		if len(resp.Results) == 0 || len(out) >= resp.Total {
			break
		}
		page++
	}
	return out, nil
}

// GetJournal fetches a single journal by ISSN.
func (c *Client) GetJournal(ctx context.Context, issn string) (Journal, error) {
	issn = normaliseISSN(issn)
	rawURL := c.baseURL + "/journals/" + url.PathEscape(issn)

	var result wireJournalResult
	if err := c.getJSON(ctx, rawURL, &result); err != nil {
		return Journal{}, err
	}
	return wireJournalToJournal(result, 0), nil
}

// GetArticle fetches a single article by its DOAJ internal id.
func (c *Client) GetArticle(ctx context.Context, id string) (Article, error) {
	rawURL := c.baseURL + "/articles/" + url.PathEscape(id)

	var result wireArticleResult
	if err := c.getJSON(ctx, rawURL, &result); err != nil {
		return Article{}, err
	}
	return wireArticleToArticle(result, 0), nil
}
