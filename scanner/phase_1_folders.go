package scanner

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/ffmpeg"
	"github.com/navidrome/navidrome/core/ftsnormalize"
	"github.com/navidrome/navidrome/core/metadataworker"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/model/query"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/pl"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/sync/errgroup"
)

func createPhaseFolders(ctx context.Context, state *scanState, ds model.DataStore, cw artwork.CacheWarmer) *phaseFolders {
	var jobs []*scanJob

	// Create scan jobs for all libraries
	for _, lib := range state.libraries {
		// Get target folders for this library if selective scan
		var targetFolders []string
		if state.isSelectiveScan() {
			targetFolders = state.targets[lib.ID]
		}

		job, err := newScanJob(ctx, ds, cw, lib, state.fullScan, targetFolders)
		if err != nil {
			log.Error(ctx, "Scanner: Error creating scan context", "lib", lib.Name, err)
			state.sendError(err)
			continue
		}
		jobs = append(jobs, job)
	}

	return &phaseFolders{jobs: jobs, ctx: ctx, ds: ds, state: state}
}

type scanJob struct {
	lib           model.Library
	fs            storage.MusicFS
	localRoot     string
	cw            artwork.CacheWarmer
	lastUpdates   map[string]model.FolderUpdateInfo // Holds last update info for all (DB) folders in this library
	knownHashes   map[string]string                 // folder path -> hash for Rust summary events
	targetFolders []string                          // Specific folders to scan (including all descendants)
	lock          sync.Mutex
	numFolders    atomic.Int64
}

type mediaFileBatchPutter interface {
	PutAll(...*model.MediaFile) error
}

func newScanJob(ctx context.Context, ds model.DataStore, cw artwork.CacheWarmer, lib model.Library, fullScan bool, targetFolders []string) (*scanJob, error) {
	// Get folder updates, optionally filtered to specific target folders
	lastUpdates, err := ds.Folder(ctx).GetFolderUpdateInfo(lib, targetFolders...)
	if err != nil {
		return nil, fmt.Errorf("getting last updates: %w", err)
	}

	fileStore, err := storage.For(lib.Path)
	if err != nil {
		log.Error(ctx, "Error getting storage for library", "library", lib.Name, "path", lib.Path, err)
		return nil, fmt.Errorf("getting storage for library: %w", err)
	}
	fsys, err := fileStore.FS()
	if err != nil {
		log.Error(ctx, "Error getting fs for library", "library", lib.Name, "path", lib.Path, err)
		return nil, fmt.Errorf("getting fs for library: %w", err)
	}

	// Ensure FullScanInProgress reflects the current scan request.
	// This is important when resuming an interrupted quick scan as a full scan:
	// the DB may have FullScanInProgress=false, but we need it true for isOutdated() to work correctly.
	lib.FullScanInProgress = lib.FullScanInProgress || fullScan
	localRoot := ""
	if localFS, ok := fsys.(storage.LocalPathFS); ok {
		localRoot = localFS.RootPath()
	}

	knownHashes, err := loadKnownFolderHashes(ctx, ds, lib, targetFolders)
	if err != nil {
		return nil, fmt.Errorf("loading folder hashes: %w", err)
	}

	return &scanJob{
		lib:           lib,
		fs:            fsys,
		localRoot:     localRoot,
		cw:            cw,
		lastUpdates:   lastUpdates,
		knownHashes:   knownHashes,
		targetFolders: targetFolders,
	}, nil
}

func loadKnownFolderHashes(ctx context.Context, ds model.DataStore, lib model.Library, targetFolders []string) (map[string]string, error) {
	folders, err := ds.Folder(ctx).GetAll(model.QueryOptions{
		Filters: query.And(query.Eq("library_id", lib.ID), query.NotMissing()),
	})
	if err != nil {
		return nil, err
	}
	hashes := make(map[string]string, len(folders))
	for _, folder := range folders {
		if folder.Hash == "" || !folderHashInScanTargets(folder.Path, targetFolders) {
			continue
		}
		hashes[folder.Path] = folder.Hash
	}
	return hashes, nil
}

