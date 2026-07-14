package podcasts

import (
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
)

func TestFeedItemToEpisodeStripsHTMLDescription(t *testing.T) {
	item := &gofeed.Item{
		GUID:  "guid-1",
		Title: "Episode 1",
		Description: `<p dir="ltr">After 25 years working in homicide, former Detective Chief ` +
			`Inspector Gary Jubelin is sitting down.&nbsp;</p> <p dir="ltr">&nbsp;</p>`,
	}

	episode := feedItemToEpisode("channel-1", item)

	if strings.ContainsAny(episode.Description, "<>") {
		t.Errorf("expected no HTML tags in description, got %q", episode.Description)
	}
	if strings.Contains(episode.Description, "&nbsp;") {
		t.Errorf("expected &nbsp; to be decoded, got %q", episode.Description)
	}
	want := "After 25 years working in homicide, former Detective Chief Inspector Gary Jubelin is sitting down."
	if got := strings.TrimSpace(episode.Description); got != want {
		t.Errorf("description = %q, want %q", got, want)
	}
}

func TestFeedChannelInfoStripsHTMLDescription(t *testing.T) {
	feed := &gofeed.Feed{
		Title:       "My Show",
		Description: `<p>Plain &amp; simple show notes.</p>`,
		Link:        "https://example.com",
	}

	_, description, _, _ := feedChannelInfo(feed)

	if strings.ContainsAny(description, "<>") {
		t.Errorf("expected no HTML tags in description, got %q", description)
	}
	want := "Plain & simple show notes."
	if description != want {
		t.Errorf("description = %q, want %q", description, want)
	}
}
