package scanner

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/squirrel"
	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/pl"
	"github.com/navidrome/navidrome/utils/slice"
)

func createPhaseFolders(ctx context.Context, ds model.DataStore, cw artwork.CacheWarmer, libs []model.Library, fullRescan bool, changesDetected *atomic.Bool, progress chan<- *scannerStatus) *phaseFolders {
	var jobs []*scanJob
	for _, lib := range libs {
		err := ds.Library(ctx).UpdateLastScanStartedAt(lib.ID, time.Now())
		if err != nil {
			log.Error(ctx, "Scanner: Error updating last scan started at", "lib", lib.Name, err)
		}
		// BFR Check LastScanStartedAt for interrupted full scans
		job, err := newScanJob(ctx, ds, cw, lib, fullRescan)
		if err != nil {
			log.Error(ctx, "Scanner: Error creating scan context", "lib", lib.Name, err)
			continue
		}
		jobs = append(jobs, job)
	}
	return &phaseFolders{jobs: jobs, ctx: ctx, ds: ds, changesDetected: changesDetected, progress: progress}
}

type scanJob struct {
	lib         model.Library
	fs          storage.MusicFS
	cw          artwork.CacheWarmer
	lastUpdates map[string]time.Time
	lock        sync.Mutex
	fullRescan  bool
	numFolders  atomic.Int64
}

func newScanJob(ctx context.Context, ds model.DataStore, cw artwork.CacheWarmer, lib model.Library, fullRescan bool) (*scanJob, error) {
	lastUpdates, err := ds.Folder(ctx).GetLastUpdates(lib)
	if err != nil {
		return nil, fmt.Errorf("error getting last updates: %w", err)
	}
	fileStore, err := storage.For(lib.Path)
	if err != nil {
		log.Error(ctx, "Error getting storage for library", "library", lib.Name, "path", lib.Path, err)
		return nil, fmt.Errorf("error getting storage for library: %w", err)
	}
	fsys, err := fileStore.FS()
	if err != nil {
		log.Error(ctx, "Error getting fs for library", "library", lib.Name, "path", lib.Path, err)
		return nil, fmt.Errorf("error getting fs for library: %w", err)
	}
	return &scanJob{
		lib:         lib,
		fs:          fsys,
		cw:          cw,
		lastUpdates: lastUpdates,
		fullRescan:  fullRescan,
	}, nil
}

func (j *scanJob) popLastUpdate(folderID string) time.Time {
	j.lock.Lock()
	defer j.lock.Unlock()

	lastUpdate := j.lastUpdates[folderID]
	delete(j.lastUpdates, folderID)
	return lastUpdate
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
	jobs            []*scanJob
	ds              model.DataStore
	ctx             context.Context
	changesDetected *atomic.Bool
	progress        chan<- *scannerStatus
}

func (p *phaseFolders) description() string {
	return "Scan all libraries and import new/updated files"
}

func (p *phaseFolders) producer() ppl.Producer[*folderEntry] {
	return ppl.NewProducer(func(put func(entry *folderEntry)) error {
		// TODO Parallelize multiple job when we have multiple libraries
		var total int64
		for _, job := range p.jobs {
			if utils.IsCtxDone(p.ctx) {
				break
			}
			outputChan, err := walkDirTree(p.ctx, job)
			if err != nil {
				log.Warn(p.ctx, "Scanner: Error scanning library", "lib", job.lib.Name, err)
			}
			for folder := range pl.ReadOrDone(p.ctx, outputChan) {
				job.numFolders.Add(1)
				p.progress <- &scannerStatus{
					libID:     job.lib.ID,
					fileCount: uint32(len(folder.audioFiles)),
					lastPath:  folder.path,
					phase:     "1",
				}
				if folder.isOutdated() || job.fullRescan {
					folder.elapsed.Stop()
					put(folder)
				}
			}
			total += job.numFolders.Load()
		}
		log.Debug(p.ctx, "Scanner: Finished loading all folders", "numFolders", total)
		return nil
	}, ppl.Name("traverse filesystem"))
}

func (p *phaseFolders) measure(entry *folderEntry) func() time.Duration {
	entry.elapsed.Start()
	return func() time.Duration { return entry.elapsed.Stop() }
}

