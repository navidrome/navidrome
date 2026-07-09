package model

import (
	"fmt"
	"iter"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model/id"
)

// Folder represents a folder in the library. Its path is relative to the library root.
// ALWAYS use NewFolder to create a new instance.
type Folder struct {
	ID              string       `structs:"id" db:"id" json:"id"`
	LibraryID       int          `structs:"library_id" db:"library_id" json:"libraryId"`
	LibraryPath     string       `structs:"-" json:"-" db:"-" hash:"ignore"`
	Path            string       `structs:"path" db:"path" json:"path"`
	Name            string       `structs:"name" db:"name" json:"name"`
	ParentID        string       `structs:"parent_id" db:"parent_id" json:"parentId"`
	NumAudioFiles   int          `structs:"num_audio_files" db:"num_audio_files" json:"numAudioFiles"`
	NumPlaylists    int          `structs:"num_playlists" db:"num_playlists" json:"numPlaylists"`
	ImageFiles      []string     `structs:"-" db:"-" json:"imageFiles"`
	ImagesUpdatedAt time.Time    `structs:"images_updated_at" db:"images_updated_at" json:"imagesUpdatedAt"`
	Hash            string       `structs:"hash" db:"hash" json:"hash"`
	Missing         bool         `structs:"missing" db:"missing" json:"missing"`
	UpdatedAt       time.Time    `structs:"updated_at" db:"updated_at" json:"updatedAt"`
	CreatedAt       time.Time    `structs:"created_at" db:"created_at" json:"createdAt"`
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

func FolderID(lib Library, folderPath string) string {
	// 1. Normalize all slashes to /
	folderPath = strings.ReplaceAll(folderPath, "\\", "/")
	libPath := strings.ReplaceAll(lib.Path, "\\", "/")

	for strings.Contains(folderPath, "//") {
		folderPath = strings.ReplaceAll(folderPath, "//", "/")
	}
	for strings.Contains(libPath, "//") {
		libPath = strings.ReplaceAll(libPath, "//", "/")
	}

	// 2. Remove library prefix if present
	folderPath = strings.TrimPrefix(folderPath, libPath)

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
	dir = path.Clean(dir)

	var parentID string
	if folderPath == "" {
		dir = ""
		name = "."
		parentID = "" // Root folder
	} else {
		parentID = FolderID(lib, dir)
	}

	f := &Folder{
		LibraryID:       lib.ID,
		ID:              newID,
		Path:            dir,
		Name:            name,
		ParentID:        parentID,
		ImageFiles:      []string{},
		UpdatedAt:       time.Now(),
		CreatedAt:       time.Now(),
		ImagesUpdatedAt: time.Time{},
	}
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
