package model

type Artist struct {
	Annotations

	ID              string `json:"id"          orm:"column(id)"`
	Name            string `json:"name"`
	AlbumCount      int    `json:"albumCount"`
	SongCount       int    `json:"songCount"`
	FullText        string `json:"fullText"`
	SortArtistName  string `json:"sortArtistName"`
	OrderArtistName string `json:"orderArtistName"`
	Size            int64  `json:"size"`
	MbzArtistID     string `json:"mbzArtistId" orm:"column(mbz_artist_id)"`
}

type Artists []Artist

type ArtistIndex struct {
	ID      string
	Artists Artists
}
type ArtistIndexes []ArtistIndex

type ArtistRepository interface {
	CountAll(options ...QueryOptions) (int64, error)
	Exists(id string) (bool, error)
	Put(m *Artist) error
	Get(id string) (*Artist, error)
	FindByName(name string) (*Artist, error)
	GetStarred(options ...QueryOptions) (Artists, error)
	Search(q string, offset int, size int) (Artists, error)
	Refresh(ids ...string) error
	GetIndex() (ArtistIndexes, error)
	AnnotatedRepository
}

func (a Artist) GetAnnotations() Annotations {
	return a.Annotations
}
