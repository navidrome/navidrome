package models

type Album struct {
	Id           string
	Name         string
	ArtistId     string
	CoverArtPath string
	Year         int
	Compilation  bool
	Rating       int
	MediaFiles   map[string]bool
}

func (a *Album) AdMediaFiles(mfs ...*MediaFile) {
	for _, mf := range mfs {
		a.MediaFiles[mf.Id] = true
	}
}

func (a *Album) AddMediaFiles(mfs  ...interface{}) {
	if a.MediaFiles == nil {
		a.MediaFiles = make(map[string]bool)
	}
	for _, v := range mfs {
		switch v := v.(type) {
		case *MediaFile:
			a.MediaFiles[v.Id] = true
		case map[string]bool:
			for k, _ := range v {
				a.MediaFiles[k] = true
			}
		}
	}
}