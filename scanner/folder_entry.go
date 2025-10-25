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

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/chrono"
)

func newFolderEntry(job *scanJob, path string) *folderEntry {
	id := model.FolderID(job.lib, path)
	info := job.popLastUpdate(id)
	f := &folderEntry{
		id:         id,
		job:        job,
		path:       path,
		audioFiles: make(map[string]fs.DirEntry),
		imageFiles: make(map[string]fs.DirEntry),
		albumIDMap: make(map[string]string),
		updTime:    info.UpdatedAt,
		prevHash:   info.Hash,
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
	numPlaylists    int
	numSubFolders   int
	imagesUpdatedAt time.Time
	prevHash        string // Previous hash from DB
	tracks          model.MediaFiles
	albums          model.Albums
	albumIDMap      map[string]string
	artists         model.Artists
	tags            model.TagList
	missingTracks   []*model.MediaFile
}

func (f *folderEntry) hasNoFiles() bool {
	return len(f.audioFiles) == 0 && len(f.imageFiles) == 0 && f.numPlaylists == 0
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

func (f *folderEntry) toFolder() *model.Folder {
	folder := model.NewFolder(f.job.lib, f.path)
	folder.NumAudioFiles = len(f.audioFiles)
	if core.InPlaylistsPath(*folder) {
		folder.NumPlaylists = f.numPlaylists
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
		f.numPlaylists,
		f.numSubFolders,
		f.imagesUpdatedAt.UTC(),
	)

	// Sort the keys of audio and image files to ensure consistent hashing
	audioKeys := slices.Collect(maps.Keys(f.audioFiles))
	slices.Sort(audioKeys)
	imageKeys := slices.Collect(maps.Keys(f.imageFiles))
	slices.Sort(imageKeys)

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

	return hex.EncodeToString(h.Sum(nil))
}
