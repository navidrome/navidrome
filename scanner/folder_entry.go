package scanner

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"slices"
	"time"

	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/chrono"
)

func newFolderEntry(job *scanJob, id, path string, info model.FolderUpdateInfo) *folderEntry {
	f := &folderEntry{
		id:                  id,
		job:                 job,
		path:                path,
		audioFiles:          make(map[string]fs.DirEntry),
		imageFiles:          make(map[string]fs.DirEntry),
		playlistFiles:       make(map[string]fs.DirEntry),
		albumIDMap:          make(map[string]string),
		updTime:             info.UpdatedAt,
		prevHash:            info.Hash,
		prevImageFiles:      info.ImageFiles,
		prevImagesUpdatedAt: info.ImagesUpdatedAt,
	}
	return f
}

type folderEntry struct {
	job             *scanJob
	elapsed         chrono.Meter
	path            string    // Full path
	id              string    // DB ID
	modTime         time.Time // From FS
	updTime         time.Time // from DB
	audioFiles      map[string]fs.DirEntry
	imageFiles      map[string]fs.DirEntry
	playlistFiles   map[string]fs.DirEntry
	numSubFolders   int
	imagesUpdatedAt time.Time
	prevHash        string // Previous hash from DB
	// Previous image state from DB, to detect image-only changes
	prevImageFiles      []string
	prevImagesUpdatedAt time.Time
	tracks              model.MediaFiles
	albums              model.Albums
	albumIDMap          map[string]string
	artists             model.Artists
	tags                model.TagList
	missingTracks       []*model.MediaFile
}

func (f *folderEntry) hasNoFiles() bool {
	return len(f.audioFiles) == 0 && len(f.imageFiles) == 0 && len(f.playlistFiles) == 0
}

func (f *folderEntry) isEmpty() bool {
	return f.hasNoFiles() && f.numSubFolders == 0
}

func (f *folderEntry) isNew() bool {
	return f.updTime.IsZero()
}

func (f *folderEntry) isOutdated() bool {
	if f.job.lib.FullScanInProgress && f.updTime.Before(f.job.lib.LastScanStartedAt) {
		return true
	}
	return f.prevHash != f.hash()
}

// imagesChanged reports whether the folder's image files differ from the previously persisted
// state, and whether an artist image is involved (present in the old or the new list).
func (f *folderEntry) imagesChanged() (changed, artistImage bool) {
	newNames := slices.Sorted(maps.Keys(f.imageFiles))
	prevNames := slices.Sorted(slices.Values(f.prevImageFiles))
	// Both empty also skips the timestamp check, which is noise for image-less folders.
	if len(prevNames) == 0 && len(newNames) == 0 {
		return false, false
	}
	if slices.Equal(prevNames, newNames) && f.prevImagesUpdatedAt.Equal(f.imagesUpdatedAt) {
		return false, false
	}
	return true, slices.ContainsFunc(slices.Concat(prevNames, newNames), artwork.IsArtistImageFile)
}

func (f *folderEntry) toFolder() *model.Folder {
	folder := model.NewFolder(f.job.lib, f.path)
	folder.NumAudioFiles = len(f.audioFiles)
	if playlists.InPath(*folder) {
		folder.NumPlaylists = len(f.playlistFiles)
	}
	folder.ImageFiles = slices.Collect(maps.Keys(f.imageFiles))
	folder.ImagesUpdatedAt = f.imagesUpdatedAt
	folder.Hash = f.hash()
	return folder
}

func (f *folderEntry) hash() string {
	h := md5.New()
	_, _ = fmt.Fprintf(
		h,
		"%s:%d:%d:%s",
		f.modTime.UTC(),
		len(f.playlistFiles),
		f.numSubFolders,
		f.imagesUpdatedAt.UTC(),
	)

	// Sort the keys of audio, image and playlist files to ensure consistent hashing
	audioKeys := slices.Collect(maps.Keys(f.audioFiles))
	slices.Sort(audioKeys)
	imageKeys := slices.Collect(maps.Keys(f.imageFiles))
	slices.Sort(imageKeys)
	playlistKeys := slices.Collect(maps.Keys(f.playlistFiles))
	slices.Sort(playlistKeys)

	// Include audio files with their size and modtime
	for _, key := range audioKeys {
		_, _ = io.WriteString(h, key)
		if info, err := f.audioFiles[key].Info(); err == nil {
			_, _ = fmt.Fprintf(h, ":%d:%s", info.Size(), info.ModTime().UTC().String())
		}
	}

	// Include image files with their size and modtime
	for _, key := range imageKeys {
		_, _ = io.WriteString(h, key)
		if info, err := f.imageFiles[key].Info(); err == nil {
			_, _ = fmt.Fprintf(h, ":%d:%s", info.Size(), info.ModTime().UTC().String())
		}
	}

	// Include playlist files with their size and modtime, so an in-place edit
	// (which changes at least the size) is detected even when the containing
	// folder's mtime is preserved (rsync -a) or is not the newest in the folder.
	for _, key := range playlistKeys {
		_, _ = io.WriteString(h, key)
		if info, err := f.playlistFiles[key].Info(); err == nil {
			_, _ = fmt.Fprintf(h, ":%d:%s", info.Size(), info.ModTime().UTC().String())
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}
