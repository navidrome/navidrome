package playlists

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type Playlists interface {
	// Reads
	GetAll(ctx context.Context, options ...model.QueryOptions) (model.Playlists, error)
	Get(ctx context.Context, id string) (*model.Playlist, error)
	GetWithTracks(ctx context.Context, id string) (*model.Playlist, error)
	GetPlaylists(ctx context.Context, mediaFileId string) (model.Playlists, error)

	// Mutations
	Create(ctx context.Context, playlistId string, name string, ids []string) (string, error)
	Delete(ctx context.Context, id string) error
	Update(ctx context.Context, playlistID string, name *string, comment *string, public *bool, idsToAdd []string, idxToRemove []int) error

	// Track management
	AddTracks(ctx context.Context, playlistID string, ids []string) (int, error)
	AddAlbums(ctx context.Context, playlistID string, albumIds []string) (int, error)
	AddArtists(ctx context.Context, playlistID string, artistIds []string) (int, error)
	AddDiscs(ctx context.Context, playlistID string, discs []model.DiscID) (int, error)
	RemoveTracks(ctx context.Context, playlistID string, trackIds []string) error
	ReorderTrack(ctx context.Context, playlistID string, pos int, newPos int) error

	// Cover art
	SetImage(ctx context.Context, playlistID string, reader io.Reader, ext string) error
	RemoveImage(ctx context.Context, playlistID string) error

	// Import
	ImportFile(ctx context.Context, folder *model.Folder, filename string) (*model.Playlist, error)
	ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error)

	// REST adapters (follows Share/Library pattern)
	NewRepository(ctx context.Context) rest.Repository
	TracksRepository(ctx context.Context, playlistId string, refreshSmartPlaylist bool) rest.Repository
}

type playlists struct {
	ds model.DataStore
}

func NewPlaylists(ds model.DataStore) Playlists {
	return &playlists{ds: ds}
}

func InPath(folder model.Folder) bool {
	if conf.Server.PlaylistsPath == "" {
		return true
	}
	rel, _ := filepath.Rel(folder.LibraryPath, folder.AbsolutePath())
	for path := range strings.SplitSeq(conf.Server.PlaylistsPath, string(filepath.ListSeparator)) {
		if match, _ := doublestar.Match(path, rel); match {
			return true
		}
	}
	return false
}

// --- Read operations ---

func (s *playlists) GetAll(ctx context.Context, options ...model.QueryOptions) (model.Playlists, error) {
	return s.ds.Playlist(ctx).GetAll(options...)
}

func (s *playlists) Get(ctx context.Context, id string) (*model.Playlist, error) {
	return s.ds.Playlist(ctx).Get(id)
}

func (s *playlists) GetWithTracks(ctx context.Context, id string) (*model.Playlist, error) {
	return s.ds.Playlist(ctx).GetWithTracks(id, true, false)
}

func (s *playlists) GetPlaylists(ctx context.Context, mediaFileId string) (model.Playlists, error) {
	return s.ds.Playlist(ctx).GetPlaylists(mediaFileId)
}

// --- Mutation operations ---

// Create creates a new playlist (when name is provided) or replaces tracks on an existing
// playlist (when playlistId is provided). This matches the Subsonic createPlaylist semantics.
func (s *playlists) Create(ctx context.Context, playlistId string, name string, ids []string) (string, error) {
	usr, _ := request.UserFrom(ctx)
	err := s.ds.WithTxImmediate(func(tx model.DataStore) error {
		var pls *model.Playlist
		var err error

		if playlistId != "" {
			pls, err = tx.Playlist(ctx).Get(playlistId)
			if err != nil {
				return err
			}
			if pls.IsSmartPlaylist() {
				return model.ErrNotAuthorized
			}
			if !usr.IsAdmin && pls.OwnerID != usr.ID {
				return model.ErrNotAuthorized
			}
		} else {
			pls = &model.Playlist{Name: name}
			pls.OwnerID = usr.ID
		}
		pls.Tracks = nil
		pls.AddMediaFilesByID(ids)

		err = tx.Playlist(ctx).Put(pls)
		playlistId = pls.ID
		return err
	})
	return playlistId, err
}

func (s *playlists) Delete(ctx context.Context, id string) error {
	pls, err := s.checkWritable(ctx, id)
	if err != nil {
		return err
	}

	// Clean up custom cover image file if one exists
	if path := pls.ArtworkPath(); path != "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Warn(ctx, "Failed to remove playlist image on delete", "path", path, err)
		}
	}

	return s.ds.Playlist(ctx).Delete(id)
}

