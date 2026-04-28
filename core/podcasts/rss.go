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

const podcastNS = "https://podcastindex.org/namespace/1.0"

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

	// Podcasting 2.0 channel fields
	PodcastGUID     string
	Locked          bool
	LockedOwner     string
	Medium          string
	FundingURL      string
	FundingText     string
	UpdateFrequency string
	UpdateRRule     string
	Complete        bool
	Persons         []model.PodcastPerson

	// Podcasting 2.0 Tier 3 channel fields
	UsesPodping bool
	Podroll     []model.PodcastPodrollItem
	LiveItems   []model.PodcastLiveItem
}

type FeedPreview struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	ImageURL      string `json:"imageUrl"`
	EpisodeCount  int    `json:"episodeCount"`
	AlreadyExists bool   `json:"alreadyExists"`

	// Podcasting 2.0
	Medium          string `json:"medium,omitempty"`
	UpdateFrequency string `json:"updateFrequency,omitempty"`
	FundingURL      string `json:"fundingUrl,omitempty"`
	FundingText     string `json:"fundingText,omitempty"`
}

func ParseFeedPreview(rssURL string) (*FeedPreview, error) {
	feed, err := fetchAndParse(rssURL)
	if err != nil {
		return nil, err
	}
	return &FeedPreview{
		Title:           feed.Title,
		Description:     feed.Description,
		ImageURL:        feed.ImageURL,
		EpisodeCount:    len(feed.Episodes),
		Medium:          feed.Medium,
		UpdateFrequency: feed.UpdateFrequency,
		FundingURL:      feed.FundingURL,
		FundingText:     feed.FundingText,
	}, nil
}

// ---- XML struct definitions ----

type rssRoot struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string      `xml:"title"`
	Description string      `xml:"description"`
	Image       rssImage    `xml:"image"`
	ItunesImage itunesImage `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Items       []rssItem   `xml:"item"`

	// Podcasting 2.0 channel tags — Tier 1 & 2
	PodcastGUID       string            `xml:"https://podcastindex.org/namespace/1.0 guid"`
	PodcastLocked     podcastLocked     `xml:"https://podcastindex.org/namespace/1.0 locked"`
	PodcastMedium     string            `xml:"https://podcastindex.org/namespace/1.0 medium"`
	PodcastFundings   []podcastFunding  `xml:"https://podcastindex.org/namespace/1.0 funding"`
	PodcastPersons    []podcastPerson   `xml:"https://podcastindex.org/namespace/1.0 person"`
	PodcastUpdateFreq podcastUpdateFreq `xml:"https://podcastindex.org/namespace/1.0 updateFrequency"`

	// Podcasting 2.0 channel tags — Tier 3
	PodcastPodping   podcastPodping       `xml:"https://podcastindex.org/namespace/1.0 podping"`
	PodcastPodroll   podcastPodroll       `xml:"https://podcastindex.org/namespace/1.0 podroll"`
	PodcastLiveItems []podcastLiveItemXML `xml:"https://podcastindex.org/namespace/1.0 liveItem"`
}

// Tier 3 XML parsing structs.

type podcastPodping struct {
	UsesPodping string `xml:"usesPodping,attr"`
}

type podcastRemoteItem struct {
	FeedGUID string `xml:"feedGuid,attr"`
	FeedURL  string `xml:"feedUrl,attr"`
	Title    string `xml:"title,attr"`
}

type podcastPodroll struct {
	Items []podcastRemoteItem `xml:"https://podcastindex.org/namespace/1.0 remoteItem"`
}

type podcastContentLink struct {
	Href string `xml:"href,attr"`
	Text string `xml:",chardata"`
}

type podcastLiveItemXML struct {
	Status      string             `xml:"status,attr"`
	Start       string             `xml:"start,attr"`
	End         string             `xml:"end,attr"`
	Title       string             `xml:"title"`
	GUID        string             `xml:"guid"`
	Enclosure   enclosure          `xml:"enclosure"`
	ContentLink podcastContentLink `xml:"https://podcastindex.org/namespace/1.0 contentLink"`
}

type rssImage struct {
	URL string `xml:"url"`
}

type itunesImage struct {
	Href string `xml:"href,attr"`
}

type podcastLocked struct {
	Owner string `xml:"owner,attr"`
	Value string `xml:",chardata"`
}

type podcastFunding struct {
	URL  string `xml:"url,attr"`
	Text string `xml:",chardata"`
}

type podcastPerson struct {
	Role  string `xml:"role,attr"`
	Group string `xml:"group,attr"`
	Img   string `xml:"img,attr"`
	Href  string `xml:"href,attr"`
	Name  string `xml:",chardata"`
}

type podcastUpdateFreq struct {
	Complete string `xml:"complete,attr"`
	RRule    string `xml:"rrule,attr"`
	Text     string `xml:",chardata"`
}

type rssItem struct {
	Title         string    `xml:"title"`
	Description   string    `xml:"description"`
	ItunesSummary string    `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
	GUID          string    `xml:"guid"`
	PubDate       string    `xml:"pubDate"`
	Enclosure     enclosure `xml:"enclosure"`
	ItunesDur     string    `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`

	// Podcasting 2.0 episode tags
	PodcastChapters    podcastChapters    `xml:"https://podcastindex.org/namespace/1.0 chapters"`
	PodcastTranscripts []podcastTranscript `xml:"https://podcastindex.org/namespace/1.0 transcript"`
	PodcastSeason      podcastSeason      `xml:"https://podcastindex.org/namespace/1.0 season"`
	PodcastEpisodeNum  podcastEpisodeNum  `xml:"https://podcastindex.org/namespace/1.0 episode"`
	PodcastSoundbite   podcastSoundbite   `xml:"https://podcastindex.org/namespace/1.0 soundbite"`
	PodcastPersons     []podcastPerson    `xml:"https://podcastindex.org/namespace/1.0 person"`
}

type enclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

type podcastChapters struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

type podcastTranscript struct {
	URL      string `xml:"url,attr"`
	Type     string `xml:"type,attr"`
	Language string `xml:"language,attr"`
	Rel      string `xml:"rel,attr"`
}

type podcastSeason struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type podcastEpisodeNum struct {
	Display string `xml:"display,attr"`
	Value   string `xml:",chardata"`
}

type podcastSoundbite struct {
	StartTime string `xml:"startTime,attr"`
	Duration  string `xml:"duration,attr"`
	Title     string `xml:",chardata"`
}

// ---- Parsing ----

func ParseRSSFeed(data []byte) (*rssFeed, error) {
	var root rssRoot
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing RSS feed: %w", err)
	}

	ch := root.Channel
	feed := &rssFeed{
		Title:       ch.Title,
		Description: ch.Description,

		// Podcasting 2.0 channel
		PodcastGUID:     ch.PodcastGUID,
		Locked:          strings.TrimSpace(ch.PodcastLocked.Value) == "yes",
		LockedOwner:     ch.PodcastLocked.Owner,
		Medium:          ch.PodcastMedium,
		UpdateFrequency: strings.TrimSpace(ch.PodcastUpdateFreq.Text),
		UpdateRRule:     ch.PodcastUpdateFreq.RRule,
		Complete:        strings.TrimSpace(ch.PodcastUpdateFreq.Complete) == "true",
	}

	// itunes:image takes precedence over <image><url>.
	if href := extractItunesImageHref(data); href != "" {
		feed.ImageURL = href
	} else if ch.ItunesImage.Href != "" {
		feed.ImageURL = ch.ItunesImage.Href
	} else {
		feed.ImageURL = ch.Image.URL
	}

	// first funding entry
	if len(ch.PodcastFundings) > 0 {
		feed.FundingURL = ch.PodcastFundings[0].URL
		feed.FundingText = strings.TrimSpace(ch.PodcastFundings[0].Text)
	}

	// channel persons
	for _, p := range ch.PodcastPersons {
		feed.Persons = append(feed.Persons, model.PodcastPerson{
			Name:  strings.TrimSpace(p.Name),
			Role:  defaultStr(p.Role, "host"),
			Group: defaultStr(p.Group, "cast"),
			Img:   p.Img,
			Href:  p.Href,
		})
	}

	// podcast:podping
	feed.UsesPodping = strings.TrimSpace(ch.PodcastPodping.UsesPodping) == "true"

	// podcast:podroll
	for i, item := range ch.PodcastPodroll.Items {
		feed.Podroll = append(feed.Podroll, model.PodcastPodrollItem{
			FeedGUID:  item.FeedGUID,
			FeedURL:   item.FeedURL,
			Title:     item.Title,
			SortOrder: i,
		})
	}

	// podcast:liveItem
	for _, li := range ch.PodcastLiveItems {
		startTime, _ := time.Parse(time.RFC3339, li.Start)
		endTime, _ := time.Parse(time.RFC3339, li.End)
		feed.LiveItems = append(feed.LiveItems, model.PodcastLiveItem{
			GUID:            li.GUID,
			Title:           li.Title,
			Status:          li.Status,
			StartTime:       startTime,
			EndTime:         endTime,
			EnclosureURL:    li.Enclosure.URL,
			EnclosureType:   li.Enclosure.Type,
			ContentLinkURL:  li.ContentLink.Href,
			ContentLinkText: strings.TrimSpace(li.ContentLink.Text),
		})
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

			// Podcasting 2.0 episode
			ChaptersURL:    item.PodcastChapters.URL,
			ChaptersType:   item.PodcastChapters.Type,
			Season:         parseInt(item.PodcastSeason.Value),
			SeasonName:     item.PodcastSeason.Name,
			EpisodeNumber:  strings.TrimSpace(item.PodcastEpisodeNum.Value),
			EpisodeDisplay: item.PodcastEpisodeNum.Display,
			SoundbiteStart: parseFloat(item.PodcastSoundbite.StartTime),
			SoundbiteDur:   parseFloat(item.PodcastSoundbite.Duration),
			SoundbiteTitle: strings.TrimSpace(item.PodcastSoundbite.Title),
		}

		for _, t := range item.PodcastTranscripts {
			ep.Transcripts = append(ep.Transcripts, model.PodcastTranscript{
				URL:      t.URL,
				MimeType: t.Type,
				Language: t.Language,
				Rel:      t.Rel,
			})
		}

		for _, p := range item.PodcastPersons {
			ep.Persons = append(ep.Persons, model.PodcastPerson{
				Name:  strings.TrimSpace(p.Name),
				Role:  defaultStr(p.Role, "host"),
				Group: defaultStr(p.Group, "cast"),
				Img:   p.Img,
				Href:  p.Href,
			})
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

func parseInt(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func defaultStr(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	return s
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
	if u, err := url.Parse(enclosureURL); err == nil {
		if ext := path.Ext(u.Path); ext != "" {
			return strings.TrimPrefix(ext, ".")
		}
	}
	return ""
}
