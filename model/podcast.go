package model

import (
	"context"
	"time"

	"github.com/deluan/rest"
)

// PodcastPodrollItem represents one entry in a podcast:podroll recommendation list.
type PodcastPodrollItem struct {
	ID        string    `structs:"id"         json:"id"`
	ChannelID string    `structs:"channel_id" json:"channelId"`
	FeedGUID  string    `structs:"feed_guid"  json:"feedGuid,omitempty"`
	FeedURL   string    `structs:"feed_url"   json:"feedUrl,omitempty"`
	Title     string    `structs:"title"      json:"title,omitempty"`
	SortOrder int       `structs:"sort_order" json:"sortOrder"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
}

// PodcastLiveItem represents a podcast:liveItem stream associated with a channel.
type PodcastLiveItem struct {
	ID              string    `structs:"id"                json:"id"`
	ChannelID       string    `structs:"channel_id"        json:"channelId"`
	GUID            string    `structs:"guid"              json:"guid,omitempty"`
	Title           string    `structs:"title"             json:"title,omitempty"`
	Status          string    `structs:"status"            json:"status"`
	StartTime       time.Time `structs:"start_time"        json:"startTime,omitempty"`
	EndTime         time.Time `structs:"end_time"          json:"endTime,omitempty"`
	EnclosureURL    string    `structs:"enclosure_url"     json:"enclosureUrl,omitempty"`
	EnclosureType   string    `structs:"enclosure_type"    json:"enclosureType,omitempty"`
	ContentLinkURL  string    `structs:"content_link_url"  json:"contentLinkUrl,omitempty"`
	ContentLinkText string    `structs:"content_link_text" json:"contentLinkText,omitempty"`
	CreatedAt       time.Time `structs:"created_at"        json:"createdAt"`
	UpdatedAt       time.Time `structs:"updated_at"        json:"updatedAt"`
}

// PodcastPodrollItems is a slice of PodcastPodrollItem.
type PodcastPodrollItems []PodcastPodrollItem

// PodcastPodrollRepository manages podcast:podroll entries for channels.
type PodcastPodrollRepository interface {
	GetByChannel(ctx context.Context, channelID string) (PodcastPodrollItems, error)
	GetByChannels(ctx context.Context, channelIDs []string) (PodcastPodrollItems, error)
	SaveForChannel(ctx context.Context, channelID string, items []PodcastPodrollItem) error
}

// PodcastLiveItemRepository manages podcast:liveItem entries (one per channel).
type PodcastLiveItemRepository interface {
	GetByChannel(ctx context.Context, channelID string) (*PodcastLiveItem, error)
	Upsert(ctx context.Context, item *PodcastLiveItem) error
	DeleteByChannel(ctx context.Context, channelID string) error
}

type PodcastStatus string

const (
	PodcastStatusNew         PodcastStatus = "new"
	PodcastStatusDownloading PodcastStatus = "downloading"
	PodcastStatusCompleted   PodcastStatus = "completed"
	PodcastStatusError       PodcastStatus = "error"
	PodcastStatusSkipped     PodcastStatus = "skipped"
	PodcastStatusDeleted     PodcastStatus = "deleted"
)

type PodcastChannel struct {
	ID           string        `structs:"id"             json:"id"`
	URL          string        `structs:"url"            json:"url"`
	Title        string        `structs:"title"          json:"title"`
	Description  string        `structs:"description"    json:"description"`
	ImageURL     string        `structs:"image_url"      json:"imageUrl"`
	Status       PodcastStatus `structs:"status"         json:"status"`
	ErrorMessage string        `structs:"error_message"  json:"errorMessage"`
	CreatedAt    time.Time     `structs:"created_at"     json:"createdAt"`
	UpdatedAt    time.Time     `structs:"updated_at"     json:"updatedAt"`

	// Podcasting 2.0 — Tier 1
	PodcastGUID string `structs:"podcast_guid" json:"podcastGuid,omitempty"`

	// Podcasting 2.0 — Tier 2
	Locked          bool   `structs:"locked"           json:"locked,omitempty"`
	LockedOwner     string `structs:"locked_owner"     json:"lockedOwner,omitempty"`
	Medium          string `structs:"medium"           json:"medium,omitempty"`
	FundingURL      string `structs:"funding_url"      json:"fundingUrl,omitempty"`
	FundingText     string `structs:"funding_text"     json:"fundingText,omitempty"`
	UpdateFrequency string `structs:"update_frequency" json:"updateFrequency,omitempty"`
	UpdateRRule     string `structs:"update_rrule"     json:"updateRRule,omitempty"`
	Complete        bool   `structs:"complete"         json:"complete,omitempty"`
	LocationName    string `structs:"location_name"    json:"locationName,omitempty"`
	LocationGeo     string `structs:"location_geo"     json:"locationGeo,omitempty"`
	LocationOSM     string `structs:"location_osm"     json:"locationOsm,omitempty"`
	License         string `structs:"license"          json:"license,omitempty"`
	PublisherName   string `structs:"publisher_name"   json:"publisherName,omitempty"`
	PublisherURL    string `structs:"publisher_url"    json:"publisherUrl,omitempty"`

	// Podcasting 2.0 — Tier 3
	UsesPodping bool                `structs:"uses_podping" json:"usesPodping,omitempty"`
	Podroll     PodcastPodrollItems `structs:"-"            json:"podroll,omitempty"`
	LiveItem    *PodcastLiveItem    `structs:"-"            json:"liveItem,omitempty"`

	// loaded separately
	Episodes     PodcastEpisodes     `structs:"-" json:"episodes,omitempty"`
	Persons      PodcastPersons      `structs:"-" json:"persons,omitempty"`
	FundingItems PodcastFundingItems `structs:"-" json:"funding,omitempty"`
	Images       PodcastImages       `structs:"-" json:"images,omitempty"`
}

type PodcastEpisode struct {
	ID              string        `structs:"id"               json:"id"`
	ChannelID       string        `structs:"channel_id"       json:"channelId"`
	StreamID        string        `structs:"stream_id"        json:"streamId"`
	GUID            string        `structs:"guid"             json:"guid"`
	Title           string        `structs:"title"            json:"title"`
	Description     string        `structs:"description"      json:"description"`
	PublishDate     time.Time     `structs:"publish_date"     json:"publishDate"`
	Duration        int           `structs:"duration"         json:"duration"`
	Size            int64         `structs:"size"             json:"size"`
	BitRate         int           `structs:"bit_rate"         json:"bitRate"`
	Suffix          string        `structs:"suffix"           json:"suffix"`
	ContentType     string        `structs:"content_type"     json:"contentType"`
	Path            string        `structs:"path"             json:"path"`
	EnclosureURL    string        `structs:"enclosure_url"    json:"enclosureUrl"`
	DownloadedBytes int64         `structs:"downloaded_bytes" json:"downloadedBytes"`
	Status          PodcastStatus `structs:"status"           json:"status"`
	ErrorMessage    string        `structs:"error_message"    json:"errorMessage"`
	CreatedAt       time.Time     `structs:"created_at"       json:"createdAt"`
	UpdatedAt       time.Time     `structs:"updated_at"       json:"updatedAt"`

	// Podcasting 2.0 — Tier 1
	Season         int    `structs:"season"          json:"season,omitempty"`
	SeasonName     string `structs:"season_name"     json:"seasonName,omitempty"`
	EpisodeNumber  string `structs:"episode_number"  json:"episodeNumber,omitempty"`
	EpisodeDisplay string `structs:"episode_display" json:"episodeDisplay,omitempty"`
	ChaptersURL    string `structs:"chapters_url"    json:"chaptersUrl,omitempty"`
	ChaptersType   string `structs:"chapters_type"   json:"chaptersType,omitempty"`

	// Podcasting 2.0 — Tier 2
	SoundbiteStart float64 `structs:"soundbite_start" json:"soundbiteStart,omitempty"`
	SoundbiteDur   float64 `structs:"soundbite_dur"   json:"soundbiteDur,omitempty"`
	SoundbiteTitle string  `structs:"soundbite_title" json:"soundbiteTitle,omitempty"`
	LocationName   string  `structs:"location_name"   json:"locationName,omitempty"`
	LocationGeo    string  `structs:"location_geo"    json:"locationGeo,omitempty"`
	LocationOSM    string  `structs:"location_osm"    json:"locationOsm,omitempty"`
	License        string  `structs:"license"         json:"license,omitempty"`

	// loaded separately
	Transcripts PodcastTranscripts `structs:"-" json:"transcripts,omitempty"`
	Persons     PodcastPersons     `structs:"-" json:"persons,omitempty"`
	Images      PodcastImages      `structs:"-" json:"images,omitempty"`
}

type PodcastTranscript struct {
	ID        string    `structs:"id"         json:"id"`
	EpisodeID string    `structs:"episode_id" json:"episodeId"`
	URL       string    `structs:"url"        json:"url"`
	MimeType  string    `structs:"mime_type"  json:"type"`
	Language  string    `structs:"language"   json:"language,omitempty"`
	Rel       string    `structs:"rel"        json:"rel,omitempty"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
}

