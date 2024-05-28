package scanner2

import (
	"context"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	"github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/exp/maps"
)

const (
	// filesBatchSize used for batching file metadata extraction
	filesBatchSize = 100
)

func processFolder(ctx context.Context) pipeline.StageFn[*folderEntry] {
	return func(entry *folderEntry) (*folderEntry, error) {
		// Load children mediafiles from DB
		mfs, err := entry.job.ds.MediaFile(ctx).GetAll(model.QueryOptions{
			Filters: squirrel.And{squirrel.Eq{"folder_id": entry.id}},
		})
		if err != nil {
			log.Error(ctx, "Scanner: Error loading mediafiles from DB", "folder", entry.path, err)
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
					log.Warn(ctx, "Scanner: Error getting file info", "folder", entry.path, "file", af.Name(), err)
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
			entry.tracks, entry.tags, err = loadTagsFromFiles(ctx, entry, filesToImport)
			if err != nil {
				log.Warn(ctx, "Scanner: Error loading tags from files. Skipping", "folder", entry.path, err)
				return entry, nil
			}

			entry.albums = loadAlbumsFromMediaFiles(ctx, entry)
			entry.artists = loadArtistsFromTags(ctx, entry)
		}

		return entry, nil
	}
}

func loadTagsFromFiles(ctx context.Context, entry *folderEntry, toImport []string) (model.MediaFiles, model.TagList, error) {
	tracks := model.MediaFiles{}
	uniqueTags := make(map[string]model.Tag)
	err := slice.RangeByChunks(toImport, filesBatchSize, func(chunk []string) error {
		allInfo, err := entry.job.fs.ReadTags(toImport...)
		if err != nil {
			log.Warn(ctx, "Scanner: Error extracting metadata from files. Skipping", "folder", entry.path, err)
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

func loadAlbumsFromMediaFiles(ctx context.Context, entry *folderEntry) model.Albums {
	grouped := slice.Group(entry.tracks, func(mf model.MediaFile) string { return mf.AlbumPID })
	var albums model.Albums
	for _, group := range grouped {
		songs := model.MediaFiles(group)
		album := songs.ToAlbum() // TODO Remove ID from ToAlbum
		album.FolderID = entry.id
		albums = append(albums, album)
	}
	return albums
}

func loadArtistsFromTags(ctx context.Context, entry *folderEntry) model.Artists {
	return nil // TODO
}
