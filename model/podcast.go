package model

import (
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
)

const (
	podcastIdPrefix = "pd-"
	episodeIdPrefix = "pe-"
)

type Podcast struct {
	Annotations `structs:"-"`

	ID          string               `structs:"id" json:"id"`
	Url         string               `structs:"url" json:"url"`
	Title       string               `structs:"title" json:"title,omitempty"`
	Description string               `structs:"description" json:"description,omitempty"`
	ImageUrl    string               `structs:"image_url" json:"imageUrl,omitempty"`
	State       consts.PodcastStatus `structs:"state" json:"state"`
	Error       string               `structs:"error" json:"error,omitempty"`

	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
	UpdatedAt time.Time `structs:"updated_at" json:"updatedAt"`

	PodcastEpisodes PodcastEpisodes `structs:"-" json:"episodes,omitempty"`
}

type Podcasts []Podcast

func cleanId(id string) string {
	return filepath.Join("/", id)[1:]
}

func (p *Podcast) AbsolutePath() string {
	return path.Join(conf.Server.Podcast.Path, cleanId(p.ID))
}

func (p Podcast) CoverArtID() ArtworkID {
	return artworkIDFromPodcast(p)
}

func (p *Podcast) ExternalId() string {
	return podcastIdPrefix + p.ID
}

func IsPodcastId(id string) bool {
	return conf.Server.Podcast.Enabled && strings.HasPrefix(id, podcastIdPrefix) && len(id) > 3
}

func ExtractExternalId(external string) string {
	return external[3:]
}

type PodcastRepository interface {
	AnnotatedRepository
	ResourceRepository

	// Cleans up podcast annotations. This will NOT remove
	// podcast episode annotations/bookmarks
	Cleanup() error
	CountAll(options ...QueryOptions) (int64, error)
	Delete(id string) error
	DeleteInternal(id string) error
	Get(id string, withEpisodes bool) (*Podcast, error)
	GetAll(withEpisodes bool, options ...QueryOptions) (Podcasts, error)
	Put(p *Podcast) error
	PutInternal(p *Podcast) error
}

type PodcastEpisode struct {
	Annotations  `structs:"-"`
	Bookmarkable `structs:"-"`

	ID          string               `structs:"id" json:"id"`
	Guid        string               `structs:"guid" json:"guid"`
	PodcastId   string               `structs:"podcast_id" json:"podcastId"`
	Url         string               `structs:"url" json:"url"`
	Title       string               `structs:"title" json:"title,omitempty"`
	Description string               `structs:"description" json:"description,omitempty"`
	ImageUrl    string               `structs:"image_url" json:"image_url"`
	PublishDate *time.Time           `structs:"publish_date" json:"publishDate,omitempty"`
	Duration    float32              `structs:"duration" json:"duration"`
	Suffix      string               `structs:"suffix" json:"suffix"`
	Size        int64                `structs:"size" json:"size"`
	BitRate     int                  `structs:"bit_rate" json:"bitRate"`
	State       consts.PodcastStatus `structs:"state" json:"state"`
	Error       string               `structs:"error" json:"error,omitempty"`

	CreatedAt time.Time `structs:"created_at" json:"createdAt"`
	UpdatedAt time.Time `structs:"updated_at" json:"updatedAt"`
}

func (pe *PodcastEpisode) BasePath() string {
	return path.Join(
		cleanId(pe.PodcastId),
		cleanId(pe.ID+"."+pe.Suffix),
	)
}

func (pe *PodcastEpisode) AbsolutePath() string {
	return path.Join(conf.Server.Podcast.Path, pe.BasePath())
}

func (pe PodcastEpisode) CoverArtID() ArtworkID {
	return artworkIDFromPodcastEpisode(pe)
}

func (pe *PodcastEpisode) ExternalId() string {
	return episodeIdPrefix + pe.ID
}

func (pe *PodcastEpisode) ToMediaFile() *MediaFile {
	mf := &MediaFile{
		ID:        pe.ExternalId(),
		AlbumID:   "pd-" + pe.PodcastId,
		BitRate:   pe.BitRate,
		Duration:  pe.Duration,
		Path:      pe.AbsolutePath(),
		Size:      pe.Size,
		Suffix:    pe.Suffix,
		Title:     pe.Title,
		CreatedAt: pe.CreatedAt,
		UpdatedAt: pe.UpdatedAt,
	}

	if pe.PublishDate != nil {
		mf.Year = pe.PublishDate.Year()
	}

	if pe.PlayCount > 0 {
		mf.PlayCount = pe.PlayCount
	}

	if pe.PlayDate != nil {
		mf.PlayDate = pe.PlayDate
	}

	if pe.Starred {
		mf.Starred = pe.Starred
		mf.StarredAt = pe.StarredAt
	}

	mf.BookmarkPosition = pe.BookmarkPosition
	mf.Rating = pe.Rating

	return mf
}

func IsPodcastEpisodeId(id string) bool {
	return conf.Server.Podcast.Enabled && strings.HasPrefix(id, episodeIdPrefix) && len(id) > 3
}

type PodcastEpisodes []PodcastEpisode

type PodcastEpisodeRepository interface {
	AnnotatedRepository
	BookmarkableRepository
	ResourceRepository

	// Removes podcast episode annotations and bookmarks
	Cleanup() error
	CountAll(options ...QueryOptions) (int64, error)
	Delete(id string) error
	Get(id string) (*PodcastEpisode, error)
	GetAll(options ...QueryOptions) (PodcastEpisodes, error)
	GetEpisodeGuids(id string) (map[string]bool, error)
	GetNewestEpisodes(count int) (PodcastEpisodes, error)
	Put(p *PodcastEpisode) error
}
