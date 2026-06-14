package cli

import (
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tamnd/reuters-cli/reuters"
)

// newsCmd fetches articles from a Reuters RSS feed section.
func (a *App) newsCmd() *cobra.Command {
	var section string
	cmd := &cobra.Command{
		Use:   "news",
		Short: "Fetch Reuters news articles",
		RunE: func(cmd *cobra.Command, _ []string) error {
			section = strings.ToLower(strings.TrimSpace(section))
			n := a.effectiveLimit(20)
			a.progressf("fetching %s news...", section)
			items, err := a.client.FetchFeed(cmd.Context(), section)
			if err != nil {
				if isUnknownSection(err) {
					return codeError(exitUsage, err)
				}
				return codeError(exitError, err)
			}
			if n > len(items) {
				n = len(items)
			}
			out := make([]reuters.NewsItem, 0, n)
			for i, it := range items[:n] {
				out = append(out, reuters.NewsItem{
					Rank:    i + 1,
					Title:   it.Title,
					Summary: reuters.StripHTML(it.Description),
					Date:    parseDate(it.PubDate),
					URL:     it.Link,
				})
			}
			return a.renderOrEmpty(out, len(out))
		},
	}
	cmd.Flags().StringVar(&section, "section", "all", "section: world, business, tech, sports, lifestyle, all")
	return cmd
}

// parseDate formats an RFC 1123 / RFC 1123Z pubDate as "2006-01-02".
func parseDate(s string) string {
	for _, layout := range []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC().Format("2006-01-02")
		}
	}
	return s
}
