package scanner

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Masterminds/squirrel"
	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/mime"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type phaseArtwork struct {
	ctx       context.Context
	scanState *scanState
	ds        model.DataStore
	processed atomic.Uint32
}

func createPhaseArtwork(ctx context.Context, scanState *scanState, ds model.DataStore) *phaseArtwork {
	return &phaseArtwork{
		ctx:       ctx,
		scanState: scanState,
		ds:        ds,
	}
}

func (p *phaseArtwork) description() string {
	return "Scan custom artwork"
}

func (p *phaseArtwork) producer() ppl.Producer[string] {
	return ppl.NewProducer(p.produce, ppl.Name("scan artwork folder"))
}

func (p *phaseArtwork) produce(put func(entry string)) error {
	if conf.Server.ArtworkFolder == "" {
		log.Debug(p.ctx, "Scanner: ArtworkFolder not configured, skipping artwork scan")
		return nil
	}

	playlistArtworkPath := filepath.Join(conf.Server.ArtworkFolder, "playlist")
	if _, err := os.Stat(playlistArtworkPath); os.IsNotExist(err) {
		log.Debug(p.ctx, "Scanner: Playlist artwork folder does not exist", "path", playlistArtworkPath)
		return nil
	}

	files, err := os.ReadDir(playlistArtworkPath)
	if err != nil {
		log.Error(p.ctx, "Scanner: Error reading playlist artwork folder", "path", playlistArtworkPath, err)
		return err
	}

	count := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if !isValidImageFile(file.Name()) {
			continue
		}
		put(filepath.Join(playlistArtworkPath, file.Name()))
		count++
	}

	if count > 0 {
		log.Debug(p.ctx, "Scanner: Found custom artwork files", "count", count)
	}

	return nil
}

func (p *phaseArtwork) stages() []ppl.Stage[string] {
	return []ppl.Stage[string]{
		ppl.NewStage(p.processArtworkFile, ppl.Name("process artwork file"), ppl.Concurrency(3)),
	}
}

func (p *phaseArtwork) processArtworkFile(artworkPath string) (string, error) {
	started := time.Now()

	// Extract playlist ID/name from filename
	filename := filepath.Base(artworkPath)
	nameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Calculate file hash
	hash, err := p.calculateFileHash(artworkPath)
	if err != nil {
		log.Warn(p.ctx, "Scanner: Error calculating artwork hash", "file", artworkPath, err)
		return artworkPath, nil
	}

	// Try to find playlist by ID first, then by name
	playlist, err := p.findPlaylistByIdOrName(nameWithoutExt)
	if err != nil {
		log.Debug(p.ctx, "Scanner: No playlist found for artwork", "file", filename, "name", nameWithoutExt)
		return artworkPath, nil //nolint:nilerr // Intentionally ignoring error when no playlist found
	}

	// Update playlist's custom artwork hash
	err = p.updatePlaylistArtworkHash(playlist.ID, hash)
	if err != nil {
		log.Error(p.ctx, "Scanner: Error updating playlist artwork hash", "playlistId", playlist.ID, "file", filename, err)
		return artworkPath, err
	}

	log.Debug(p.ctx, "Scanner: Updated playlist artwork hash", "playlistId", playlist.ID, "name", playlist.Name, "file", filename, "hash", hash, "elapsed", time.Since(started))
	p.processed.Add(1)

	return artworkPath, nil
}

func (p *phaseArtwork) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Get file info for modification time
	info, err := file.Stat()
	if err != nil {
		return "", err
	}

	// Create hash from file content + modification time
	hash := md5.New()

	// Include modification time in hash
	_, err = hash.Write([]byte(info.ModTime().Format(time.RFC3339Nano)))
	if err != nil {
		return "", err
	}

	// Include file content in hash
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil))[:8], nil // Use first 8 chars for brevity
}

func (p *phaseArtwork) findPlaylistByIdOrName(name string) (*model.Playlist, error) {
	// Try to find by ID first
	playlist, err := p.ds.Playlist(p.ctx).Get(name)
	if err == nil {
		return playlist, nil
	}

	// If not found by ID, try to find by name
	playlists, err := p.ds.Playlist(p.ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"playlist.name": name},
	})
	if err != nil {
		return nil, err
	}

	if len(playlists) == 0 {
		return nil, model.ErrNotFound
	}

	// Return the first match
	return &playlists[0], nil
}

