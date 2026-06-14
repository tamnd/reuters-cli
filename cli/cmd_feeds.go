package cli

import (
	"sort"

	"github.com/spf13/cobra"
	"github.com/tamnd/reuters-cli/reuters"
)

// feedsCmd lists all available Reuters RSS feeds.
func (a *App) feedsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "feeds",
		Short: "List available Reuters RSS feeds",
		RunE: func(cmd *cobra.Command, _ []string) error {
			feedMap := a.client.Feeds()
			out := make([]reuters.FeedItem, 0, len(feedMap))
			for name, url := range feedMap {
				out = append(out, reuters.FeedItem{Name: name, URL: url})
			}
			sort.Slice(out, func(i, j int) bool {
				return out[i].Name < out[j].Name
			})
			return a.renderOrEmpty(out, len(out))
		},
	}
}