type PodcastPerson struct {
	ID        string    `structs:"id"         json:"id"`
	ChannelID string    `structs:"channel_id" json:"channelId,omitempty"`
	EpisodeID string    `structs:"episode_id" json:"episodeId,omitempty"`
	Name      string    `structs:"name"       json:"name"`
	Role      string    `structs:"role"       json:"role,omitempty"`
	Group     string    `structs:"group_name" db:"group_name" json:"group,omitempty"`
	Img       string    `structs:"img"        json:"img,omitempty"`
	Href      string    `structs:"href"       json:"href,omitempty"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
}

type PodcastFundingItem struct {
	ID        string    `structs:"id"         json:"id"`
	ChannelID string    `structs:"channel_id" json:"channelId"`
	URL       string    `structs:"url"        json:"url"`
	Text      string    `structs:"text"       json:"text,omitempty"`
	SortOrder int       `structs:"sort_order" json:"sortOrder"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
}

type PodcastImage struct {
	ID        string    `structs:"id"         json:"id"`
	ChannelID string    `structs:"channel_id" json:"channelId,omitempty"`
	EpisodeID string    `structs:"episode_id" json:"episodeId,omitempty"`
	URL       string    `structs:"url"        json:"url"`
	Width     int       `structs:"width"      json:"width,omitempty"`
	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
}