func folderHashInScanTargets(folderPath string, targetFolders []string) bool {
	if len(targetFolders) == 0 {
		return true
	}
	for _, target := range targetFolders {
		if folderPath == target || strings.HasPrefix(folderPath, target+"/") {
			return true
		}
	}
	return false
}

// popLastUpdate retrieves and removes the last update info for the given folder ID
// This is used to track which folders have been found during the walk_dir_tree
func (j *scanJob) popLastUpdate(folderID string) model.FolderUpdateInfo {
	j.lock.Lock()
	defer j.lock.Unlock()

	lastUpdate := j.lastUpdates[folderID]
	delete(j.lastUpdates, folderID)
	return lastUpdate
}

// createFolderEntry creates a new folderEntry for the given path, using the last update info from the job
// to populate the previous update time and hash. It also removes the folder from the job's lastUpdates map.
// This is used to track which folders have been found during the walk_dir_tree.
func (j *scanJob) createFolderEntry(path string) *folderEntry {
	id := model.FolderID(j.lib, path)
	info := j.popLastUpdate(id)
	return newFolderEntry(j, id, path, info.UpdatedAt, info.Hash)
}

// phaseFolders represents the first phase of the scanning process, which is responsible
// for scanning all libraries and importing new or updated files. This phase involves
// traversing the directory tree of each library, identifying new or modified media files,
// and updating the database with the relevant information.
//
// The phaseFolders struct holds the context, data store, and jobs required for the scanning
// process. Each job represents a library being scanned, and contains information about the
// library, file system, and the last updates of the folders.
//
// The phaseFolders struct implements the phase interface, providing methods to produce
// folder entries, process folders, persist changes to the database, and log the results.
type phaseFolders struct {
	jobs             []*scanJob
	ds               model.DataStore
	ctx              context.Context
	state            *scanState
	prevAlbumPIDConf string
}

func (p *phaseFolders) description() string {
	return "Scan all libraries and import new/updated files"
}

func (p *phaseFolders) producer() ppl.Producer[*folderEntry] {
	return ppl.NewProducer(func(put func(entry *folderEntry)) error {
		var err error
		p.prevAlbumPIDConf, err = p.ds.Property(p.ctx).DefaultGet(consts.PIDAlbumKey, "")
		if err != nil {
			return fmt.Errorf("getting album PID conf: %w", err)
		}

		var totalChanged atomic.Int64
		group, groupCtx := errgroup.WithContext(p.ctx)
		group.SetLimit(max(int(conf.Server.DevScannerThreads), 1))
		for _, job := range p.jobs {
			group.Go(func() error {
				if utils.IsCtxDone(groupCtx) {
					return nil
				}
				outputChan, walkErr := walkDirTree(groupCtx, job, job.targetFolders...)
				if walkErr != nil {
					log.Warn(p.ctx, "Scanner: Error scanning library", "lib", job.lib.Name, walkErr)
				}
				for folder := range pl.ReadOrDone(groupCtx, outputChan) {
					job.numFolders.Add(1)
					p.state.sendProgress(&ProgressInfo{
						LibID: job.lib.ID, FileCount: uint32(len(folder.audioFiles)), Path: folder.path, Phase: "1",
					})
					log.Trace(p.ctx, "Scanner: Checking folder state", " folder", folder.path, "_updTime", folder.updTime,
						"_modTime", folder.modTime, "_lastScanStartedAt", folder.job.lib.LastScanStartedAt,
						"numAudioFiles", len(folder.audioFiles), "numImageFiles", len(folder.imageFiles),
						"numPlaylists", folder.numPlaylists, "numSubfolders", folder.numSubFolders)
					if folder.isOutdated() {
						if !p.state.fullScan {
							if folder.hasNoFiles() && folder.isNew() {
								log.Trace(p.ctx, "Scanner: Skipping new folder with no files", "folder", folder.path, "lib", job.lib.Name)
								continue
							}
							log.Debug(p.ctx, "Scanner: Detected changes in folder", "folder", folder.path, "lastUpdate", folder.modTime, "lib", job.lib.Name)
						}
						totalChanged.Add(1)
						folder.elapsed.Stop()
						put(folder)
					} else {
						log.Trace(p.ctx, "Scanner: Skipping up-to-date folder", "folder", folder.path, "lastUpdate", folder.modTime, "lib", job.lib.Name)
					}
				}
				return nil
			})
		}
		if err := group.Wait(); err != nil {
			return err
		}
		var total int64
		for _, job := range p.jobs {
			total += job.numFolders.Load()
		}
		log.Debug(p.ctx, "Scanner: Finished loading all folders", "numFolders", total, "numChanged", totalChanged.Load())
		return nil
	}, ppl.Name("traverse filesystem"))
}

