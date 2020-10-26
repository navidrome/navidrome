package model

type MediaFolder struct {
	ID   int32
	Name string
	Path string
}

type MediaFolders []MediaFolder

type MediaFolderRepository interface {
	Get(id int32) (*MediaFolder, error)
	GetAll() (MediaFolders, error)
}
