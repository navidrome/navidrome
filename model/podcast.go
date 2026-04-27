package model

import "time"

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
	ID           string          `structs:"id"            json:"id"`
	URL          string          `structs:"url"           json:"url"`
	Title        string          `structs:"title"         json:"title"`
	Description  string          `structs:"description"   json:"description"`
	ImageURL     string          `structs:"image_url"     json:"imageUrl"`
	Status       PodcastStatus   `structs:"status"        json:"status"`
	ErrorMessage string          `structs:"error_message" json:"errorMessage"`
	CreatedAt    time.Time       `structs:"created_at"    json:"createdAt"`
	UpdatedAt    time.Time       `structs:"updated_at"    json:"updatedAt"`
	Episodes     PodcastEpisodes `structs:"-"             json:"episodes,omitempty"`
}

type PodcastEpisode struct {
	ID           string        `structs:"id"             json:"id"`
	ChannelID    string        `structs:"channel_id"     json:"channelId"`
	StreamID     string        `structs:"stream_id"      json:"streamId"`
	GUID         string        `structs:"guid"           json:"guid"`
	Title        string        `structs:"title"          json:"title"`
	Description  string        `structs:"description"    json:"description"`
	PublishDate  time.Time     `structs:"publish_date"   json:"publishDate"`
	Duration     int           `structs:"duration"       json:"duration"`
	Size         int64         `structs:"size"           json:"size"`
	BitRate      int           `structs:"bit_rate"       json:"bitRate"`
	Suffix       string        `structs:"suffix"         json:"suffix"`
	ContentType  string        `structs:"content_type"   json:"contentType"`
	Path            string        `structs:"path"             json:"path"`
	EnclosureURL    string        `structs:"enclosure_url"    json:"enclosureUrl"`
	DownloadedBytes int64         `structs:"downloaded_bytes" json:"downloadedBytes"`
	Status       PodcastStatus `structs:"status"         json:"status"`
	ErrorMessage string        `structs:"error_message"  json:"errorMessage"`
	CreatedAt    time.Time     `structs:"created_at"     json:"createdAt"`
	UpdatedAt    time.Time     `structs:"updated_at"     json:"updatedAt"`
}

type PodcastChannels []PodcastChannel
type PodcastEpisodes []PodcastEpisode

type PodcastChannelRepository interface {
	Get(id string) (*PodcastChannel, error)
	GetAll(withEpisodes bool) (PodcastChannels, error)
	ExistsByURL(url string) (bool, error)
	Create(channel *PodcastChannel) error
	UpdateChannel(channel *PodcastChannel) error
	Delete(id string) error
}

type PodcastEpisodeRepository interface {
	Get(id string) (*PodcastEpisode, error)
	GetNewest(count int) (PodcastEpisodes, error)
	GetByChannel(channelID string) (PodcastEpisodes, error)
	GetByChannels(channelIDs []string) (PodcastEpisodes, error)
	GetByGUID(channelID, guid string) (*PodcastEpisode, error)
	Create(ep *PodcastEpisode) error
	Update(ep *PodcastEpisode) error
	Delete(id string) error
}
