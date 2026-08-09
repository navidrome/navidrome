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

// PodcastChannel is shared, admin-managed feed infrastructure - one row per RSS feed regardless
// of how many users subscribe to it, so the feed is only ever fetched/refreshed once. Per-user
// concerns (download policy, retention limits) live on PodcastSubscription instead - see that
// type and FEATURE_ROADMAP.md's podcast subscriptions entry for why.
type PodcastChannel struct {
	ID               string               `structs:"id" json:"id"`
	Url              string               `structs:"url" json:"url"`
	Title            string               `structs:"title" json:"title"`
	Description      string               `structs:"description" json:"description,omitempty"`
	CoverArtUrl      string               `structs:"cover_art_url" json:"coverArtUrl,omitempty"`
	UploadedImage    string               `structs:"uploaded_image" json:"uploadedImage,omitempty"`
	OriginalImageUrl string               `structs:"original_image_url" json:"originalImageUrl,omitempty"`
	HomePageUrl      string               `structs:"home_page_url" json:"homePageUrl,omitempty"`
	Status           PodcastChannelStatus `structs:"status" json:"status"`
	ErrorMessage     string               `structs:"error_message" json:"errorMessage,omitempty"`
	LastCheckedAt    *time.Time           `structs:"last_checked_at" json:"lastCheckedAt,omitempty"`
	CreatedAt        time.Time            `structs:"created_at" json:"createdAt"`
	UpdatedAt        time.Time            `structs:"updated_at" json:"updatedAt"`

	Episodes PodcastEpisodes `structs:"-" json:"episodes,omitempty"`
	// Subscription is the current user's own subscription to this channel, populated by
	// repository reads that scope to the caller (nil for an admin viewing a channel they haven't
	// personally subscribed to).
	Subscription *PodcastSubscription `structs:"-" json:"subscription,omitempty"`
}

func (c PodcastChannel) CoverArtID() ArtworkID {
	return artworkIDFromPodcastChannel(c)
}

func (c PodcastChannel) UploadedImagePath() string {
	return UploadedImagePath(consts.EntityPodcastChannel, c.UploadedImage)
}

type PodcastChannels []PodcastChannel

// PodcastSubscription is one user's subscription to a shared PodcastChannel - subscribing is
// what makes a channel show up in that user's own podcast list, and carries that user's own
// download/retention preferences for it. Two different users subscribed to the same channel can
// have completely different policies; the underlying feed/files stay shared regardless.
type PodcastSubscription struct {
	ID             string                `structs:"id" json:"id"`
	ChannelID      string                `structs:"channel_id" json:"channelId"`
	UserID         string                `structs:"user_id" json:"userId"`
	DownloadPolicy PodcastDownloadPolicy `structs:"download_policy" json:"downloadPolicy"`
	RetentionCount int                   `structs:"retention_count" json:"retentionCount"`
	RetentionDays  int                   `structs:"retention_days" json:"retentionDays"`
	MaxStorageMB   int                   `structs:"max_storage_mb" json:"maxStorageMb"`
	CreatedAt      time.Time             `structs:"created_at" json:"createdAt"`
	UpdatedAt      time.Time             `structs:"updated_at" json:"updatedAt"`
}

// HasRetentionLimit reports whether this subscription has any retention constraint configured at
// all - a subscription with none never has episodes evicted from the subscriber's own list.
func (s PodcastSubscription) HasRetentionLimit() bool {
	return s.RetentionCount > 0 || s.RetentionDays > 0 || s.MaxStorageMB > 0
}

type PodcastSubscriptions []PodcastSubscription

// PodcastSubscriptionRepository is intentionally not a ResourceRepository (no generic REST CRUD
// wiring) - subscribe/unsubscribe are user-scoped operations with their own dedicated native API
// routes (mirroring /user/{id}/library), not a plain resource collection.
type PodcastSubscriptionRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Get(id string) (*PodcastSubscription, error)
	GetAll(options ...QueryOptions) (PodcastSubscriptions, error)
	// FindByChannelAndUser returns the given user's subscription to the given channel, or
	// ErrNotFound if they're not subscribed.
	FindByChannelAndUser(channelID, userID string) (*PodcastSubscription, error)
	// FindByChannel returns every subscription to the given channel, across all users - used by
	// the per-subscriber download-policy fan-out and retention eviction.
	FindByChannel(channelID string) (PodcastSubscriptions, error)
	Put(s *PodcastSubscription) error
	// Delete removes a single subscription (unsubscribe). Returns the number of subscriptions
	// remaining for that subscription's channel afterward, so callers can decide whether the
	// shared channel itself should now be torn down.
	Delete(id string) (remainingForChannel int64, err error)
}

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

	// PlayCount/PlayDate/Downloaded/DownloadedAt come from the per-user annotation table (the
	// same mechanism songs use), not a podcast_episode column - populated by repository reads,
	// excluded from writes.
	PlayCount int64      `structs:"-" json:"playCount,omitempty"`
	PlayDate  *time.Time `structs:"-" json:"playDate,omitempty"`
	// Downloaded/DownloadedAt is the CURRENT USER's own "this is in my downloaded list" flag -
	// distinct from DownloadStatus, which tracks whether the underlying file exists on disk at
	// all (a shared, server-wide fact, since the file itself is only ever fetched once regardless
	// of how many subscribers want it). A file can exist on disk (DownloadStatus=downloaded)
	// while Downloaded is false for a given caller, if some other subscriber requested it first.
	Downloaded   bool       `structs:"-" json:"downloaded,omitempty"`
	DownloadedAt *time.Time `structs:"-" json:"downloadedAt,omitempty"`
}

// IsDownloaded reports whether the underlying file exists on disk at all - a shared,
// server-wide fact, not specific to the current user. See Downloaded for the per-user flag.
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
	// Delete removes the shared channel and (via cascade) its episodes - callers must have
	// already confirmed no subscriptions remain (see PodcastSubscriptionRepository.Delete).
	Delete(id string) error
	// Get/GetAll/GetWithEpisodes scope to the current user's own subscriptions - admins see
	// every channel regardless of whether they've personally subscribed (management access),
	// matching the admin-bypass convention used throughout this fork's permission system.
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
	// ResetPlayCount clears the current user's play_count/play_date for this
	// episode, marking it unlistened again - the explicit counterpart to
	// IncPlayCount, for clients that want to toggle listened state manually
	// rather than relying on the passive stream.view side-effect.
	ResetPlayCount(itemID string) error
	// SetDownloaded sets/clears the given user's own "in my downloaded list" flag - does not
	// touch the shared file on disk. See PodcastEpisode.Downloaded.
	SetDownloaded(userID string, downloaded bool, ids ...string) error
	// AnyUserWantsDownload reports whether any user, anywhere, still has this episode flagged as
	// downloaded - used to decide whether the shared file on disk can finally be deleted (only
	// once nobody's personal downloaded list references it anymore).
	AnyUserWantsDownload(episodeID string) (bool, error)
	// GetDownloadedForUser returns userID's own downloaded episodes for the given channel,
	// newest-first by publish date - used by per-subscription retention eviction, which runs
	// against a system/admin context with no single "current user" to scope GetAll's usual
	// annotation join to.
	GetDownloadedForUser(userID, channelID string) (PodcastEpisodes, error)
}
