package core

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
)

type Playlists interface {
	ImportFile(ctx context.Context, dir string, fname string) (*model.Playlist, error)
}

type playlists struct {
	ds model.DataStore
}

func NewPlaylists(ds model.DataStore) Playlists {
	return &playlists{ds: ds}
}

func IsPlaylist(filePath string) bool {
	extension := strings.ToLower(filepath.Ext(filePath))
	return extension == ".m3u" || extension == ".m3u8" || extension == ".nsp"
}

func (s *playlists) ImportFile(ctx context.Context, dir string, fname string) (*model.Playlist, error) {
	pls, err := s.parsePlaylist(ctx, fname, dir)
	if err != nil {
		log.Error(ctx, "Error parsing playlist", "playlist", fname, err)
		return nil, err
	}
	log.Debug("Found playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", len(pls.Tracks))
	err = s.updatePlaylist(ctx, pls)
	if err != nil {
		log.Error(ctx, "Error updating playlist", "playlist", fname, err)
	}
	return pls, err
}

func (s *playlists) parsePlaylist(ctx context.Context, playlistFile string, baseDir string) (*model.Playlist, error) {
	pls, err := s.newSyncedPlaylist(baseDir, playlistFile)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(pls.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	extension := strings.ToLower(filepath.Ext(playlistFile))
	switch extension {
	case ".nsp":
		return s.parseNSP(ctx, pls, file)
	default:
		return s.parseM3U(ctx, pls, baseDir, file)
	}
}

func (s *playlists) newSyncedPlaylist(baseDir string, playlistFile string) (*model.Playlist, error) {
	playlistPath := filepath.Join(baseDir, playlistFile)
	info, err := os.Stat(playlistPath)
	if err != nil {
		return nil, err
	}

	var extension = filepath.Ext(playlistFile)
	var name = playlistFile[0 : len(playlistFile)-len(extension)]

	pls := &model.Playlist{
		Name:      name,
		Comment:   fmt.Sprintf("Auto-imported from '%s'", playlistFile),
		Public:    false,
		Path:      playlistPath,
		Sync:      true,
		UpdatedAt: info.ModTime(),
	}
	return pls, nil
}

func (s *playlists) parseNSP(ctx context.Context, pls *model.Playlist, file io.Reader) (*model.Playlist, error) {
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	pls.Rules = &criteria.Criteria{}
	err = json.Unmarshal(content, pls.Rules)
	if err != nil {
		log.Error(ctx, "Error parsing SmartPlaylist", "playlist", pls.Name, err)
		return nil, err
	}
	return pls, nil
}

func (s *playlists) parseM3U(ctx context.Context, pls *model.Playlist, baseDir string, file io.Reader) (*model.Playlist, error) {
	mediaFileRepository := s.ds.MediaFile(ctx)
	scanner := bufio.NewScanner(file)
	scanner.Split(scanLines)
	var mfs model.MediaFiles
	for scanner.Scan() {
		path := scanner.Text()
		// Skip extended info
		if strings.HasPrefix(path, "#") {
			continue
		}
		if strings.HasPrefix(path, "file://") {
			path = strings.TrimPrefix(path, "file://")
			path, _ = url.QueryUnescape(path)
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		mf, err := mediaFileRepository.FindByPath(path)
		if err != nil {
			log.Warn(ctx, "Path in playlist not found", "playlist", pls.Name, "path", path, err)
			continue
		}
		mfs = append(mfs, *mf)
	}
	pls.Tracks = nil
	pls.AddMediaFiles(mfs)

	return pls, scanner.Err()
}

func (s *playlists) updatePlaylist(ctx context.Context, newPls *model.Playlist) error {
	owner, _ := request.UsernameFrom(ctx)

	pls, err := s.ds.Playlist(ctx).FindByPath(newPls.Path)
	if err != nil && err != model.ErrNotFound {
		return err
	}
	if err == nil && !pls.Sync {
		log.Debug(ctx, "Playlist already imported and not synced", "playlist", pls.Name, "path", pls.Path)
		return nil
	}

	if err == nil {
		log.Info(ctx, "Updating synced playlist", "playlist", pls.Name, "path", newPls.Path)
		newPls.ID = pls.ID
		newPls.Name = pls.Name
		newPls.Comment = pls.Comment
		newPls.Owner = pls.Owner
		newPls.Public = pls.Public
	} else {
		log.Info(ctx, "Adding synced playlist", "playlist", newPls.Name, "path", newPls.Path, "owner", owner)
		newPls.Owner = owner
	}
	return s.ds.Playlist(ctx).Put(newPls)
}

// From https://stackoverflow.com/a/41433698
func scanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		if data[i] == '\n' {
			// We have a line terminated by single newline.
			return i + 1, data[0:i], nil
		}
		advance = i + 1
		if len(data) > i+1 && data[i+1] == '\n' {
			advance += 1
		}
		return advance, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
