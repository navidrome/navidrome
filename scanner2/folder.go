package scanner2

import (
	"io/fs"
	"time"

	"github.com/navidrome/navidrome/model"
)

type folderEntry struct {
	job             *scanJob
	startTime       time.Time
	path            string    // Full path
	id              string    // DB ID
	modTime         time.Time // From FS
	audioFiles      map[string]fs.DirEntry
	imageFiles      map[string]fs.DirEntry
	playlists       []fs.DirEntry
	imagesUpdatedAt time.Time
	tracks          model.MediaFiles
	albums          model.Albums
	artists         model.Artists
	tags            model.TagList
	missingTracks   model.MediaFiles
}

func newFolderEntry(job *scanJob, path string) *folderEntry {
	return &folderEntry{
		id:         model.FolderID(job.lib, path),
		job:        job,
		path:       path,
		startTime:  time.Now(),
		audioFiles: make(map[string]fs.DirEntry),
		imageFiles: make(map[string]fs.DirEntry),
	}
}

func (f *folderEntry) updatedTime() time.Time {
	return f.job.lastUpdates[f.id]
}

func (f *folderEntry) isOutdated() bool {
	return f.updatedTime().Before(f.modTime)
}
