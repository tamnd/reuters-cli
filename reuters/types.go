package reuters

import (
	"regexp"
	"strings"
)

// rssRoot is the top-level structure of an RSS 2.0 feed.
type rssRoot struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

// rssItem is a single entry in an RSS channel.
type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

// Item is the public record emitted for each news article.
type Item struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Description string `json:"description"`
	PubDate     string `json:"pub_date"`
}

// NewsItem is the rendered record for the news command.
type NewsItem struct {
	Rank    int    `json:"rank"             table:"rank"`
	Title   string `json:"title"   kit:"id" table:"title"`
	Summary string `json:"summary"          table:"summary"`
	Date    string `json:"date"             table:"date"`
	URL     string `json:"url"              table:"url,url"`
	Section string `json:"section"          table:"section"`
}

// FeedItem is the rendered record for the feeds command.
type FeedItem struct {
	Name string `json:"name" kit:"id" table:"name"`
	URL  string `json:"url"           table:"url,url"`
}

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

// StripHTML removes HTML tags and decodes common entities.
func StripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.TrimSpace(s)
	return s
}
