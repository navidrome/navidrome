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
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type phasePlaylists struct {
	ctx       context.Context
	scanState *scanState
	ds        model.DataStore
	pls       playlists.Playlists
	cw        artwork.CacheWarmer
	refreshed atomic.Uint32
	// pendingImport is true when a previous scan deferred playlist import (no admin
	// existed). In that case this phase imports all playlist folders and clears the
	// pending flag on completion.
	pendingImport bool
}

func createPhasePlaylists(ctx context.Context, scanState *scanState, ds model.DataStore, pls playlists.Playlists, cw artwork.CacheWarmer) *phasePlaylists {
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
	return ppl.NewProducer(p.produce, ppl.Name("load folders with playlists from db"))
}

func (p *phasePlaylists) produce(put func(entry *model.Folder)) error {
	if !conf.Server.AutoImportPlaylists {
		log.Info(p.ctx, "Playlists will not be imported, AutoImportPlaylists is set to false")
		return nil
	}

	// Resolve the admin user at phase time, not from the scan-start context: a
	// long scan may have begun before any admin existed, but one may have been
	// created since. Playlists are owned by the first admin.
	admin, err := p.ds.User(p.ctx).FindFirstAdmin()
	if err != nil || admin == nil || admin.ID == "" {
		// Still no admin: defer playlist import. Record a pending flag so the next
		// scan that runs with an admin imports them, regardless of folder timestamps.
		_ = p.ds.Property(p.ctx).Put(consts.PlaylistsImportPendingFlagKey, "1")
		log.Warn(p.ctx, "Playlists will not be imported, as there are no admin users yet. "+
			"They will be imported automatically once an admin user is created.")
		return nil
	}
	// Ensure downstream import uses the resolved admin as the playlist owner.
	p.ctx = request.WithUser(p.ctx, *admin)

	// If a previous scan deferred playlist import, recover ALL playlist folders
	// (ignoring the per-folder timestamp gate); otherwise only the touched ones.
	p.pendingImport = p.importPending()
	loadFolders := p.ds.Folder(p.ctx).GetTouchedWithPlaylists
	if p.pendingImport {
		loadFolders = p.ds.Folder(p.ctx).GetAllWithPlaylists
	}

	count := 0
	cursor, err := loadFolders()
	if err != nil {
		return fmt.Errorf("loading folders with playlists: %w", err)
	}
	log.Debug(p.ctx, "Scanner: Checking playlists that may need refresh", "pendingImport", p.pendingImport)
	for folder, err := range cursor {
		if err != nil {
			return fmt.Errorf("loading folder with playlists: %w", err)
		}
		count++
		put(&folder)
	}
	if count == 0 {
		log.Debug(p.ctx, "Scanner: No playlists need refreshing")
	} else {
		log.Debug(p.ctx, "Scanner: Found folders with playlists that may need refreshing", "count", count)
	}

	return nil
}

// importPending reports whether a previous scan deferred playlist import because
// no admin user existed yet.
func (p *phasePlaylists) importPending() bool {
	v, _ := p.ds.Property(p.ctx).DefaultGet(consts.PlaylistsImportPendingFlagKey, "0")
	return v == "1"
}

func (p *phasePlaylists) stages() []ppl.Stage[*model.Folder] {
	return []ppl.Stage[*model.Folder]{
		ppl.NewStage(p.processPlaylistsInFolder, ppl.Name("process playlists in folder"), ppl.Concurrency(3)),
	}
}

func (p *phasePlaylists) processPlaylistsInFolder(folder *model.Folder) (*model.Folder, error) {
	files, err := os.ReadDir(folder.AbsolutePath())
	if err != nil {
		log.Error(p.ctx, "Scanner: Error reading files", "folder", folder, err)
		p.scanState.sendWarning(err.Error())
		return folder, nil
	}
	for _, f := range files {
		started := time.Now()
		if strings.HasPrefix(f.Name(), ".") {
			continue
		}
		if !model.IsValidPlaylist(f.Name()) {
			continue
		}
		// BFR: Check if playlist needs to be refreshed (timestamp, sync flag, etc)
		pls, err := p.pls.ImportFromFolder(p.ctx, folder, f.Name())
		if err != nil {
			continue
		}
		if pls.IsSmartPlaylist() {
			log.Debug("Scanner: Imported smart playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "elapsed", time.Since(started))
		} else {
			log.Debug("Scanner: Imported playlist", "name", pls.Name, "lastUpdated", pls.UpdatedAt, "path", pls.Path, "numTracks", len(pls.Tracks), "elapsed", time.Since(started))
		}
		p.cw.PreCache(pls.CoverArtID())
		p.refreshed.Add(1)
	}
	return folder, nil
}

func (p *phasePlaylists) finalize(err error) error {
	refreshed := p.refreshed.Load()
	logF := log.Info
	if refreshed == 0 {
		logF = log.Debug
	} else {
		p.scanState.changesDetected.Store(true)
	}
	// If this phase ran to recover playlists deferred by a previous scan, clear the
	// pending flag now that they've been imported.
	if p.pendingImport && err == nil {
		if derr := p.ds.Property(p.ctx).Delete(consts.PlaylistsImportPendingFlagKey); derr != nil {
			log.Warn(p.ctx, "Scanner: Could not clear pending playlist-import flag", derr)
		}
	}
	logF(p.ctx, "Scanner: Finished refreshing playlists", "refreshed", refreshed, err)
	return err
}

var _ phase[*model.Folder] = (*phasePlaylists)(nil)
