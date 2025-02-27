package model

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model/criteria"
)

type Playlist struct {
	ID        string         `structs:"id" json:"id"`
	Name      string         `structs:"name" json:"name"`
	Comment   string         `structs:"comment" json:"comment"`
	Duration  float32        `structs:"duration" json:"duration"`
	Size      int64          `structs:"size" json:"size"`
	SongCount int            `structs:"song_count" json:"songCount"`
	OwnerName string         `structs:"-" json:"ownerName"`
	OwnerID   string         `structs:"owner_id" json:"ownerId"`
	Public    bool           `structs:"public" json:"public"`
	Tracks    PlaylistTracks `structs:"-" json:"tracks,omitempty"`
	Path      string         `structs:"path" json:"path"`
	Sync      bool           `structs:"sync" json:"sync"`
	CreatedAt time.Time      `structs:"created_at" json:"createdAt"`
	UpdatedAt time.Time      `structs:"updated_at" json:"updatedAt"`

	// SmartPlaylist attributes
	Rules       *criteria.Criteria `structs:"rules" json:"rules"`
	EvaluatedAt *time.Time         `structs:"evaluated_at" json:"evaluatedAt"`
}

func (pls Playlist) IsSmartPlaylist() bool {
	return pls.Rules != nil && pls.Rules.Expression != nil
}

func (pls Playlist) MediaFiles() MediaFiles {
	if len(pls.Tracks) == 0 {
		return nil
	}
	return pls.Tracks.MediaFiles()
}

func (pls *Playlist) RemoveTracks(idxToRemove []int) {
	var newTracks PlaylistTracks
	for i, t := range pls.Tracks {
		if slices.Contains(idxToRemove, i) {
			continue
		}
		newTracks = append(newTracks, t)
	}
	pls.Tracks = newTracks
}

// ToM3U8 exports the playlist to the Extended M3U8 format, as specified in
// https://docs.fileformat.com/audio/m3u/#extended-m3u
func (pls *Playlist) ToM3U8() string {
	buf := strings.Builder{}
	buf.WriteString("#EXTM3U\n")
	buf.WriteString(fmt.Sprintf("#PLAYLIST:%s\n", pls.Name))
	for _, t := range pls.Tracks {
		buf.WriteString(fmt.Sprintf("#EXTINF:%.f,%s - %s\n", t.Duration, t.Artist, t.Title))
		buf.WriteString(t.AbsolutePath() + "\n")
	}
	return buf.String()
}

func (pls *Playlist) AddTracks(mediaFileIds []string) {
	pos := len(pls.Tracks)
	for _, mfId := range mediaFileIds {
		pos++
		t := PlaylistTrack{
			ID:          strconv.Itoa(pos),
			MediaFileID: mfId,
			MediaFile:   MediaFile{ID: mfId},
			PlaylistID:  pls.ID,
		}
		pls.Tracks = append(pls.Tracks, t)
	}
}

func (pls *Playlist) AddMediaFiles(mfs MediaFiles) {
	pos := len(pls.Tracks)
	for _, mf := range mfs {
		pos++
		t := PlaylistTrack{
			ID:          strconv.Itoa(pos),
			MediaFileID: mf.ID,
			MediaFile:   mf,
			PlaylistID:  pls.ID,
		}
		pls.Tracks = append(pls.Tracks, t)
	}
}

func (pls Playlist) CoverArtID() ArtworkID {
	return artworkIDFromPlaylist(pls)
}

type Playlists []Playlist

type PlaylistRepository interface {
	ResourceRepository
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(pls *Playlist) error
	Get(id string) (*Playlist, error)
	GetWithTracks(id string, refreshSmartPlaylist, includeMissing bool) (*Playlist, error)
	GetAll(options ...QueryOptions) (Playlists, error)
	FindByPath(path string) (*Playlist, error)
	Delete(id string) error
	Tracks(playlistId string, refreshSmartPlaylist bool) PlaylistTrackRepository
}

type PlaylistTrack struct {
	ID          string `json:"id"`
	MediaFileID string `json:"mediaFileId"`
	PlaylistID  string `json:"playlistId"`
	MediaFile
}

type PlaylistTracks []PlaylistTrack

func (plt PlaylistTracks) MediaFiles() MediaFiles {
	mfs := make(MediaFiles, len(plt))
	for i, t := range plt {
		mfs[i] = t.MediaFile
	}
	return mfs
}

type PlaylistTrackRepository interface {
	ResourceRepository
	GetAll(options ...QueryOptions) (PlaylistTracks, error)
	GetAlbumIDs(options ...QueryOptions) ([]string, error)
	Add(mediaFileIds []string) (int, error)
	AddAlbums(albumIds []string) (int, error)
	AddArtists(artistIds []string) (int, error)
	AddDiscs(discs []DiscID) (int, error)
	Delete(id ...string) error
	DeleteAll() error
	Reorder(pos int, newPos int) error
}