func (p *phaseFolders) measure(entry *folderEntry) func() time.Duration {
	entry.elapsed.Start()
	return func() time.Duration { return entry.elapsed.Stop() }
}

func (p *phaseFolders) stages() []ppl.Stage[*folderEntry] {
	return []ppl.Stage[*folderEntry]{
		ppl.NewStage(p.processFolder, ppl.Name("process folder"), ppl.Concurrency(conf.Server.DevScannerThreads)),
		ppl.NewStage(p.persistChanges, ppl.Name("persist changes")),
		ppl.NewStage(p.logFolder, ppl.Name("log results")),
	}
}

func (p *phaseFolders) processFolder(entry *folderEntry) (*folderEntry, error) {
	defer p.measure(entry)()

	// Load children mediafiles from DB
	cursor, err := p.ds.MediaFile(p.ctx).GetCursor(model.QueryOptions{
		Filters: query.Eq("folder_id", entry.id),
	})
	if err != nil {
		log.Error(p.ctx, "Scanner: Error loading mediafiles from DB", "folder", entry.path, err)
		return entry, err
	}
	dbTracks := make(map[string]*model.MediaFile)
	for mf, err := range cursor {
		if err != nil {
			log.Error(p.ctx, "Scanner: Error loading mediafiles from DB", "folder", entry.path, err)
			return entry, err
		}
		dbTracks[mf.Path] = &mf
	}

	// Get list of files to import, based on modtime (or all if fullScan),
	// leave in dbTracks only tracks that are missing (not found in the FS)
	filesToImport := make(map[string]*model.MediaFile, len(entry.audioFiles))
	for afPath, af := range entry.audioFiles {
		fullPath := path.Join(entry.path, afPath)
		dbTrack, foundInDB := dbTracks[fullPath]
		if !foundInDB || p.state.fullScan {
			filesToImport[fullPath] = dbTrack
		} else {
			info, err := entry.fileInfo(afPath, af)
			if err != nil {
				log.Warn(p.ctx, "Scanner: Error getting file info; keeping existing DB entry and continuing", "folder", entry.path, "file", af.Name(), err)
				p.state.sendWarning(fmt.Sprintf("Error getting file info for %s/%s: %v", entry.path, af.Name(), err))
				delete(dbTracks, fullPath)
				continue
			}
			if info.ModTime().After(dbTrack.UpdatedAt) || dbTrack.Missing {
				filesToImport[fullPath] = dbTrack
			}
		}
		delete(dbTracks, fullPath)
	}

	// Remaining dbTracks are tracks that were not found in the FS, so they should be marked as missing
	entry.missingTracks = mapValues(dbTracks)

	// Load metadata from files that need to be imported
	if len(filesToImport) > 0 {
		err = p.loadTagsFromFiles(entry, filesToImport)
		if err != nil {
			log.Warn(p.ctx, "Scanner: Error loading tags from files. Skipping", "folder", entry.path, err)
			p.state.sendWarning(fmt.Sprintf("Error loading tags from files in %s: %v", entry.path, err))
			return entry, nil
		}

		p.createAlbumsFromMediaFiles(entry)
		p.createArtistsFromMediaFiles(entry)
	}

	return entry, nil
}

const filesBatchSize = 200

