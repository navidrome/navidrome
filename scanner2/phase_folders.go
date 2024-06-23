package scanner2

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Masterminds/squirrel"
	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/pl"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/exp/maps"
)

func createPhaseFolders(ctx context.Context, ds model.DataStore, libs []model.Library, fullRescan bool) *phaseFolders {
	var jobs []*scanJob
	for _, lib := range libs {
		err := ds.Library(ctx).UpdateLastScanStartedAt(lib.ID, time.Now())
		if err != nil {
			log.Error(ctx, "Scanner: Error updating last scan started at", "lib", lib.Name, err)
		}
		// TODO Check LastScanStartedAt for interrupted full scans
		job, err := newScanJob(ctx, ds, lib, fullRescan)
		if err != nil {
			log.Error(ctx, "Scanner: Error creating scan context", "lib", lib.Name, err)
			continue
		}
		jobs = append(jobs, job)
	}
	return &phaseFolders{jobs: jobs, ctx: ctx, ds: ds}
}

type scanJob struct {
	lib         model.Library
	fs          storage.MusicFS
	lastUpdates map[string]time.Time
	lock        sync.Mutex
	fullRescan  bool
	numFolders  atomic.Int64
}

func newScanJob(ctx context.Context, ds model.DataStore, lib model.Library, fullRescan bool) (*scanJob, error) {
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

type phaseFolders struct {
	jobs []*scanJob
	ds   model.DataStore
	ctx  context.Context
}

func (p *phaseFolders) description() string {
	return "Scan all libraries and import new/updated files"
}

func (p *phaseFolders) producer() ppl.Producer[*folderEntry] {
	return ppl.NewProducer(func(put func(entry *folderEntry)) error {
		// TODO Parallelize multiple job
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
				if folder.isOutdated() || job.fullRescan {
					put(folder)
				}
			}
			total += job.numFolders.Load()
		}
		log.Info(p.ctx, "Scanner: Finished loading all folders", "numFolders", total)
		return nil
	}, ppl.Name("produce folders"))
}

func (p *phaseFolders) stages() []ppl.Stage[*folderEntry] {
	return []ppl.Stage[*folderEntry]{
		ppl.NewStage(p.processFolder, ppl.Name("process folder")),
		ppl.NewStage(p.persistChanges, ppl.Name("persist changes")),
		ppl.NewStage(p.logFolder, ppl.Name("log folder")),
	}
}

