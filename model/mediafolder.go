package model

import (
	"io/fs"
	"os"
)

type MediaFolder struct {
	ID       string `structs:"id" json:"id" orm:"pk;column(id)"`
	Name     string `structs:"name" json:"name"`
	Path     string `structs:"path" json:"path"`
	ParentId string `structs:"parent_id" json:"parentId"`
}

func (f MediaFolder) FS() fs.FS {
	return os.DirFS(f.Path)
}

type MediaFolders []MediaFolder

type MediaFolderOrFile struct {
	MediaFile  `structs:"-"`
	FolderId   string `structs:"folder_id" json:"folderId"`
	FolderName string `structs:"folder_name" json:"folderName"`
	ParentId   string `structs:"parent_id" json:"parentId"`
	IsDir      bool   `structs:"is_dir" json:"isDir"`
}

type MediaFolderOrFiles []MediaFolderOrFile
type MediaFolderRepository interface {
	BrowserDirectory(id string) (MediaFolderOrFiles, error)
	Delete(id string) error
	Get(id string) (*MediaFolder, error)
	GetDbRoot() (MediaFolders, error)
	GetRoot() (MediaFolders, error)
	GetAllDirectories() (MediaFolders, error)
	Put(*MediaFolder) error
}