// loadTagsFromFiles reads metadata from the files in the given list and populates
// the entry's tracks and tags with the results.
func (p *phaseFolders) loadTagsFromFiles(entry *folderEntry, toImport map[string]*model.MediaFile) error {
	tracks := make([]model.MediaFile, 0, len(toImport))
	uniqueTags := make(map[string]model.Tag, len(toImport))
	var audioProbe ffmpeg.FFmpeg
	for chunk := range slice.CollectChunks(maps.Keys(toImport), filesBatchSize) {
		allInfo, err := p.readTagsResilient(entry, chunk)
		if err != nil {
			log.Warn(p.ctx, "Scanner: Error extracting metadata from files. Skipping batch", "folder", entry.path, err)
			return err
		}
		for filePath, info := range allInfo {
			md := metadata.New(filePath, info)
			track := md.ToMediaFile(entry.job.lib.ID, entry.id)
			if conf.Server.DevEnableMediaFileProbe && needsAudioProbe(&track) {
				if audioProbe == nil {
					audioProbe = ffmpeg.New()
				}
				p.probeMissingAudioProperties(&track, entry.job.lib.Path, filePath, audioProbe)
			}
			tracks = append(tracks, track)
			for _, t := range track.Tags.FlattenAll() {
				uniqueTags[t.ID] = t
			}

			// Keep track of any album ID changes, to reassign annotations later
			prevAlbumID := ""
			if prev := toImport[filePath]; prev != nil {
				prevAlbumID = prev.AlbumID
			} else {
				prevAlbumID = md.AlbumID(track, p.prevAlbumPIDConf)
			}
			_, ok := entry.albumIDMap[track.AlbumID]
			if prevAlbumID != track.AlbumID && !ok {
				entry.albumIDMap[track.AlbumID] = prevAlbumID
			}
		}
	}
	entry.tracks = tracks
	entry.tags = mapValues(uniqueTags)
	return nil
}

func (p *phaseFolders) readTagsResilient(entry *folderEntry, paths []string) (map[string]metadata.Info, error) {
	readTags := func(paths ...string) (map[string]metadata.Info, error) {
		ctx := metadataworker.WithLibraryID(p.ctx, entry.job.lib.ID)
		if reader, ok := entry.job.fs.(storage.ContextMusicFS); ok {
			return reader.ReadTagsContext(ctx, paths...)
		}
		return entry.job.fs.ReadTags(paths...)
	}
	allInfo, err := readTags(paths...)
	if err == nil || len(paths) <= 1 {
		return allInfo, err
	}
	if ctxErr := p.ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}

	log.Warn(p.ctx, "Scanner: Batch metadata extraction failed; retrying files in smaller batches", "folder", entry.path, "files", len(paths), err)
	return p.readTagsBisect(readTags, paths)
}

func (p *phaseFolders) readTagsBisect(readTags func(...string) (map[string]metadata.Info, error), paths []string) (map[string]metadata.Info, error) {
	if ctxErr := p.ctx.Err(); ctxErr != nil {
		return nil, ctxErr
	}
	info, err := readTags(paths...)
	if err == nil {
		return info, nil
	}
	if len(paths) <= 1 {
		return nil, err
	}
	mid := len(paths) / 2
	left, leftErr := p.readTagsBisect(readTags, paths[:mid])
	right, rightErr := p.readTagsBisect(readTags, paths[mid:])
	result := make(map[string]metadata.Info, len(paths))
	maps.Copy(result, left)
	maps.Copy(result, right)
	joined := errors.Join(leftErr, rightErr)
	if len(result) < len(paths) {
		return result, errors.Join(joined, fmt.Errorf("metadata extraction incomplete: got %d of %d files", len(result), len(paths)))
	}
	return result, nil
}

func (p *phaseFolders) probeMissingAudioProperties(track *model.MediaFile, libPath, filePath string, audioProbe ffmpeg.FFmpeg) {
	probePath, ok := scannerProbePath(libPath, filePath)
	if !ok {
		return
	}
	result, err := audioProbe.ProbeAudioStream(p.ctx, probePath)
	if err != nil {
		log.Debug(p.ctx, "Scanner: Skipping audio probe fallback", "path", filePath, err)
		return
	}
	mergeAudioProbeProperties(track, result)
	if data, err := json.Marshal(result); err == nil {
		track.ProbeData = string(data)
	}
}

