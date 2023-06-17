package scanner

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-zglob"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type playlistImporter struct {
	ds          model.DataStore
	pls         core.Playlists
	cacheWarmer artwork.CacheWarmer
	rootFolder  string
}

func newPlaylistImporter(ds model.DataStore, playlists core.Playlists, cacheWarmer artwork.CacheWarmer, rootFolder string) *playlistImporter {
	return &playlistImporter{ds: ds, pls: playlists, cacheWarmer: cacheWarmer, rootFolder: rootFolder}
}

func (s *playlistImporter) processPlaylists(ctx context.Context, dir string) int64 {
	if !s.inPlaylistsPath(dir) {
		return 0
	}
	var count int64
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Error(ctx, "Error reading files", "dir", dir, err)
		return count
	}
	for _, f := range files {
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if !model.IsValidPlaylist(f.Name()) {
			continue
		}
		pls, err := s.pls.ImportFile(ctx, dir, f.Name())
		if err != nil {
			continue
		}
		if pls.IsSmartPlaylist() {
			log.Debug("Imported smart playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", pls.SongCount)
		} else {
			log.Debug("Imported playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", pls.SongCount)
		}
		s.cacheWarmer.PreCache(pls.CoverArtID())
		count++
	}
	return count
}

func (s *playlistImporter) inPlaylistsPath(dir string) bool {
	rel, _ := filepath.Rel(s.rootFolder, dir)
	for _, path := range strings.Split(conf.Server.PlaylistsPath, string(filepath.ListSeparator)) {
		if match, _ := zglob.Match(path, rel); match {
			return true
		}
	}
	return false
}
