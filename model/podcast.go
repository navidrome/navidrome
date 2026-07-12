package model

import (
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
)

type PodcastDownloadPolicy string

const (
	PodcastDownloadPolicyNone PodcastDownloadPolicy = "none" // stream-only, never auto-download
	PodcastDownloadPolicyNew  PodcastDownloadPolicy = "new"  // auto-download new episodes as they appear
	PodcastDownloadPolicyAll  PodcastDownloadPolicy = "all"  // download new + backfill all existing episodes
)

// PodcastChannelStatus mirrors the Subsonic API's podcast-channel "status" enum.
type PodcastChannelStatus string

const (
	PodcastChannelStatusNew         PodcastChannelStatus = "new"
	PodcastChannelStatusDownloading PodcastChannelStatus = "downloading" // a refresh/feed-fetch is in progress
	PodcastChannelStatusCompleted   PodcastChannelStatus = "completed"
	PodcastChannelStatusError       PodcastChannelStatus = "error"
)

type PodcastChannel struct {
	ID               string                `structs:"id" json:"id"`
	Url              string                `structs:"url" json:"url"`
	Title            string                `structs:"title" json:"title"`
	Description      string                `structs:"description" json:"description,omitempty"`
	CoverArtUrl      string                `structs:"cover_art_url" json:"coverArtUrl,omitempty"`
	UploadedImage    string                `structs:"uploaded_image" json:"uploadedImage,omitempty"`
	OriginalImageUrl string                `structs:"original_image_url" json:"originalImageUrl,omitempty"`
	HomePageUrl      string                `structs:"home_page_url" json:"homePageUrl,omitempty"`
	Status           PodcastChannelStatus  `structs:"status" json:"status"`
	ErrorMessage     string                `structs:"error_message" json:"errorMessage,omitempty"`
	DownloadPolicy   PodcastDownloadPolicy `structs:"download_policy" json:"downloadPolicy"`
	RetentionCount   int                   `structs:"retention_count" json:"retentionCount"`
	RetentionDays    int                   `structs:"retention_days" json:"retentionDays"`
	MaxStorageMB     int                   `structs:"max_storage_mb" json:"maxStorageMb"`
	LastCheckedAt    *time.Time            `structs:"last_checked_at" json:"lastCheckedAt,omitempty"`
	CreatedAt        time.Time             `structs:"created_at" json:"createdAt"`
	UpdatedAt        time.Time             `structs:"updated_at" json:"updatedAt"`

	Episodes PodcastEpisodes `structs:"-" json:"episodes,omitempty"`
}

func (c PodcastChannel) CoverArtID() ArtworkID {
	return artworkIDFromPodcastChannel(c)
}

func (c PodcastChannel) UploadedImagePath() string {
	return UploadedImagePath(consts.EntityPodcastChannel, c.UploadedImage)
}

type PodcastChannels []PodcastChannel

type PodcastEpisodeDownloadStatus string

const (
	PodcastEpisodeNotDownloaded PodcastEpisodeDownloadStatus = "not_downloaded"
	PodcastEpisodeQueued        PodcastEpisodeDownloadStatus = "queued"
	PodcastEpisodeDownloading   PodcastEpisodeDownloadStatus = "downloading"
	PodcastEpisodeDownloaded    PodcastEpisodeDownloadStatus = "downloaded"
	PodcastEpisodeDownloadError PodcastEpisodeDownloadStatus = "error"
	PodcastEpisodeDeleted       PodcastEpisodeDownloadStatus = "deleted"
)

type PodcastEpisode struct {
	ID             string                       `structs:"id" json:"id"`
	ChannelID      string                       `structs:"channel_id" json:"channelId"`
	Guid           string                       `structs:"guid" json:"guid"`
	Title          string                       `structs:"title" json:"title"`
	Description    string                       `structs:"description" json:"description,omitempty"`
	EnclosureUrl   string                       `structs:"enclosure_url" json:"enclosureUrl"`
	ContentType    string                       `structs:"content_type" json:"contentType,omitempty"`
	Size           int64                        `structs:"size" json:"size,omitempty"`
	Duration       float32                      `structs:"duration" json:"duration,omitempty"`
	PublishDate    *time.Time                   `structs:"publish_date" json:"publishDate,omitempty"`
	DownloadStatus PodcastEpisodeDownloadStatus `structs:"download_status" json:"downloadStatus"`
	ErrorMessage   string                       `structs:"error_message" json:"errorMessage,omitempty"`
	Path           string                       `structs:"path" json:"-"`
	Suffix         string                       `structs:"suffix" json:"suffix,omitempty"`
	BitRate        int                          `structs:"bit_rate" json:"bitRate,omitempty"`
	CreatedAt      time.Time                    `structs:"created_at" json:"createdAt"`
	UpdatedAt      time.Time                    `structs:"updated_at" json:"updatedAt"`

	// PlayCount/PlayDate come from the per-user annotation table (the same
	// mechanism songs use), not a podcast_episode column - populated by
	// repository reads, excluded from writes.
	PlayCount int64      `structs:"-" json:"playCount,omitempty"`
	PlayDate  *time.Time `structs:"-" json:"playDate,omitempty"`
}

func (e PodcastEpisode) IsDownloaded() bool {
	return e.DownloadStatus == PodcastEpisodeDownloaded
}

// IsListened reports whether the current user has ever played this episode.
func (e PodcastEpisode) IsListened() bool {
	return e.PlayCount > 0
}

// AbsolutePath mirrors MediaFile.AbsolutePath(), joining the episode's
// relative Path against the podcasts storage root.
func (e PodcastEpisode) AbsolutePath() string {
	if e.Path == "" {
		return ""
	}
	return filepath.Join(conf.Server.Podcasts.StorageFolder.String(), e.Path)
}

type PodcastEpisodes []PodcastEpisode

type PodcastChannelRepository interface {
	ResourceRepository
	CountAll(options ...QueryOptions) (int64, error)
	Delete(id string) error
	Get(id string) (*PodcastChannel, error)
	GetAll(options ...QueryOptions) (PodcastChannels, error)
	GetWithEpisodes(id string) (*PodcastChannel, error)
	Put(c *PodcastChannel, colsToUpdate ...string) error
	FindByUrl(url string) (*PodcastChannel, error)
}

type PodcastEpisodeRepository interface {
	ResourceRepository
	CountAll(options ...QueryOptions) (int64, error)
	Delete(id string) error
	Get(id string) (*PodcastEpisode, error)
	GetAll(options ...QueryOptions) (PodcastEpisodes, error)
	Put(e *PodcastEpisode, colsToUpdate ...string) error
	FindByGuid(channelID, guid string) (*PodcastEpisode, error)
	GetNewest(count int) (PodcastEpisodes, error)
	// IncPlayCount marks the episode as listened to (by the current user),
	// mirroring MediaFile's play-tracking mechanism via the shared
	// annotation table. Podcast episodes don't support starring/rating (no
	// average_rating column), so only this piece of AnnotatedRepository is
	// exposed.
	IncPlayCount(itemID string, ts time.Time) error
}