func needsAudioProbe(track *model.MediaFile) bool {
	return track.SampleRate <= 0 || track.BitRate <= 0 || track.Channels <= 0 || track.Codec == ""
}

func mergeAudioProbeProperties(track *model.MediaFile, result *ffmpeg.AudioProbeResult) {
	if result == nil {
		return
	}
	if track.SampleRate <= 0 && result.SampleRate > 0 {
		track.SampleRate = result.SampleRate
	}
	if track.BitRate <= 0 && result.BitRate > 0 {
		track.BitRate = result.BitRate
	}
	if track.Channels <= 0 && result.Channels > 0 {
		track.Channels = result.Channels
	}
	if track.Codec == "" && result.Codec != "" {
		track.Codec = strings.ToUpper(result.Codec)
	}
	if track.BitDepth == nil && result.BitDepth > 0 {
		track.BitDepth = &result.BitDepth
	}
}

func scannerProbePath(libPath, filePath string) (string, bool) {
	u, err := url.Parse(libPath)
	if err == nil && u.Scheme != "" {
		if u.Scheme != storage.LocalSchemaID {
			return "", false
		}
		return filepath.Join(u.Path, filePath), true
	}
	return filepath.Join(libPath, filePath), true
}

// createAlbumsFromMediaFiles groups the entry's tracks by album ID and creates albums
func (p *phaseFolders) createAlbumsFromMediaFiles(entry *folderEntry) {
	grouped := slice.Group(entry.tracks, func(mf model.MediaFile) string { return mf.AlbumID })
	albums := make(model.Albums, 0, len(grouped))
	for _, group := range grouped {
		songs := model.MediaFiles(group)
		album := songs.ToAlbum()
		albums = append(albums, album)
	}
	entry.albums = albums
}

// createArtistsFromMediaFiles creates artists from the entry's tracks
func (p *phaseFolders) createArtistsFromMediaFiles(entry *folderEntry) {
	participants := make(model.Participants, len(entry.tracks)*3) // preallocate ~3 artists per track
	for _, track := range entry.tracks {
		participants.Merge(track.Participants)
	}
	entry.artists = participants.AllArtists()
}

