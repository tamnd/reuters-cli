package reuters

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

// Domain is the reuters kit driver. It carries no state; the per-run Client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "reuters",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "reuters",
			Short:  "A command line for Reuters news.",
			Long: `A command line for Reuters news.

Fetch the latest Reuters headlines via RSS. No API key required.`,
			Site: "https://" + Host,
			Repo: "https://github.com/tamnd/reuters-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "feeds", Group: "news", List: true,
		URIType: "feed", Summary: "List available Reuters RSS feeds"}, feedsCmd)

	kit.Handle(app, kit.OpMeta{Name: "news", Group: "news", List: true,
		URIType: "news_item", Summary: "Latest Reuters news headlines"}, newsCmd)
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient(DefaultConfig())
	if cfg.UserAgent != "" {
		c.cfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.cfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.cfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.httpClient.Timeout = cfg.Timeout
		c.cfg.Timeout = cfg.Timeout
	}
	return c, nil
}

// feedsCmd emits the static feed list without any network call.
func feedsCmd(_ context.Context, _ struct{}, emit func(*FeedItem) error) error {
	c := NewClient(DefaultConfig())
	feedMap := c.Feeds()
	names := make([]string, 0, len(feedMap))
	for name := range feedMap {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		f := &FeedItem{Name: name, URL: feedMap[name]}
		if err := emit(f); err != nil {
			return err
		}
	}
	return nil
}

type newsIn struct {
	Section string  `kit:"flag" help:"section: world, business, tech, sports, lifestyle, all"`
	Limit   int     `kit:"flag,inherit" help:"max results"`
	Client  *Client `kit:"inject"`
}

func newsCmd(ctx context.Context, in newsIn, emit func(*NewsItem) error) error {
	section := in.Section
	if section == "" {
		section = "all"
	}
	items, err := in.Client.FetchFeed(ctx, section)
	if err != nil {
		return err
	}
	if in.Limit > 0 && in.Limit < len(items) {
		items = items[:in.Limit]
	}
	for i, it := range items {
		ni := &NewsItem{
			Rank:    i + 1,
			Title:   it.Title,
			Summary: StripHTML(it.Description),
			Date:    parseNewsDate(it.PubDate),
			URL:     it.Link,
			Section: section,
		}
		if err := emit(ni); err != nil {
			return err
		}
	}
	return nil
}

// parseNewsDate parses RFC1123 / RFC1123Z and returns "2006-01-02 15:04".
func parseNewsDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	t, err := time.Parse(time.RFC1123, s)
	if err != nil {
		t, err = time.Parse(time.RFC1123Z, s)
		if err != nil {
			return s
		}
	}
	return t.UTC().Format("2006-01-02 15:04")
}
