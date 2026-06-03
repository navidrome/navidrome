package model

import (
	"context"
	"fmt"
	"iter"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model/id"
)

// Folder represents a folder in the library. Its path is relative to the library root.
// ALWAYS use NewFolder to create a new instance.
type Folder struct {
	ID              string    `structs:"id" db:"id" json:"id"`
	LibraryID       int       `structs:"library_id" db:"library_id" json:"libraryId"`
	LibraryPath     string    `structs:"-" json:"-" db:"-" hash:"ignore"`
	Path            string    `structs:"path" db:"path" json:"path"`
	Name            string    `structs:"name" db:"name" json:"name"`
	ParentID        string    `structs:"parent_id" db:"parent_id" json:"parentId"`
	NumAudioFiles   int       `structs:"num_audio_files" db:"num_audio_files" json:"numAudioFiles"`
	NumPlaylists    int       `structs:"num_playlists" db:"num_playlists" json:"numPlaylists"`
	ImageFiles      []string  `structs:"-" db:"-" json:"imageFiles"`
	ImagesUpdatedAt time.Time `structs:"-" db:"-" json:"imagesUpdatedAt"`
	Hash            string    `structs:"-" db:"-" json:"hash"`
	Missing         bool      `structs:"missing" db:"missing" json:"missing"`
	UpdatedAt       time.Time `structs:"updated_at" db:"updated_at" json:"updatedAt"`
	CreatedAt       time.Time `structs:"created_at" db:"created_at" json:"createdAt"`
	Breadcrumbs     []Breadcrumb `structs:"-" json:"breadcrumbs" db:"-"`
}

type Breadcrumb struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (f Folder) AbsolutePath() string {
	return filepath.Join(f.LibraryPath, f.Path, f.Name)
}

func (f Folder) String() string {
	return f.AbsolutePath()
}

func (f Folder) CoverArtID() ArtworkID {
	return artworkIDFromFolder(f)
}

// FolderID generates a unique ID for a folder in a library.
// The ID is generated based on the library ID and the folder path relative to the library root.
// Any leading or trailing slashes are removed from the folder path.
func FolderID(lib Library, folderPath string) string {
	// 1. Normalize all slashes to /
	folderPath = strings.ReplaceAll(folderPath, "\\", "/")

	// 2. Remove library prefix if present
	folderPath = strings.TrimPrefix(folderPath, lib.Path)

	// 3. Clean and trim slashes ONLY (preserving dots in names)
	folderPath = path.Clean(folderPath)
	folderPath = strings.Trim(folderPath, "/")

	// 4. Root folder should always be ""
	if folderPath == "." {
		folderPath = ""
	}

	key := fmt.Sprintf("%d:%s", lib.ID, folderPath)
	return id.NewHash(key)
}

func NewFolder(lib Library, folderPath string) *Folder {
	// Normalize path immediately
	folderPath = strings.ReplaceAll(folderPath, "\\", "/")
	folderPath = path.Clean(folderPath)
	folderPath = strings.Trim(folderPath, "/")
	if folderPath == "." {
		folderPath = ""
	}

	newID := FolderID(lib, folderPath)
	dir, name := path.Split(folderPath)
	dir = strings.Trim(dir, "/")

	var parentID string
	if folderPath == "" {
		dir = ""
		parentID = "" // Root folder
	} else if dir == "" || dir == "." {
		dir = ""
		parentID = strconv.Itoa(lib.ID) // Top-level
	} else {
		parentID = FolderID(lib, dir)
	}
	f := &Folder{
		LibraryID:       lib.ID,
		ID:              newID,
		Path:            folderPath,
		Name:            name,
		ParentID:        parentID,
		ImageFiles:      []string{},
		UpdatedAt:       time.Now(),
		CreatedAt:       time.Now(),
		ImagesUpdatedAt: time.Time{},
	}
	log.Error(context.Background(), "!!!CRITICAL_DEBUG!!! NewFolder created", "id", f.ID, "path", f.Path, "parentId", f.ParentID)
	return f
}

type Folders []Folder

type FolderCursor iter.Seq2[Folder, error]

type FolderUpdateInfo struct {
	UpdatedAt time.Time
	Hash      string
}

type FolderRepository interface {
	Get(id string) (*Folder, error)
	GetByPath(lib Library, path string) (*Folder, error)
	GetAll(...QueryOptions) (Folders, error)
	CountAll(...QueryOptions) (int64, error)
	GetFolderUpdateInfo(lib Library, targetPaths ...string) (map[string]FolderUpdateInfo, error)
	Put(*Folder) error
	MarkMissing(missing bool, ids ...string) error
	GetTouchedWithPlaylists() (FolderCursor, error)
}