func (p *phaseFolders) persistChanges(entry *folderEntry) (*folderEntry, error) {
	defer p.measure(entry)()
	p.state.changesDetected.Store(true)

	// Collect artwork IDs only when pre-cache is enabled. On small servers this
	// avoids per-folder slice growth during scans where the cache warmer is a noop.
	preCacheArtwork := conf.Server.EnableArtworkPrecache && conf.Server.ImageCacheSize != "0"
	var artworkIDs []model.ArtworkID

	err := p.ds.WithTx(func(tx model.DataStore) error {
		// Instantiate all repositories just once per folder
		folderRepo := tx.Folder(p.ctx)
		tagRepo := tx.Tag(p.ctx)
		artistRepo := tx.Artist(p.ctx)
		libraryRepo := tx.Library(p.ctx)
		albumRepo := tx.Album(p.ctx)
		mfRepo := tx.MediaFile(p.ctx)

		// Save folder to DB
		folder := entry.toFolder()
		err := folderRepo.Put(folder)
		if err != nil {
			log.Error(p.ctx, "Scanner: Error persisting folder to DB", "folder", entry.path, err)
			return err
		}

		// Save all tags to DB
		err = tagRepo.Add(entry.job.lib.ID, entry.tags...)
		if err != nil {
			log.Error(p.ctx, "Scanner: Error persisting tags to DB", "folder", entry.path, err)
			return err
		}

		// Prefill search_normalized in one Rust metadata batch RPC before artist Puts.
		prefillArtistSearchNormalized(p.ctx, entry.artists)

		// Save all new/modified artists to DB. Their information will be incomplete, but they will be refreshed later
		artistIDs := make([]string, 0, len(entry.artists))
		for i := range entry.artists {
			err = artistRepo.Put(&entry.artists[i], "name",
				"mbz_artist_id", "sort_artist_name", "order_artist_name", "full_text", "search_normalized", "updated_at")
			if err != nil {
				log.Error(p.ctx, "Scanner: Error persisting artist to DB", "folder", entry.path, "artist", entry.artists[i].Name, err)
				return err
			}
			artistIDs = append(artistIDs, entry.artists[i].ID)
			if preCacheArtwork && entry.artists[i].Name != consts.UnknownArtist && entry.artists[i].Name != consts.VariousArtists {
				artworkIDs = append(artworkIDs, entry.artists[i].CoverArtID())
			}
		}
		if err = libraryRepo.AddArtist(entry.job.lib.ID, artistIDs...); err != nil {
			log.Error(p.ctx, "Scanner: Error adding artists to library", "lib", entry.job.lib.ID, "count", len(artistIDs), err)
			return err
		}

		// Prefill album search_normalized in one batch RPC before album Puts.
		prefillAlbumSearchNormalized(p.ctx, entry.albums)

		// Save all new/modified albums to DB. Their information will be incomplete, but they will be refreshed later
		for i := range entry.albums {
			err = p.persistAlbum(albumRepo, &entry.albums[i], entry.albumIDMap)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error persisting album to DB", "folder", entry.path, "album", entry.albums[i], err)
				return err
			}
			if preCacheArtwork && entry.albums[i].Name != consts.UnknownAlbum {
				artworkIDs = append(artworkIDs, entry.albums[i].CoverArtID())
			}
		}

		// Save all tracks to DB. The SQL repository collapses relationship-table
		// rewrites into folder-sized batches; alternate stores retain the legacy
		// per-track contract through this fallback.
		if batchRepo, ok := mfRepo.(mediaFileBatchPutter); ok {
			tracks := make([]*model.MediaFile, len(entry.tracks))
			for i := range entry.tracks {
				tracks[i] = &entry.tracks[i]
			}
			if err = batchRepo.PutAll(tracks...); err != nil {
				log.Error(p.ctx, "Scanner: Error persisting mediafile batch to DB", "folder", entry.path, "tracks", len(tracks), err)
				return err
			}
		} else {
			for i := range entry.tracks {
				err = mfRepo.Put(&entry.tracks[i])
				if err != nil {
					log.Error(p.ctx, "Scanner: Error persisting mediafile to DB", "folder", entry.path, "track", entry.tracks[i], err)
					return err
				}
			}
		}

		// Mark all missing tracks as not available
		if len(entry.missingTracks) > 0 {
			err = mfRepo.MarkMissing(true, entry.missingTracks...)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error marking missing tracks", "folder", entry.path, err)
				return err
			}

			// Touch all albums that have missing tracks, so they get refreshed in later phases
			groupedMissingTracks := slice.ToMap(entry.missingTracks, func(mf *model.MediaFile) (string, struct{}) {
				return mf.AlbumID, struct{}{}
			})
			albumsToUpdate := mapKeys(groupedMissingTracks)
			err = albumRepo.Touch(albumsToUpdate...)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error touching album", "folder", entry.path, "albums", albumsToUpdate, err)
				return err
			}
		}
		return nil
	}, "scanner: persist changes")
	if err != nil {
		log.Error(p.ctx, "Scanner: Error persisting changes to DB", "folder", entry.path, err)
	}

	// Pre-cache artwork after the transaction commits successfully
	if err == nil && len(artworkIDs) > 0 {
		for _, artID := range artworkIDs {
			entry.job.cw.PreCache(artID)
		}
	}

	return entry, err
}

