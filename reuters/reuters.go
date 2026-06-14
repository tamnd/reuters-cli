// Package reuters is the library behind the reuters command: the HTTP client,
// feed URL resolution, and RSS parsing for Reuters news.
//
// The client fetches RSS feeds from the public Reuters agency endpoint at
// https://www.reutersagency.com. No authentication is required. It sets a
// real User-Agent, paces requests, and retries transient 429/5xx errors with
// exponential backoff.
package reuters

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to Reuters.
const DefaultUserAgent = "reuters/dev (+https://github.com/tamnd/reuters-cli)"

// ErrUnknownSection is returned when an unknown section name is requested.
var ErrUnknownSection = errors.New("unknown section")

// sections lists supported feed sections and their URL topic parameters.
// The empty string for "all" means no best-topics filter is applied.
var sections = map[string]string{
	"world":     "world",
	"business":  "business-finance",
	"tech":      "tech",
	"sports":    "sports",
	"lifestyle": "lifestyle",
	"all":       "",
}

// Config holds constructor parameters.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://www.reutersagency.com",
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client fetches Reuters RSS feeds.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// feedURL builds the RSS URL for a section using cfg.BaseURL.
func (c *Client) feedURL(section string) (string, error) {
	topic, ok := sections[section]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownSection, section)
	}
	if topic == "" {
		return c.cfg.BaseURL + "/feed/?post_type=best", nil
	}
	return c.cfg.BaseURL + "/feed/?best-topics=" + topic + "&post_type=best", nil
}

// Feeds returns a map of section name to fully resolved feed URL.
func (c *Client) Feeds() map[string]string {
	out := make(map[string]string, len(sections))
	for name := range sections {
		u, _ := c.feedURL(name)
		out[name] = u
	}
	return out
}

// FetchFeed returns articles from the named section.
// section must be one of: world, business, tech, sports, lifestyle, all.
func (c *Client) FetchFeed(ctx context.Context, section string) ([]Item, error) {
	url, err := c.feedURL(section)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		items, retry, err := c.fetch(ctx, url)
		if err == nil {
			return items, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("fetch feed: %w", lastErr)
}

func (c *Client) fetch(ctx context.Context, url string) ([]Item, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := c.httpClient.Do(req)
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

	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}

	var root rssRoot
	if err := xml.Unmarshal(b, &root); err != nil {
		return nil, false, fmt.Errorf("parse RSS: %w", err)
	}

	items := make([]Item, 0, len(root.Channel.Items))
	for _, ri := range root.Channel.Items {
		items = append(items, Item{
			Title:       ri.Title,
			Link:        ri.Link,
			Description: ri.Description,
			PubDate:     ri.PubDate,
		})
	}
	return items, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
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
