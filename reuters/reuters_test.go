package reuters

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

const mockRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Reuters</title>
    <item>
      <title>Global markets rally on trade optimism</title>
      <link>https://www.reuters.com/markets/global-markets-rally-2024-03-15/</link>
      <description>&lt;p&gt;World equity markets rose on Friday&lt;/p&gt;</description>
      <pubDate>Fri, 15 Mar 2024 10:00:00 +0000</pubDate>
    </item>
    <item>
      <title>Tech giants face new EU regulation</title>
      <link>https://www.reuters.com/technology/tech-giants-eu-2024-03-14/</link>
      <description>European regulators &amp; tech companies clash over AI rules.</description>
      <pubDate>Thu, 14 Mar 2024 08:30:00 +0000</pubDate>
    </item>
  </channel>
</rss>`

func newTestClient(ts *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return NewClient(cfg)
}

func TestFetchFeedParsesItems(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(mockRSS))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	items, err := c.FetchFeed(context.Background(), "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	it := items[0]
	if it.Title != "Global markets rally on trade optimism" {
		t.Errorf("title = %q", it.Title)
	}
	if it.Link != "https://www.reuters.com/markets/global-markets-rally-2024-03-15/" {
		t.Errorf("link = %q", it.Link)
	}
	if it.PubDate != "Fri, 15 Mar 2024 10:00:00 +0000" {
		t.Errorf("pub_date = %q", it.PubDate)
	}
	// Description should be the raw RSS value (stripping is done at render time).
	if it.Description == "" {
		t.Error("description is empty")
	}

	it2 := items[1]
	if it2.Title != "Tech giants face new EU regulation" {
		t.Errorf("second item title = %q", it2.Title)
	}
}

func TestFetchFeedSendsUserAgent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(mockRSS))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.FetchFeed(context.Background(), "world")
	if err != nil {
		t.Fatal(err)
	}
}

func TestFetchFeedUnknownSection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockRSS))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.FetchFeed(context.Background(), "nosuchsection")
	if !errors.Is(err, ErrUnknownSection) {
		t.Fatalf("got %v, want ErrUnknownSection", err)
	}
}

func TestFeedsReturnsAll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockRSS))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	feeds := c.Feeds()
	want := []string{"world", "business", "tech", "sports", "lifestyle", "all"}
	for _, name := range want {
		if _, ok := feeds[name]; !ok {
			t.Errorf("Feeds() missing section %q", name)
		}
	}
	if len(feeds) != len(want) {
		t.Errorf("Feeds() returned %d sections, want %d", len(feeds), len(want))
	}
}

func TestStripHTML(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"<p>Hello <b>world</b></p>", "Hello world"},
		{"&amp; &lt; &gt; &quot; &#39; &nbsp;", "& < > \" '"},
		// TrimSpace removes leading/trailing whitespace; test inline &nbsp; works:
		{"a &nbsp; b", "a   b"},
		{"no tags here", "no tags here"},
	}
	for _, tc := range cases {
		got := StripHTML(tc.in)
		if got != tc.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFetchFeedRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(mockRSS))
	}))
	defer srv.Close()

	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := NewClient(cfg)

	_, err := c.FetchFeed(context.Background(), "all")
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}
