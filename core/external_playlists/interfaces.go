package external_playlists

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
)

type PlaylistSourceInfo struct {
	Name  string   `json:"name"`
	Types []string `json:"types"`
}

type ExternalPlaylist struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	ID          string          `json:"id"`
	Url         string          `json:"url,omitempty"`
	Creator     string          `json:"creator"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	Existing    bool            `json:"existing"`
	Syncable    bool            `json:"syncable"`
	Tracks      []ExternalTrack `json:"-"`
}

type ExternalTrack struct {
	Title  string
	Artist string
	ID     string
}

type ExternalPlaylists struct {
	Total int
	Lists []ExternalPlaylist
}

type PlaylistAgent interface {
	GetPlaylistTypes() []string
	GetPlaylists(ctx context.Context, offset, count int, userId, playlistType string) (*ExternalPlaylists, error)
	ImportPlaylist(ctx context.Context, update bool, sync bool, userId, id, name string) error
	IsAuthorized(ctx context.Context, userId string) bool
	SyncPlaylist(ctx context.Context, tx model.DataStore, pls *model.Playlist) error
	SyncRecommended(ctx context.Context, userId string) error
}

type Constructor func(ds model.DataStore) PlaylistAgent