func (s *playlists) Update(ctx context.Context, playlistID string,
	name *string, comment *string, public *bool,
	idsToAdd []string, idxToRemove []int) error {
	var pls *model.Playlist
	var err error
	hasTrackChanges := len(idsToAdd) > 0 || len(idxToRemove) > 0
	if hasTrackChanges {
		pls, err = s.checkTracksEditable(ctx, playlistID)
	} else {
		pls, err = s.checkWritable(ctx, playlistID)
	}
	if err != nil {
		return err
	}
	return s.ds.WithTxImmediate(func(tx model.DataStore) error {
		repo := tx.Playlist(ctx)

		if len(idxToRemove) > 0 {
			tracksRepo := repo.Tracks(playlistID, false)
			// Convert 0-based indices to 1-based position IDs and delete them directly,
			// avoiding the need to load all tracks into memory.
			positions := make([]string, len(idxToRemove))
			for i, idx := range idxToRemove {
				positions[i] = strconv.Itoa(idx + 1)
			}
			if err := tracksRepo.Delete(positions...); err != nil {
				return err
			}
			if len(idsToAdd) > 0 {
				if _, err := tracksRepo.Add(idsToAdd); err != nil {
					return err
				}
			}
			return s.updateMetadata(ctx, tx, pls, name, comment, public)
		}

		if len(idsToAdd) > 0 {
			if _, err := repo.Tracks(playlistID, false).Add(idsToAdd); err != nil {
				return err
			}
		}
		if name == nil && comment == nil && public == nil {
			return nil
		}
		// Reuse the playlist from checkWritable (no tracks loaded, so Put only refreshes counters)
		return s.updateMetadata(ctx, tx, pls, name, comment, public)
	})
}

// --- Permission helpers ---

// checkWritable fetches the playlist and verifies the current user can modify it.
func (s *playlists) checkWritable(ctx context.Context, id string) (*model.Playlist, error) {
	pls, err := s.ds.Playlist(ctx).Get(id)
	if err != nil {
		return nil, err
	}
	usr, _ := request.UserFrom(ctx)
	if !usr.IsAdmin && pls.OwnerID != usr.ID {
		return nil, model.ErrNotAuthorized
	}
	return pls, nil
}

// checkTracksEditable verifies the user can modify tracks (ownership + not smart playlist).
func (s *playlists) checkTracksEditable(ctx context.Context, playlistID string) (*model.Playlist, error) {
	pls, err := s.checkWritable(ctx, playlistID)
	if err != nil {
		return nil, err
	}
	if pls.IsSmartPlaylist() {
		return nil, model.ErrNotAuthorized
	}
	return pls, nil
}

// updateMetadata applies optional metadata changes to a playlist and persists it.
// Accepts a DataStore parameter so it can be used inside transactions.
// The caller is responsible for permission checks.
func (s *playlists) updateMetadata(ctx context.Context, ds model.DataStore, pls *model.Playlist, name *string, comment *string, public *bool) error {
	if name != nil {
		pls.Name = *name
	}
	if comment != nil {
		pls.Comment = *comment
	}
	if public != nil {
		pls.Public = *public
	}
	return ds.Playlist(ctx).Put(pls)
}

// --- Track management operations ---

func (s *playlists) AddTracks(ctx context.Context, playlistID string, ids []string) (int, error) {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return 0, err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).Add(ids)
}

func (s *playlists) AddAlbums(ctx context.Context, playlistID string, albumIds []string) (int, error) {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return 0, err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).AddAlbums(albumIds)
}

func (s *playlists) AddArtists(ctx context.Context, playlistID string, artistIds []string) (int, error) {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return 0, err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).AddArtists(artistIds)
}

func (s *playlists) AddDiscs(ctx context.Context, playlistID string, discs []model.DiscID) (int, error) {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return 0, err
	}
	return s.ds.Playlist(ctx).Tracks(playlistID, false).AddDiscs(discs)
}

func (s *playlists) RemoveTracks(ctx context.Context, playlistID string, trackIds []string) error {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return err
	}
	return s.ds.WithTx(func(tx model.DataStore) error {
		return tx.Playlist(ctx).Tracks(playlistID, false).Delete(trackIds...)
	})
}

func (s *playlists) ReorderTrack(ctx context.Context, playlistID string, pos int, newPos int) error {
	if _, err := s.checkTracksEditable(ctx, playlistID); err != nil {
		return err
	}
	return s.ds.WithTx(func(tx model.DataStore) error {
		return tx.Playlist(ctx).Tracks(playlistID, false).Reorder(pos, newPos)
	})
}

// --- Cover art operations ---

func (s *playlists) SetImage(ctx context.Context, playlistID string, reader io.Reader, ext string) error {
	pls, err := s.checkWritable(ctx, playlistID)
	if err != nil {
		return err
	}

	filename := playlistID + ext
	oldPath := pls.ArtworkPath()
	pls.ImageFile = filename
	absPath := pls.ArtworkPath()

	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("creating playlist images directory: %w", err)
	}

	// Remove old image if it exists
	if oldPath != "" {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			log.Warn(ctx, "Failed to remove old playlist image", "path", oldPath, err)
		}
	}

	// Save new image
	f, err := os.Create(absPath)
	if err != nil {
		return fmt.Errorf("creating playlist image file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("writing playlist image file: %w", err)
	}

	return s.ds.Playlist(ctx).Put(pls)
}

func (s *playlists) RemoveImage(ctx context.Context, playlistID string) error {
	pls, err := s.checkWritable(ctx, playlistID)
	if err != nil {
		return err
	}

	if path := pls.ArtworkPath(); path != "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			log.Warn(ctx, "Failed to remove playlist image", "path", path, err)
		}
	}

	pls.ImageFile = ""
	return s.ds.Playlist(ctx).Put(pls)
}
