package model

import (
	"fmt"
	"iter"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model/id"
)

// Folder represents a folder in the library. Its path is relative to the library root.
// ALWAYS use NewFolder to create a new instance.
type Folder struct {
	ID              string    `structs:"id"`
	LibraryID       int       `structs:"library_id"`
	LibraryPath     string    `structs:"-" json:"-" hash:"ignore"`
	Path            string    `structs:"path"`
	Name            string    `structs:"name"`
	ParentID        string    `structs:"parent_id"`
	NumAudioFiles   int       `structs:"num_audio_files"`
	NumPlaylists    int       `structs:"num_playlists"`
	ImageFiles      []string  `structs:"image_files"`
	ImagesUpdatedAt time.Time `structs:"images_updated_at"`
	Hash            string    `structs:"hash"`
	Missing         bool      `structs:"missing"`
	UpdateAt        time.Time `structs:"updated_at"`
	CreatedAt       time.Time `structs:"created_at"`
}

func (f Folder) AbsolutePath() string {
	return filepath.Join(f.LibraryPath, f.Path, f.Name)
}

func (f Folder) String() string {
	return f.AbsolutePath()
}

// FolderID generates a unique ID for a folder in a library.
// The ID is generated based on the library ID and the folder path relative to the library root.
// Any leading or trailing slashes are removed from the folder path.
func FolderID(lib Library, path string) string {
	path = strings.TrimPrefix(path, lib.Path)
	path = strings.TrimPrefix(path, string(os.PathSeparator))
	path = filepath.Clean(path)
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

type FolderCursor iter.Seq2[Folder, error]

type FolderUpdateInfo struct {
	UpdatedAt time.Time
	Hash      string
}

type FolderRepository interface {
	Get(id string) (*Folder, error)
	GetByPath(lib Library, path string) (*Folder, error)
	GetAll(...QueryOptions) ([]Folder, error)
	CountAll(...QueryOptions) (int64, error)
	GetLastUpdates(lib Library) (map[string]FolderUpdateInfo, error)
	Put(*Folder) error
	MarkMissing(missing bool, ids ...string) error
	GetTouchedWithPlaylists() (FolderCursor, error)
}
