package persistence

import (
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
)

type Album struct {
	ID           string    `orm:"pk;column(id)"`
	Name         string    `orm:"index"`
	ArtistID     string    `orm:"column(artist_id);index"`
	CoverArtPath string    ``
	CoverArtId   string    ``
	Artist       string    `orm:"index"`
	AlbumArtist  string    ``
	Year         int       `orm:"index"`
	Compilation  bool      ``
	Starred      bool      `orm:"index"`
	PlayCount    int       `orm:"index"`
	PlayDate     time.Time `orm:"null;index"`
	SongCount    int       ``
	Duration     int       ``
	Rating       int       `orm:"index"`
	Genre        string    ``
	StarredAt    time.Time `orm:"null"`
	CreatedAt    time.Time `orm:"null"`
	UpdatedAt    time.Time `orm:"null"`
}

type albumRepository struct {
	searchableRepository
}

func NewAlbumRepository() model.AlbumRepository {
	r := &albumRepository{}
	r.tableName = "album"
	return r
}

func (r *albumRepository) Put(a *model.Album) error {
	ta := Album(*a)
	return withTx(func(o orm.Ormer) error {
		return r.put(o, a.ID, a.Name, &ta)
	})
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	ta := Album{ID: id}
	err := Db().Read(&ta)
	if err == orm.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := model.Album(ta)
	return &a, err
}

func (r *albumRepository) FindByArtist(artistId string) (model.Albums, error) {
	var albums []Album
	_, err := r.newQuery(Db()).Filter("artist_id", artistId).OrderBy("year", "name").All(&albums)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(albums), nil
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	var all []Album
	_, err := r.newQuery(Db(), options...).All(&all)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(all), nil
}

func (r *albumRepository) toAlbums(all []Album) model.Albums {
	result := make(model.Albums, len(all))
	for i, a := range all {
		result[i] = model.Album(a)
	}
	return result
}

// TODO Remove []string from return
func (r *albumRepository) PurgeInactive(activeList model.Albums) error {
	return withTx(func(o orm.Ormer) error {
		_, err := r.purgeInactive(o, activeList, func(item interface{}) string {
			return item.(model.Album).ID
		})
		return err
	})
}

func (r *albumRepository) GetStarred(options ...model.QueryOptions) (model.Albums, error) {
	var starred []Album
	_, err := r.newQuery(Db(), options...).Filter("starred", true).All(&starred)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(starred), nil
}

func (r *albumRepository) Search(q string, offset int, size int) (model.Albums, error) {
	if len(q) <= 2 {
		return nil, nil
	}

	var results []Album
	err := r.doSearch(r.tableName, q, offset, size, &results, "rating desc", "starred desc", "play_count desc", "name")
	if err != nil {
		return nil, err
	}
	return r.toAlbums(results), nil
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ = model.Album(Album{})
