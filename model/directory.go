package model

type DirectoryEntry struct {
	ID       string `structs:"id" json:"id" orm:"pk;column(id)"`
	Name     string `structs:"name" json:"name"`
	Path     string `structs:"path" json:"path"`
	ParentId string `structs:"parent_id" json:"parentId"`
}

type DirectoryEntries []DirectoryEntry

type DirectoryEntryOrFile struct {
	MediaFile  `structs:"-"`
	FolderId   string `structs:"folder_id" json:"folderId"`
	FolderName string `structs:"folder_name" json:"folderName"`
	ParentId   string `structs:"parent_id" json:"parentId"`
	IsDir      bool   `structs:"is_dir" json:"isDir"`
}

type DirectoryEntiesOrFiles []DirectoryEntryOrFile
type DirectoryEntryRepository interface {
	BrowserDirectory(id string) (DirectoryEntiesOrFiles, error)
	Delete(id string) error
	Get(id string) (*DirectoryEntry, error)
	GetDbRoot() (DirectoryEntries, error)
	GetAllDirectories() (DirectoryEntries, error)
	Put(*DirectoryEntry) error
}