// persistAlbum persists the given album to the database, and reassigns annotations from the previous album ID
func (p *phaseFolders) persistAlbum(repo model.AlbumRepository, a *model.Album, idMap map[string]string) error {
	prevID := idMap[a.ID]
	log.Trace(p.ctx, "Persisting album", "album", a.Name, "albumArtist", a.AlbumArtist, "id", a.ID, "prevID", cmp.Or(prevID, "nil"))
	if err := repo.Put(a); err != nil {
		return fmt.Errorf("persisting album %s: %w", a.ID, err)
	}
	if prevID == "" {
		return nil
	}

	// Reassign annotation from previous album to new album
	log.Trace(p.ctx, "Reassigning album annotations", "from", prevID, "to", a.ID, "album", a.Name)
	if err := repo.ReassignAnnotation(prevID, a.ID); err != nil {
		log.Warn(p.ctx, "Scanner: Could not reassign annotations", "from", prevID, "to", a.ID, "album", a.Name, err)
		p.state.sendWarning(fmt.Sprintf("Could not reassign annotations from %s to %s ('%s'): %v", prevID, a.ID, a.Name, err))
	}

	// Keep created_at field from previous instance of the album
	if err := repo.CopyAttributes(prevID, a.ID, "created_at"); err != nil {
		// Silently ignore when the previous album is not found
		if !errors.Is(err, model.ErrNotFound) {
			log.Warn(p.ctx, "Scanner: Could not copy fields", "from", prevID, "to", a.ID, "album", a.Name, err)
			p.state.sendWarning(fmt.Sprintf("Could not copy fields from %s to %s ('%s'): %v", prevID, a.ID, a.Name, err))
		}
	}
	// Don't keep track of this mapping anymore
	delete(idMap, a.ID)
	return nil
}

func (p *phaseFolders) logFolder(entry *folderEntry) (*folderEntry, error) {
	logCall := log.Info
	if entry.isEmpty() {
		logCall = log.Trace
	}
	logCall(p.ctx, "Scanner: Completed processing folder",
		"audioCount", len(entry.audioFiles), "imageCount", len(entry.imageFiles), "plsCount", entry.numPlaylists,
		"elapsed", entry.elapsed.Elapsed(), "tracksMissing", len(entry.missingTracks),
		"tracksImported", len(entry.tracks), "library", entry.job.lib.Name, consts.Zwsp+"folder", entry.path)
	return entry, nil
}

func (p *phaseFolders) finalize(err error) error {
	errF := p.ds.WithTx(func(tx model.DataStore) error {
		for _, job := range p.jobs {
			// Mark all folders that were not updated as missing
			if len(job.lastUpdates) == 0 {
				continue
			}
			folderIDs := mapKeys(job.lastUpdates)
			err := tx.Folder(p.ctx).MarkMissing(true, folderIDs...)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error marking missing folders", "lib", job.lib.Name, err)
				return err
			}
			err = tx.MediaFile(p.ctx).MarkMissingByFolder(true, folderIDs...)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error marking tracks in missing folders", "lib", job.lib.Name, err)
				return err
			}
			// Touch all albums that have missing folders, so they get refreshed in later phases
			_, err = tx.Album(p.ctx).TouchByMissingFolder()
			if err != nil {
				log.Error(p.ctx, "Scanner: Error touching albums with missing folders", "lib", job.lib.Name, err)
				return err
			}
		}
		return nil
	}, "scanner: finalize phaseFolders")
	return errors.Join(err, errF)
}

var _ phase[*folderEntry] = (*phaseFolders)(nil)

func prefillArtistSearchNormalized(ctx context.Context, artists []model.Artist) {
	groups := make([][]string, 0, len(artists))
	indexes := make([]int, 0, len(artists))
	for i := range artists {
		if artists[i].SearchNormalized != "" {
			continue
		}
		groups = append(groups, []string{artists[i].Name})
		indexes = append(indexes, i)
	}
	if len(groups) == 0 {
		return
	}
	normalized := ftsnormalize.NormalizeMany(ctx, groups)
	for j, idx := range indexes {
		artists[idx].SearchNormalized = normalized[j]
	}
}

func prefillAlbumSearchNormalized(ctx context.Context, albums []model.Album) {
	groups := make([][]string, 0, len(albums))
	indexes := make([]int, 0, len(albums))
	for i := range albums {
		if albums[i].SearchNormalized != "" {
			continue
		}
		groups = append(groups, []string{albums[i].Name, albums[i].AlbumArtist})
		indexes = append(indexes, i)
	}
	if len(groups) == 0 {
		return
	}
	normalized := ftsnormalize.NormalizeMany(ctx, groups)
	for j, idx := range indexes {
		albums[idx].SearchNormalized = normalized[j]
	}
}