func (p *phaseFolders) processFolder(entry *folderEntry) (*folderEntry, error) {
	// Load children mediafiles from DB
	mfs, err := p.ds.MediaFile(p.ctx).GetAll(model.QueryOptions{
		Filters: squirrel.And{squirrel.Eq{"folder_id": entry.id}},
	})
	if err != nil {
		log.Error(p.ctx, "Scanner: Error loading mediafiles from DB", "folder", entry.path, err)
		return entry, err
	}
	dbTracks := slice.ToMap(mfs, func(mf model.MediaFile) (string, model.MediaFile) { return mf.Path, mf })

	// Get list of files to import, leave in dbTracks only tracks that are missing
	var filesToImport []string
	for afPath, af := range entry.audioFiles {
		fullPath := filepath.Join(entry.path, afPath)
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

	// Remaining dbTracks are tracks that were not found in the folder, so they should be marked as missing
	entry.missingTracks = maps.Values(dbTracks)

	if len(filesToImport) > 0 {
		entry.tracks, entry.tags, err = p.loadTagsFromFiles(entry, filesToImport)
		if err != nil {
			log.Warn(p.ctx, "Scanner: Error loading tags from files. Skipping", "folder", entry.path, err)
			return entry, nil
		}

		entry.albums = p.loadAlbumsFromMediaFiles(entry)
		entry.artists = p.loadArtistsFromMediaFiles(entry)
	}

	return entry, nil
}

const filesBatchSize = 200

func (p *phaseFolders) loadTagsFromFiles(entry *folderEntry, toImport []string) (model.MediaFiles, model.TagList, error) {
	tracks := model.MediaFiles{}
	uniqueTags := make(map[string]model.Tag)
	err := slice.RangeByChunks(toImport, filesBatchSize, func(chunk []string) error {
		allInfo, err := entry.job.fs.ReadTags(toImport...)
		if err != nil {
			log.Warn(p.ctx, "Scanner: Error extracting metadata from files. Skipping", "folder", entry.path, err)
			return err
		}
		for path, info := range allInfo {
			md := metadata.New(path, info)
			track := md.ToMediaFile()
			track.LibraryID = entry.job.lib.ID
			track.FolderID = entry.id
			tracks = append(tracks, track)
			for _, t := range track.Tags.FlattenAll() {
				uniqueTags[t.ID] = t
			}
		}
		return nil
	})
	return tracks, maps.Values(uniqueTags), err
}

func (p *phaseFolders) loadAlbumsFromMediaFiles(entry *folderEntry) model.Albums {
	grouped := slice.Group(entry.tracks, func(mf model.MediaFile) string { return mf.AlbumID })
	var albums model.Albums
	for _, group := range grouped {
		songs := model.MediaFiles(group)
		album := songs.ToAlbum()
		albums = append(albums, album)
	}
	return albums
}

func (p *phaseFolders) loadArtistsFromMediaFiles(entry *folderEntry) model.Artists {
	participants := model.Participations{}
	for _, track := range entry.tracks {
		participants.Merge(track.Participations)
	}
	return participants.All()
}

func (p *phaseFolders) persistChanges(entry *folderEntry) (*folderEntry, error) {
	err := p.ds.WithTx(func(tx model.DataStore) error {
		// Save folder to DB
		err := tx.Folder(p.ctx).Put(entry.job.lib, entry.path)
		if err != nil {
			log.Error(p.ctx, "Scanner: Error persisting folder to DB", "folder", entry.path, err)
			return err
		}

		// Save all tags to DB
		err = tx.Tag(p.ctx).Add(entry.tags...)
		if err != nil {
			log.Error(p.ctx, "Scanner: Error persisting tags to DB", "folder", entry.path, err)
			return err
		}

		// Save all new/modified artists to DB. Their information will be incomplete, but they will be refreshed later
		for i := range entry.artists {
			err := tx.Artist(p.ctx).Put(&entry.artists[i], "name", "mbz_artist_id", "sort_artist_name", "order_artist_name")
			if err != nil {
				log.Error(p.ctx, "Scanner: Error persisting artist to DB", "folder", entry.path, "artist", entry.artists[i], err)
				return err
			}
		}

		// Save all new/modified albums to DB. Their information will be incomplete, but they will be refreshed later
		for i := range entry.albums {
			err := tx.Album(p.ctx).Put(&entry.albums[i])
			if err != nil {
				log.Error(p.ctx, "Scanner: Error persisting album to DB", "folder", entry.path, "album", entry.albums[i], err)
				return err
			}
		}

		// Save all tracks to DB
		for i := range entry.tracks {
			err = tx.MediaFile(p.ctx).Put(&entry.tracks[i])
			if err != nil {
				log.Error(p.ctx, "Scanner: Error persisting mediafile to DB", "folder", entry.path, "track", entry.tracks[i], err)
				return err
			}
		}

		// Mark all missing tracks as not available
		if len(entry.missingTracks) > 0 {
			err = tx.MediaFile(p.ctx).MarkMissing(true, entry.missingTracks...)
			if err != nil {
				log.Error(p.ctx, "Scanner: Error marking missing tracks", "folder", entry.path, err)
				return err
			}

			// Touch all albums that have missing tracks, so they get refreshed in later phases
			groupedMissingTracks := slice.ToMap(entry.missingTracks, func(mf model.MediaFile) (string, struct{}) { return mf.AlbumID, struct{}{} })
			albumsToUpdate := maps.Keys(groupedMissingTracks)
			err = tx.Album(p.ctx).Touch(albumsToUpdate...)
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
	log.Debug(p.ctx, "Scanner: Completed processing folder", " path", entry.path,
		"audioCount", len(entry.audioFiles), "imageCount", len(entry.imageFiles), "plsCount", len(entry.playlists),
		"elapsed", time.Since(entry.startTime), "tracksMissing", len(entry.missingTracks),
		"tracksImported", len(entry.tracks))
	return entry, nil
}

func (p *phaseFolders) finalize(err error) error {
	errF := p.ds.WithTx(func(tx model.DataStore) error {
		for _, job := range p.jobs {
			if len(job.lastUpdates) == 0 {
				continue
			}
			folderIDs := maps.Keys(job.lastUpdates)
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
