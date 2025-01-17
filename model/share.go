package model

import (
	"cmp"
	"fmt"
	"strings"
	"time"

	"github.com/navidrome/navidrome/utils/random"
)

type Share struct {
	ID            string     `structs:"id" json:"id,omitempty"`
	UserID        string     `structs:"user_id" json:"userId,omitempty"`
	Username      string     `structs:"-" json:"username,omitempty"`
	Description   string     `structs:"description" json:"description,omitempty"`
	Downloadable  bool       `structs:"downloadable" json:"downloadable"`
	ExpiresAt     *time.Time `structs:"expires_at" json:"expiresAt,omitempty"`
	LastVisitedAt *time.Time `structs:"last_visited_at" json:"lastVisitedAt,omitempty"`
	ResourceIDs   string     `structs:"resource_ids" json:"resourceIds,omitempty"`
	ResourceType  string     `structs:"resource_type" json:"resourceType,omitempty"`
	Contents      string     `structs:"contents" json:"contents,omitempty"`
	Format        string     `structs:"format" json:"format,omitempty"`
	MaxBitRate    int        `structs:"max_bit_rate" json:"maxBitRate,omitempty"`
	VisitCount    int        `structs:"visit_count" json:"visitCount,omitempty"`
	CreatedAt     time.Time  `structs:"created_at" json:"createdAt,omitempty"`
	UpdatedAt     time.Time  `structs:"updated_at" json:"updatedAt,omitempty"`
	Tracks        MediaFiles `structs:"-" json:"tracks,omitempty"`
	Albums        Albums     `structs:"-" json:"albums,omitempty"`
	URL           string     `structs:"-" json:"-"`
	ImageURL      string     `structs:"-" json:"-"`
}

func (s Share) CoverArtID() ArtworkID {
	ids := strings.SplitN(s.ResourceIDs, ",", 2)
	if len(ids) == 0 {
		return ArtworkID{}
	}
	switch s.ResourceType {
	case "album":
		return Album{ID: ids[0]}.CoverArtID()
	case "playlist":
		return Playlist{ID: ids[0]}.CoverArtID()
	case "artist":
		return Artist{ID: ids[0]}.CoverArtID()
	}
	rnd := random.Int64N(len(s.Tracks))
	return s.Tracks[rnd].CoverArtID()
}

type Shares []Share

// ToM3U8 exports the playlist to the Extended M3U8 format, as specified in
// https://docs.fileformat.com/audio/m3u/#extended-m3u
func (s Share) ToM3U8() string {
	buf := strings.Builder{}
	buf.WriteString("#EXTM3U\n")
	buf.WriteString(fmt.Sprintf("#PLAYLIST:%s\n", cmp.Or(s.Description, s.ID)))
	for _, t := range s.Tracks {
		buf.WriteString(fmt.Sprintf("#EXTINF:%.f,%s - %s\n", t.Duration, t.Artist, t.Title))
		buf.WriteString(t.Path + "\n")
	}
	return buf.String()
}

type ShareRepository interface {
	Exists(id string) (bool, error)
	Get(id string) (*Share, error)
	GetAll(options ...QueryOptions) (Shares, error)
	CountAll(options ...QueryOptions) (int64, error)
}
