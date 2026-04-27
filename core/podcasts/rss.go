package podcasts

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"
)

// Go's encoding/xml has a known limitation with inherited namespace prefixes.
// We fall back to a regex scan for itunes:image when struct tag parsing yields nothing.
var itunesImageRe = regexp.MustCompile(`<[^:>]*:image[^>]+href="([^"]*)"`)

func extractItunesImageHref(data []byte) string {
	if m := itunesImageRe.FindSubmatch(data); len(m) > 1 {
		return string(m[1])
	}
	return ""
}

type rssFeed struct {
	Title       string
	Description string
	ImageURL    string
	Episodes    []model.PodcastEpisode
}

type FeedPreview struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	ImageURL      string `json:"imageUrl"`
	EpisodeCount  int    `json:"episodeCount"`
	AlreadyExists bool   `json:"alreadyExists"`
}

func ParseFeedPreview(rssURL string) (*FeedPreview, error) {
	feed, err := fetchAndParse(rssURL)
	if err != nil {
		return nil, err
	}
	return &FeedPreview{
		Title:        feed.Title,
		Description:  feed.Description,
		ImageURL:     feed.ImageURL,
		EpisodeCount: len(feed.Episodes),
	}, nil
}

type rssRoot struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string      `xml:"title"`
	Description string      `xml:"description"`
	Image       rssImage    `xml:"image"`
	ItunesImage itunesImage `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Items       []rssItem   `xml:"item"`
}

type rssImage struct {
	URL string `xml:"url"`
}

type itunesImage struct {
	Href string `xml:"href,attr"`
}

type rssItem struct {
	Title         string    `xml:"title"`
	Description   string    `xml:"description"`
	ItunesSummary string    `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
	GUID          string    `xml:"guid"`
	PubDate       string    `xml:"pubDate"`
	Enclosure     enclosure `xml:"enclosure"`
	ItunesDur     string    `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`
}

type enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

func ParseRSSFeed(data []byte) (*rssFeed, error) {
	var root rssRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing RSS feed: %w", err)
	}

	ch := root.Channel
	feed := &rssFeed{
		Title:       ch.Title,
		Description: ch.Description,
	}

	// itunes:image takes precedence over <image><url>.
	// Use regex fallback because Go's xml package doesn't always resolve
	// namespace prefixes inherited from ancestor elements.
	if href := extractItunesImageHref(data); href != "" {
		feed.ImageURL = href
	} else if ch.ItunesImage.Href != "" {
		feed.ImageURL = ch.ItunesImage.Href
	} else {
		feed.ImageURL = ch.Image.URL
	}

	for _, item := range ch.Items {
		desc := item.Description
		if desc == "" {
			desc = item.ItunesSummary
		}

		pubDate, _ := parseRSSDate(item.PubDate)
		suffix := suffixFromMIME(item.Enclosure.Type, item.Enclosure.URL)

		ep := model.PodcastEpisode{
			GUID:         item.GUID,
			Title:        item.Title,
			Description:  desc,
			PublishDate:  pubDate,
			EnclosureURL: item.Enclosure.URL,
			Size:         item.Enclosure.Length,
			ContentType:  item.Enclosure.Type,
			Suffix:       suffix,
			Duration:     parseDuration(item.ItunesDur),
			Status:       model.PodcastStatusNew,
		}
		feed.Episodes = append(feed.Episodes, ep)
	}

	return feed, nil
}

func parseRSSDate(s string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %q", s)
}

func parseDuration(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	switch len(parts) {
	case 3:
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		sec, _ := strconv.Atoi(parts[2])
		return h*3600 + m*60 + sec
	case 2:
		m, _ := strconv.Atoi(parts[0])
		sec, _ := strconv.Atoi(parts[1])
		return m*60 + sec
	default:
		sec, _ := strconv.Atoi(s)
		return sec
	}
}

var mimeToSuffix = map[string]string{
	"audio/mpeg":  "mp3",
	"audio/mp3":   "mp3",
	"audio/mp4":   "m4a",
	"audio/m4a":   "m4a",
	"audio/ogg":   "ogg",
	"audio/opus":  "opus",
	"audio/flac":  "flac",
	"audio/x-m4a": "m4a",
}

func suffixFromMIME(mimeType, enclosureURL string) string {
	base := strings.Split(mimeType, ";")[0]
	base = strings.TrimSpace(strings.ToLower(base))
	if s, ok := mimeToSuffix[base]; ok {
		return s
	}
	// fallback: extract from URL path
	if u, err := url.Parse(enclosureURL); err == nil {
		if ext := path.Ext(u.Path); ext != "" {
			return strings.TrimPrefix(ext, ".")
		}
	}
	return ""
}
