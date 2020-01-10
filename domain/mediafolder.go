package domain

type MediaFolder struct {
	ID   string
	Name string
	Path string
}

type MediaFolders []MediaFolder

type MediaFolderRepository interface {
	GetAll() (MediaFolders, error)
}
