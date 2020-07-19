package scanner

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/model/request"
	"github.com/deluan/navidrome/utils"
)

type playlistSync struct {
	ds model.DataStore
}

func newPlaylistSync(ds model.DataStore) *playlistSync {
	return &playlistSync{ds: ds}
}

func (s *playlistSync) processPlaylists(ctx context.Context, dir string) int {
	count := 0
	files, err := ioutil.ReadDir(dir)
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
	file, err := os.Open(playlistPath)
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

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		path := scanner.Text()
		// Skip extended info
		if strings.HasPrefix(path, "#") {
			continue
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		mf, err := s.ds.MediaFile(ctx).FindByPath(path)
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
	} else {
		log.Info(ctx, "Adding synced playlist", "playlist", newPls.Name, "path", newPls.Path, "owner", owner)
		newPls.Owner = owner
	}
	return s.ds.Playlist(ctx).Put(newPls)
}
