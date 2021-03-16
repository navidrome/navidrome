package scanner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils"
)

type playlistSync struct {
	ds   model.DataStore
	fsys fs.FS
}

func newPlaylistSync(ds model.DataStore, fsys fs.FS) *playlistSync {
	return &playlistSync{ds: ds, fsys: fsys}
}

func (s *playlistSync) processPlaylists(ctx context.Context, dir string) int64 {
	var count int64
	files, err := fs.ReadDir(s.fsys, dir)
	if err != nil {
		log.Error(ctx, "Error reading files", "dir", dir, err)
		return count
	}
	for _, f := range files {
		if !utils.IsPlaylist(f.Name()) {
			continue
		}
		pls, err := s.parsePlaylist(ctx, f.Name(), dir)
		if err != nil {
			log.Error(ctx, "Error parsing playlist", "playlist", f.Name(), err)
			continue
		}
		log.Debug("Found playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", len(pls.Tracks))
		err = s.updatePlaylist(ctx, pls)
		if err != nil {
			log.Error(ctx, "Error updating playlist", "playlist", f.Name(), err)
		}
		count++
	}
	return count
}

func (s *playlistSync) parsePlaylist(ctx context.Context, playlistFile string, baseDir string) (*model.Playlist, error) {
	playlistPath := filepath.Join(baseDir, playlistFile)
	file, err := s.fsys.Open(playlistPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
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

	mediaFileRepository := s.ds.MediaFile(ctx)
	scanner := bufio.NewScanner(file)
	scanner.Split(scanLines)
	for scanner.Scan() {
		path := scanner.Text()
		// Skip extended info
		if strings.HasPrefix(path, "#") {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		mf, err := mediaFileRepository.FindByPath(path)
		if err != nil {
			log.Warn(ctx, "Path in playlist not found", "playlist", playlistFile, "path", path, err)
			continue
		}
		pls.Tracks = append(pls.Tracks, *mf)
	}

	return pls, scanner.Err()
}

func (s *playlistSync) updatePlaylist(ctx context.Context, newPls *model.Playlist) error {
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
