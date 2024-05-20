package scanner2

import (
	"io/fs"
	"time"
)

type folderEntry struct {
	job             *scanJob
	startTime       time.Time
	path            string    // Full path
	id              string    // DB ID
	updTime         time.Time // From DB
	modTime         time.Time // From FS
	audioFiles      map[string]fs.DirEntry
	imageFiles      map[string]fs.DirEntry
	playlists       []fs.DirEntry
	imagesUpdatedAt time.Time
	//tracks          model.MediaFiles
	//albums          model.Albums
	//artists         model.Artists
	//tags            model.FlattenedTags
	//missingTracks   model.MediaFiles
}

func (f *folderEntry) isOutdated() bool {
	return f.updTime.Before(f.modTime)
}
