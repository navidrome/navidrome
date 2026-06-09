package playlists

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/ioutils"
	"golang.org/x/text/unicode/norm"
)

func (s *playlists) ImportFile(ctx context.Context, absolutePath string, sync bool) (*model.Playlist, error) {
	absPath, err := filepath.Abs(absolutePath)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute path: %w", err)
	}

	dir := filepath.Dir(absPath)
	filename := filepath.Base(absPath)

	folder, err := s.resolveFolder(ctx, dir)
	if err != nil && !errors.Is(err, errNotInLibrary) {
		return nil, err
	}
	if err == nil {
		pls, err := s.importFromFolder(ctx, folder, filename, sync)
		if err != nil {
			return nil, err
		}
		if pls.ID != "" && pls.Sync != sync {
			pls.Sync = sync
			if putErr := s.ds.Playlist(ctx).Put(pls); putErr != nil {
				return nil, putErr
			}
		}
		return pls, nil
	}

	log.Debug(ctx, "Playlist file is outside all libraries, using path-based import", "path", absPath)
	pls, err := s.newSyncedPlaylist(dir, filename)
	if err != nil {
		return nil, fmt.Errorf("reading playlist file: %w", err)
	}
	pls.Sync = sync

	file, err := os.Open(absPath)
	if err != nil {
		return nil, fmt.Errorf("opening playlist file: %w", err)
	}
	defer file.Close()

	reader := ioutils.UTF8Reader(file)
	if err := s.parseM3U(ctx, pls, nil, reader); err != nil {
		return nil, err
	}
	if err := s.updatePlaylist(ctx, pls, sync); err != nil {
		return nil, err
	}
	return pls, nil
}

var errNotInLibrary = fmt.Errorf("path not in any library")

func (s *playlists) resolveFolder(ctx context.Context, dir string) (*model.Folder, error) {
	libs, err := s.ds.Library(ctx).GetAll()
	if err != nil {
		return nil, err
	}
	matcher := newLibraryMatcher(libs)
	lib, ok := matcher.findLibrary(dir)
	if !ok {
		return nil, fmt.Errorf("%w: %s", errNotInLibrary, dir)
	}

	folder, err := s.ds.Folder(ctx).GetByPath(lib, dir)
	if err != nil {
		return nil, fmt.Errorf("resolving folder for path %s: %w", dir, err)
	}
	folder.LibraryPath = lib.Path
	return folder, nil
}

func (s *playlists) ImportFromFolder(ctx context.Context, folder *model.Folder, filename string) (*model.Playlist, error) {
	return s.importFromFolder(ctx, folder, filename, false)
}

func (s *playlists) importFromFolder(ctx context.Context, folder *model.Folder, filename string, forceSync bool) (*model.Playlist, error) {
	pls, err := s.parsePlaylist(ctx, filename, folder)
	if err != nil {
		log.Error(ctx, "Error parsing playlist", "path", filepath.Join(folder.AbsolutePath(), filename), err)
		return nil, err
	}
	log.Debug(ctx, "Found playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", len(pls.Tracks))
	err = s.updatePlaylist(ctx, pls, forceSync)
	if err != nil {
		log.Error(ctx, "Error updating playlist", "path", filepath.Join(folder.AbsolutePath(), filename), err)
	}
	return pls, err
}

func (s *playlists) ImportM3U(ctx context.Context, reader io.Reader) (*model.Playlist, error) {
	owner, _ := request.UserFrom(ctx)
	pls := &model.Playlist{
		OwnerID: owner.ID,
		Public:  false,
		Sync:    false,
	}
	err := s.parseM3U(ctx, pls, nil, reader)
	if err != nil {
		log.Error(ctx, "Error parsing playlist", err)
		return nil, err
	}
	err = s.ds.Playlist(ctx).Put(pls)
	if err != nil {
		log.Error(ctx, "Error saving playlist", err)
		return nil, err
	}
	return pls, nil
}

func (s *playlists) parsePlaylist(ctx context.Context, playlistFile string, folder *model.Folder) (*model.Playlist, error) {
	pls, err := s.newSyncedPlaylist(folder.AbsolutePath(), playlistFile)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(pls.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := ioutils.UTF8Reader(file)
	extension := strings.ToLower(filepath.Ext(playlistFile))
	switch extension {
	case ".nsp":
		err = s.parseNSP(ctx, pls, reader)
	default:
		err = s.parseM3U(ctx, pls, folder, reader)
	}
	return pls, err
}

// findByPathNormalized looks up a playlist by path, trying both NFC and NFD Unicode
// normalization forms to handle cross-platform filesystem differences.
func (s *playlists) findByPathNormalized(ctx context.Context, path string) (*model.Playlist, error) {
	pls, err := s.ds.Playlist(ctx).FindByPath(path)
	if errors.Is(err, model.ErrNotFound) {
		altPath := norm.NFD.String(path)
		if altPath == path {
			altPath = norm.NFC.String(path)
		}
		if altPath != path {
			pls, err = s.ds.Playlist(ctx).FindByPath(altPath)
		}
	}
	return pls, err
}

func (s *playlists) updatePlaylist(ctx context.Context, newPls *model.Playlist, forceSync bool) error {
	owner, _ := request.UserFrom(ctx)

	pls, err := s.findByPathNormalized(ctx, newPls.Path)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}
	alreadyImportedAndNotSynced := err == nil && !pls.Sync && !forceSync
	if alreadyImportedAndNotSynced {
		log.Debug(ctx, "Playlist already imported and not synced", "playlist", pls.Name, "path", pls.Path)
		return nil
	}

	if err == nil {
		log.Info(ctx, "Updating synced playlist", "playlist", pls.Name, "path", newPls.Path)
		newPls.ID = pls.ID
		newPls.Name = pls.Name
		newPls.Comment = pls.Comment
		newPls.OwnerID = pls.OwnerID
		newPls.Public = pls.Public
		newPls.UploadedImage = pls.UploadedImage // Preserve manual upload
		newPls.EvaluatedAt = &time.Time{}
	} else {
		log.Info(ctx, "Adding synced playlist", "playlist", newPls.Name, "path", newPls.Path, "owner", owner.UserName)
		newPls.OwnerID = owner.ID
		// For NSP files, Public may already be set from the file; for M3U, use server default
		if !newPls.IsSmartPlaylist() {
			newPls.Public = conf.Server.DefaultPlaylistPublicVisibility
		}
	}
	return s.ds.Playlist(ctx).Put(newPls)
}
