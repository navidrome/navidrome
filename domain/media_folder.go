package domain

type MediaFolder struct {
	Id string
	Name string
	Path string
}

type MediaFolderRepository interface {
	GetAll() ([]MediaFolder, error)
}

