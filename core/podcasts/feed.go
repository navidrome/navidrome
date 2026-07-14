package podcasts

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/deluan/sanitize"
	"github.com/mmcdole/gofeed"
	"github.com/navidrome/navidrome/model"
)

func fetchFeed(ctx context.Context, url string) (*gofeed.Feed, error) {
	parser := gofeed.NewParser()
	return parser.ParseURLWithContext(url, ctx)
}

func feedItemToEpisode(channelID string, item *gofeed.Item) model.PodcastEpisode {
	episode := model.PodcastEpisode{
		ChannelID:      channelID,
		Guid:           item.GUID,
		Title:          item.Title,
		Description:    strings.TrimSpace(sanitize.HTML(item.Description)),
		DownloadStatus: model.PodcastEpisodeNotDownloaded,
	}

	if len(item.Enclosures) > 0 {
		enc := item.Enclosures[0]
		episode.EnclosureUrl = enc.URL
		episode.ContentType = enc.Type
	}
	if episode.Guid == "" {
		episode.Guid = episode.EnclosureUrl
	}

	if item.PublishedParsed != nil {
		t := *item.PublishedParsed
		episode.PublishDate = &t
	}

	if item.ITunesExt != nil {
		episode.Duration = parseItunesDuration(item.ITunesExt.Duration)
	}

	return episode
}

// parseItunesDuration handles both "HH:MM:SS"/"MM:SS" and raw-seconds forms
// that real-world podcast feeds use for <itunes:duration>.
func parseItunesDuration(raw string) float32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if !strings.Contains(raw, ":") {
		secs, err := strconv.ParseFloat(raw, 32)
		if err != nil {
			return 0
		}
		return float32(secs)
	}
	parts := strings.Split(raw, ":")
	var total time.Duration
	multiplier := []time.Duration{time.Second, time.Minute, time.Hour}
	for i := 0; i < len(parts) && i < len(multiplier); i++ {
		idx := len(parts) - 1 - i
		v, err := strconv.Atoi(strings.TrimSpace(parts[idx]))
		if err != nil {
			return 0
		}
		total += time.Duration(v) * multiplier[i]
	}
	return float32(total.Seconds())
}

func feedChannelInfo(feed *gofeed.Feed) (title, description, homePage, imageUrl string) {
	title = feed.Title
	description = strings.TrimSpace(sanitize.HTML(feed.Description))
	homePage = feed.Link
	if feed.Image != nil {
		imageUrl = feed.Image.URL
	}
	if feed.ITunesExt != nil && feed.ITunesExt.Image != "" {
		imageUrl = feed.ITunesExt.Image
	}
	return
}
