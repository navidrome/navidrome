package model

type MediaFolder struct {
	ID   string
	Name string
	Path string
}

type MediaFolders []MediaFolder

type MediaFolderRepository interface {
	Get(id string) (*MediaFolder, error)
	GetAll() (MediaFolders, error)
}