func (p *phaseFolders) stages() []ppl.Stage[*folderEntry] {
	return []ppl.Stage[*folderEntry]{
		ppl.NewStage(p.processFolder, ppl.Name("process folder"), ppl.Concurrency(5)),
		ppl.NewStage(p.persistChanges, ppl.Name("persist changes")),
		ppl.NewStage(p.logFolder, ppl.Name("log results")),
	}
}

func (p *phaseFolders) processFolder(entry *folderEntry) (*folderEntry, error) {
	defer p.measure(entry)()

	// Load children mediafiles from DB
	cursor, err := p.ds.MediaFile(p.ctx).GetCursor(model.QueryOptions{
		Filters: squirrel.And{squirrel.Eq{"folder_id": entry.id}},
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

	// Get list of files to import, based on modtime (or all if fullRescan),
	// leave in dbTracks only tracks that are missing (not found in the FS)
	filesToImport := make([]string, 0, len(entry.audioFiles))
	for afPath, af := range entry.audioFiles {
		fullPath := path.Join(entry.path, afPath)
		dbTrack, foundInDB := dbTracks[fullPath]
		if !foundInDB || entry.job.fullRescan {
			filesToImport = append(filesToImport, fullPath)
		} else {
			info, err := af.Info()
			if err != nil {
				log.Warn(p.ctx, "Scanner: Error getting file info", "folder", entry.path, "file", af.Name(), err)
				return nil, err
			}
			if info.ModTime().After(dbTrack.UpdatedAt) || dbTrack.Missing {
				filesToImport = append(filesToImport, fullPath)
			}
		}
		delete(dbTracks, fullPath)
	}

	// Remaining dbTracks are tracks that were not found in the FS, so they should be marked as missing
	entry.missingTracks = slices.Collect(maps.Values(dbTracks))

	// Load metadata from files that need to be imported
	if len(filesToImport) > 0 {
		entry.tracks, entry.tags, err = p.loadTagsFromFiles(entry, filesToImport)
		if err != nil {
			log.Warn(p.ctx, "Scanner: Error loading tags from files. Skipping", "folder", entry.path, err)
			return entry, nil
		}

		entry.albums = p.createAlbumsFromMediaFiles(entry)
		entry.artists = p.createArtistsFromMediaFiles(entry)
	}

	return entry, nil
}

const filesBatchSize = 200

func (p *phaseFolders) loadTagsFromFiles(entry *folderEntry, toImport []string) (model.MediaFiles, model.TagList, error) {
	tracks := make([]model.MediaFile, 0, len(toImport))
	uniqueTags := make(map[string]model.Tag, len(toImport))
	for chunk := range slices.Chunk(toImport, filesBatchSize) {
		allInfo, err := entry.job.fs.ReadTags(chunk...)
		if err != nil {
			log.Warn(p.ctx, "Scanner: Error extracting metadata from files. Skipping", "folder", entry.path, err)
			return nil, nil, err
		}
		for filePath, info := range allInfo {
			md := metadata.New(filePath, info)
			track := md.ToMediaFile()
			track.LibraryID = entry.job.lib.ID
			track.FolderID = entry.id
			tracks = append(tracks, track)
			for _, t := range track.Tags.FlattenAll() {
				uniqueTags[t.ID] = t
			}
		}
	}
	return tracks, slices.Collect(maps.Values(uniqueTags)), nil
}

func (p *phaseFolders) createAlbumsFromMediaFiles(entry *folderEntry) model.Albums {
	grouped := slice.Group(entry.tracks, func(mf model.MediaFile) string { return mf.AlbumID })
	albums := make(model.Albums, 0, len(grouped))
	for _, group := range grouped {
		songs := model.MediaFiles(group)
		album := songs.ToAlbum()
		albums = append(albums, album)
	}
	return albums
}

func (p *phaseFolders) createArtistsFromMediaFiles(entry *folderEntry) model.Artists {
	participants := make(model.Participations, len(entry.tracks)*3) // preallocate ~3 artists per track
	for _, track := range entry.tracks {
		participants.Merge(track.Participations)
	}
	return participants.All()
}

func (p *phaseFolders) persistChanges(entry *folderEntry) (*folderEntry, error) {
	defer p.measure(entry)()
	p.changesDetected.Store(true)

	err := p.ds.WithTx(func(tx model.DataStore) error {
		// Instantiate all repositories just once per folder
		folderRepo := tx.Folder(p.ctx)
		tagRepo := tx.Tag(p.ctx)
		artistRepo := tx.Artist(p.ctx)
		libraryRepo := tx.Library(p.ctx)
		albumRepo := tx.Album(p.ctx)
		mfRepo := tx.MediaFile(p.ctx)

		// Save folder to DB
		folder := model.NewFolder(entry.job.lib, entry.path)
		folder.NumAudioFiles = len(entry.audioFiles)
		folder.ImageFiles = slices.Collect(maps.Keys(entry.imageFiles))
		folder.ImagesUpdatedAt = entry.imagesUpdatedAt
		err := folderRepo.Put(folder)
		if err != nil {
			log.Error(p.ctx, "Scanner: Error persisting folder to DB", "folder", entry.path, err)
			return err
		}

		// Save all tags to DB
		err = tagRepo.Add(entry.tags...)
		if err != nil {
			log.Error(p.ctx, "Scanner: Error persisting tags to DB", "folder", entry.path, err)
			return err
		}

		// Save all new/modified artists to DB. Their information will be incomplete, but they will be refreshed later
		for i := range entry.artists {
			err = artistRepo.Put(&entry.artists[i], "name", "mbz_artist_id", "sort_artist_name", "order_artist_name")
			if err != nil {
				log.Error(p.ctx, "Scanner: Error persisting artist to DB", "folder", entry.path, "artist", entry.artists[i].Name, err)
				return err
			}
			err = libraryRepo.AddArtist(entry.job.lib.ID, entry.artists[i].ID)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error adding artist to library", "lib", entry.job.lib.ID, "artist", entry.artists[i].Name, err)
				return err
			}
			if entry.artists[i].Name != consts.UnknownArtist && entry.artists[i].Name != consts.VariousArtists {
				entry.job.cw.PreCache(entry.artists[i].CoverArtID())
			}
		}

		// Save all new/modified albums to DB. Their information will be incomplete, but they will be refreshed later
		for i := range entry.albums {
			err = albumRepo.Put(&entry.albums[i])
			if err != nil {
				log.Error(p.ctx, "Scanner: Error persisting album to DB", "folder", entry.path, "album", entry.albums[i], err)
				return err
			}
			if entry.albums[i].Name != consts.UnknownAlbum {
				entry.job.cw.PreCache(entry.albums[i].CoverArtID())
			}
		}

		// Save all tracks to DB
		for i := range entry.tracks {
			err = mfRepo.Put(&entry.tracks[i])
			if err != nil {
				log.Error(p.ctx, "Scanner: Error persisting mediafile to DB", "folder", entry.path, "track", entry.tracks[i], err)
				return err
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
			albumsToUpdate := slices.Collect(maps.Keys(groupedMissingTracks))
			err = albumRepo.Touch(albumsToUpdate...)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error touching album", "folder", entry.path, "albums", albumsToUpdate, err)
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Error(p.ctx, "Scanner: Error persisting changes to DB", "folder", entry.path, err)
	}
	return entry, err
}

func (p *phaseFolders) logFolder(entry *folderEntry) (*folderEntry, error) {
	logCall := log.Info
	if entry.hasNoFiles() {
		logCall = log.Trace
	}
	logCall(p.ctx, "Scanner: Completed processing folder",
		"audioCount", len(entry.audioFiles), "imageCount", len(entry.imageFiles), "plsCount", len(entry.playlists),
		"elapsed", entry.elapsed.Elapsed(), "tracksMissing", len(entry.missingTracks),
		"tracksImported", len(entry.tracks), "library", entry.job.lib.Name, consts.Zwsp+"folder", entry.path)
	return entry, nil
}

func (p *phaseFolders) finalize(err error) error {
	errF := p.ds.WithTx(func(tx model.DataStore) error {
		for _, job := range p.jobs {
			if len(job.lastUpdates) == 0 {
				continue
			}
			folderIDs := slices.Collect(maps.Keys(job.lastUpdates))
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
		}
		return nil
	})
	return errors.Join(err, errF)
}

var _ phase[*folderEntry] = (*phaseFolders)(nil)
