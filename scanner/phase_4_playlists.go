package scanner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type phasePlaylists struct {
	ctx       context.Context
	scanState *scanState
	ds        model.DataStore
	pls       core.Playlists
	cw        artwork.CacheWarmer
	refreshed atomic.Uint32
}

func createPhasePlaylists(ctx context.Context, scanState *scanState, ds model.DataStore, pls core.Playlists, cw artwork.CacheWarmer) *phasePlaylists {
	return &phasePlaylists{
		ctx:       ctx,
		scanState: scanState,
		ds:        ds,
		pls:       pls,
		cw:        cw,
	}
}

func (p *phasePlaylists) description() string {
	return "Import/update playlists"
}

func (p *phasePlaylists) producer() ppl.Producer[*model.Folder] {
	return ppl.NewProducer(func(put func(entry *model.Folder)) error {
		u, _ := request.UserFrom(p.ctx)
		if !conf.Server.AutoImportPlaylists || !u.IsAdmin {
			log.Warn(p.ctx, "Playlists will not be imported, as there are no admin users yet, "+
				"Please create an admin user first, and then update the playlists for them to be imported")
			return nil
		}

		count := 0
		cursor, err := p.ds.Folder(p.ctx).GetTouchedWithPlaylists()
		if err != nil {
			return fmt.Errorf("error loading touched folders: %w", err)
		}
		log.Debug(p.ctx, "Scanner: Checking playlists that may need refresh")
		for folder, err := range cursor {
			if err != nil {
				return fmt.Errorf("error loading touched folder: %w", err)
			}
			count++
			put(&folder)
		}
		if count == 0 {
			log.Debug(p.ctx, "Scanner: No folders with playlists needs refreshing")
		} else {
			log.Debug(p.ctx, "Scanner: Found folders with playlists that may need refreshing", "count", count)
		}

		return nil
	}, ppl.Name("load folders with playlists from db"))
}

func (p *phasePlaylists) stages() []ppl.Stage[*model.Folder] {
	return []ppl.Stage[*model.Folder]{
		ppl.NewStage(p.processPlaylistsInFolder, ppl.Name("process playlists in folder"), ppl.Concurrency(3)),
	}
}

func (p *phasePlaylists) processPlaylistsInFolder(folderPath *model.Folder) (*model.Folder, error) {
	// BFR PlaylistsPath
	//if !s.inPlaylistsPath(dir) {
	//	return 0
	//}
	files, err := os.ReadDir(folderPath.AbsolutePath())
	if err != nil {
		log.Error(p.ctx, "Error reading files", "folderPath", folderPath, err)
		p.scanState.progress <- &ProgressInfo{Err: err}
	}
	for _, f := range files {
		started := time.Now()
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if !model.IsValidPlaylist(f.Name()) {
			continue
		}
		pls, err := p.pls.ImportFile(p.ctx, folderPath, f.Name())
		if err != nil {
			continue
		}
		if pls.IsSmartPlaylist() {
			log.Debug("Imported smart playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "elapsed", time.Since(started))
		} else {
			log.Debug("Imported playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", len(pls.Tracks), "elapsed", time.Since(started))
		}
		p.cw.PreCache(pls.CoverArtID())
		p.refreshed.Add(1)
	}
	return folderPath, nil
}

func (p *phasePlaylists) finalize(err error) error {
	refreshed := p.refreshed.Load()
	logF := log.Info
	if refreshed == 0 {
		logF = log.Debug
	} else {
		p.scanState.changesDetected.Store(true)
	}
	logF(p.ctx, "Scanner: Finished refreshing playlists", "refreshed", refreshed, err)
	return err
}

//func (s *playlistImporter) inPlaylistsPath(dir string) bool {
//	rel, _ := filepath.Rel(s.rootFolder, dir)
//	for _, path := range strings.Split(conf.Server.PlaylistsPath, string(filepath.ListSeparator)) {
//		if match, _ := zglob.Match(path, rel); match {
//			return true
//		}
//	}
//	return false
//}

var _ phase[*model.Folder] = (*phasePlaylists)(nil)
