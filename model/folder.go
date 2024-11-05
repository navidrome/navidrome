package model

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model/id"
)

// Folder represents a folder in the library. Its path is relative to the library root.
// ALWAYS use NewFolder to create a new instance.
type Folder struct {
	ID              string    `structs:"id"`
	LibraryID       int       `structs:"library_id"`
	Path            string    `structs:"path"`
	Name            string    `structs:"name"`
	ParentID        string    `structs:"parent_id"`
	NumAudioFiles   int       `structs:"num_audio_files"`
	ImageFiles      []string  `structs:"image_files"`
	ImagesUpdatedAt time.Time `structs:"images_updated_at"`
	Missing         bool      `structs:"missing"`
	UpdateAt        time.Time `structs:"updated_at"`
	CreatedAt       time.Time `structs:"created_at"`
}

func FolderID(lib Library, path string) string {
	path = strings.TrimPrefix(path, lib.Path)
	key := fmt.Sprintf("%d:%s", lib.ID, path)
	return id.NewHash(key)
}
func NewFolder(lib Library, folderPath string) *Folder {
	newID := FolderID(lib, folderPath)
	dir, name := path.Split(folderPath)
	dir = path.Clean(dir)
	var parentID string
	if dir == "." && name == "." {
		dir = ""
		parentID = ""
	} else {
		parentID = FolderID(lib, dir)
	}
	return &Folder{
		LibraryID:  lib.ID,
		ID:         newID,
		Path:       dir,
		Name:       name,
		ParentID:   parentID,
		ImageFiles: []string{},
		UpdateAt:   time.Now(),
		CreatedAt:  time.Now(),
	}
}

type FolderRepository interface {
	Get(id string) (*Folder, error)
	GetByPath(lib Library, path string) (*Folder, error)
	GetAll(...QueryOptions) ([]Folder, error)
	GetLastUpdates(lib Library) (map[string]time.Time, error)
	Put(*Folder) error
	MarkMissing(missing bool, ids ...string) error
	Touch(lib Library, path string, t time.Time) error
}
