package model

type MediaFolder struct {
	ID   int32
	Name string
	Path string
}

type MediaFolders []MediaFolder

type MediaFolderRepository interface {
	Get(id string) (*MediaFolder, error)
	GetAll() (MediaFolders, error)
}