type PodcastChannels []PodcastChannel
type PodcastEpisodes []PodcastEpisode
type PodcastTranscripts []PodcastTranscript
type PodcastPersons []PodcastPerson
type PodcastFundingItems []PodcastFundingItem
type PodcastImages []PodcastImage

type PodcastChannelRepository interface {
	rest.Repository[PodcastChannel]
	rest.Persistable[PodcastChannel]
	Get(ctx context.Context, id string) (*PodcastChannel, error)
	GetAll(ctx context.Context, withEpisodes bool) (PodcastChannels, error)
	ExistsByURL(ctx context.Context, url string) (bool, error)
	Create(ctx context.Context, channel *PodcastChannel) error
	UpdateChannel(ctx context.Context, channel *PodcastChannel) error
	Delete(ctx context.Context, ids ...string) error
}

type PodcastEpisodeRepository interface {
	Get(ctx context.Context, id string) (*PodcastEpisode, error)
	GetNewest(ctx context.Context, count int) (PodcastEpisodes, error)
	GetByChannel(ctx context.Context, channelID string) (PodcastEpisodes, error)
	GetByChannels(ctx context.Context, channelIDs []string) (PodcastEpisodes, error)
	GetByGUID(ctx context.Context, channelID, guid string) (*PodcastEpisode, error)
	Create(ctx context.Context, ep *PodcastEpisode) error
	Update(ctx context.Context, ep *PodcastEpisode) error
	Delete(ctx context.Context, id string) error
}

type PodcastTranscriptRepository interface {
	GetByEpisode(ctx context.Context, episodeID string) (PodcastTranscripts, error)
	GetByEpisodes(ctx context.Context, episodeIDs []string) (PodcastTranscripts, error)
	Save(ctx context.Context, transcripts []PodcastTranscript) error
	DeleteByEpisode(ctx context.Context, episodeID string) error
}

type PodcastPersonRepository interface {
	GetByChannel(ctx context.Context, channelID string) (PodcastPersons, error)
	GetByEpisode(ctx context.Context, episodeID string) (PodcastPersons, error)
	GetByEpisodes(ctx context.Context, episodeIDs []string) (PodcastPersons, error)
	SaveForChannel(ctx context.Context, channelID string, persons []PodcastPerson) error
	SaveForEpisode(ctx context.Context, episodeID string, persons []PodcastPerson) error
}

type PodcastFundingRepository interface {
	GetByChannel(ctx context.Context, channelID string) (PodcastFundingItems, error)
	GetByChannels(ctx context.Context, channelIDs []string) (PodcastFundingItems, error)
	SaveForChannel(ctx context.Context, channelID string, items []PodcastFundingItem) error
}

type PodcastImageRepository interface {
	GetByChannel(ctx context.Context, channelID string) (PodcastImages, error)
	GetByChannels(ctx context.Context, channelIDs []string) (PodcastImages, error)
	GetByEpisode(ctx context.Context, episodeID string) (PodcastImages, error)
	GetByEpisodes(ctx context.Context, episodeIDs []string) (PodcastImages, error)
	SaveForChannel(ctx context.Context, channelID string, images []PodcastImage) error
	SaveForEpisode(ctx context.Context, episodeID string, images []PodcastImage) error
}
