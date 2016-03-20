package domain

type MediaFolder struct {
	Id   string
	Name string
	Path string
}

type MediaFolders []MediaFolder

type MediaFolderRepository interface {
	GetAll() (MediaFolders, error)
}
