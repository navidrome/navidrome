package model

import (
	"crypto/md5"
	"fmt"
	"path"
	"strings"
	"time"
)

// Folder represents a folder in the library. Its path is relative to the library root.
type Folder struct {
	ID        string    `structs:"id"`
	LibraryID int       `structs:"library_id"`
	Path      string    `structs:"path"`
	Name      string    `structs:"name"`
	Missing   bool      `structs:"missing"`
	ParentID  string    `structs:"parent_id"`
	UpdateAt  time.Time `structs:"updated_at"`
	CreatedAt time.Time `structs:"created_at"`
}

func FolderID(lib Library, path string) string {
	path = strings.TrimPrefix(path, lib.Path)
	key := fmt.Sprintf("%d:%s", lib.ID, path)
	return fmt.Sprintf("%x", md5.Sum([]byte(key)))
}
func NewFolder(lib Library, folderPath string) *Folder {
	id := FolderID(lib, folderPath)
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
		LibraryID: lib.ID,
		ID:        id,
		Path:      dir,
		Name:      name,
		ParentID:  parentID,
		UpdateAt:  time.Now(),
		CreatedAt: time.Now(),
	}
}

type FolderRepository interface {
	Get(lib Library, path string) (*Folder, error)
	GetAll(lib Library) ([]Folder, error)
	GetLastUpdates(lib Library) (map[string]time.Time, error)
	Put(lib Library, path string) error
	MarkMissing(missing bool, ids ...string) error
	Touch(lib Library, path string, t time.Time) error
}