func (p *phaseArtwork) updatePlaylistArtworkHash(playlistID, hash string) error {
	return p.ds.WithTx(func(tx model.DataStore) error {
		playlist, err := tx.Playlist(p.ctx).Get(playlistID)
		if err != nil {
			return err
		}

		// Only update if hash has changed
		if playlist.CustomArtworkHash == hash {
			return nil
		}

		playlist.CustomArtworkHash = hash
		return tx.Playlist(p.ctx).Put(playlist)
	}, "scanner: update playlist artwork hash")
}

func (p *phaseArtwork) finalize(err error) error {
	processed := p.processed.Load()

	// Clear artwork hashes for playlists that no longer have custom artwork
	if processed > 0 {
		p.clearOrphanedArtworkHashes()
	}

	logF := log.Info
	if processed == 0 {
		logF = log.Debug
	} else {
		p.scanState.changesDetected.Store(true)
	}
	logF(p.ctx, "Scanner: Finished scanning custom artwork", "processed", processed, err)
	return err
}

func (p *phaseArtwork) clearOrphanedArtworkHashes() {
	if conf.Server.ArtworkFolder == "" {
		return
	}

	playlistArtworkPath := filepath.Join(conf.Server.ArtworkFolder, "playlist")
	if _, err := os.Stat(playlistArtworkPath); os.IsNotExist(err) {
		// If artwork folder doesn't exist, clear all hashes
		p.clearAllCustomArtworkHashes()
		return
	}

	// Get all existing artwork files
	files, err := os.ReadDir(playlistArtworkPath)
	if err != nil {
		log.Warn(p.ctx, "Scanner: Error reading playlist artwork folder for cleanup", "path", playlistArtworkPath, err)
		return
	}

	existingFiles := make(map[string]bool)
	for _, file := range files {
		if file.IsDir() || !isValidImageFile(file.Name()) {
			continue
		}
		nameWithoutExt := strings.TrimSuffix(file.Name(), filepath.Ext(file.Name()))
		existingFiles[nameWithoutExt] = true
	}

	// Get all playlists with custom artwork hashes
	playlists, err := p.ds.Playlist(p.ctx).GetAll(model.QueryOptions{
		Filters: squirrel.NotEq{"custom_artwork_hash": ""},
	})
	if err != nil {
		log.Warn(p.ctx, "Scanner: Error getting playlists with custom artwork", err)
		return
	}

	// Clear hashes for playlists without corresponding files
	for _, playlist := range playlists {
		hasFile := existingFiles[playlist.ID] || existingFiles[playlist.Name]
		if !hasFile {
			log.Debug(p.ctx, "Scanner: Clearing orphaned artwork hash", "playlistId", playlist.ID, "name", playlist.Name)
			err := p.updatePlaylistArtworkHash(playlist.ID, "")
			if err != nil {
				log.Warn(p.ctx, "Scanner: Error clearing artwork hash", "playlistId", playlist.ID, err)
			}
		}
	}
}

func (p *phaseArtwork) clearAllCustomArtworkHashes() {
	playlists, err := p.ds.Playlist(p.ctx).GetAll(model.QueryOptions{
		Filters: squirrel.NotEq{"custom_artwork_hash": ""},
	})
	if err != nil {
		log.Warn(p.ctx, "Scanner: Error getting playlists with custom artwork for cleanup", err)
		return
	}

	for _, playlist := range playlists {
		log.Debug(p.ctx, "Scanner: Clearing artwork hash (artwork folder missing)", "playlistId", playlist.ID, "name", playlist.Name)
		err := p.updatePlaylistArtworkHash(playlist.ID, "")
		if err != nil {
			log.Warn(p.ctx, "Scanner: Error clearing artwork hash", "playlistId", playlist.ID, err)
		}
	}
}

func isValidImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, validExt := range mime.ValidImageExtensions {
		if ext == validExt {
			return true
		}
	}
	return false
}

var _ phase[string] = (*phaseArtwork)(nil)
