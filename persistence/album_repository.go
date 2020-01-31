package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type albumRepository struct {
	sqlRepository
}

func NewAlbumRepository(ctx context.Context, o orm.Ormer) model.AlbumRepository {
	r := &albumRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "media_file"
	return r
}

func (r *albumRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sel := r.selectAlbum(options...)
	return r.count(sel, options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"album_id": id}))
}

func (r *albumRepository) Put(a *model.Album) error {
	return nil
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	//select album_id as id, album as name, f.artist, f.album_artist, f.artist_id, f.compilation, f.genre,
	//	max(f.year) as year, sum(f.duration) as duration, max(f.updated_at) as updated_at,
	//	min(f.created_at) as created_at, count(*) as song_count, a.id as current_id, f.id as cover_art_id,
	//	f.path as cover_art_path, f.has_cover_art
	//	group by album_id
	return r.newSelectWithAnnotation(model.AlbumItemType, "album_id", options...).
		Columns("album_id as id", "album as name", "artist", "album_artist", "artist", "artist_id",
			"compilation", "genre", "id as cover_art_id", "path as cover_art_path", "has_cover_art",
			"max(year) as year", "sum(duration) as duration", "max(updated_at) as updated_at",
			"min(created_at) as created_at", "count(*) as song_count").GroupBy("album_id")
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	sq := r.selectAlbum().Where(Eq{"album_id": id})
	var res model.Album
	err := r.queryOne(sq, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *albumRepository) FindByArtist(artistId string) (model.Albums, error) {
	sq := r.selectAlbum().Where(Eq{"artist_id": artistId}).OrderBy("album")
	var res model.Albums
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	var res model.Albums
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *albumRepository) GetMap(ids []string) (map[string]model.Album, error) {
	return nil, nil
}

// TODO Keep order when paginating
func (r *albumRepository) GetRandom(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	switch r.ormer.Driver().Type() {
	case orm.DRMySQL:
		sq = sq.OrderBy("RAND()")
	default:
		sq = sq.OrderBy("RANDOM()")
	}
	sql, args, err := r.toSql(sq)
	if err != nil {
		return nil, err
	}
	var results model.Albums
	_, err = r.ormer.Raw(sql, args...).QueryRows(&results)
	return results, err
}

func (r *albumRepository) Refresh(ids ...string) error {
	return nil
}

func (r *albumRepository) PurgeEmpty() error {
	return nil
}

func (r *albumRepository) GetStarred(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...).Where("starred = true")
	var starred model.Albums
	err := r.queryAll(sq, &starred)
	return starred, err
}

func (r *albumRepository) Search(q string, offset int, size int) (model.Albums, error) {
	return nil, nil
}

func (r *albumRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *albumRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *albumRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *albumRepository) EntityName() string {
	return "album"
}

func (r *albumRepository) NewInstance() interface{} {
	return &model.Album{}
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ model.ResourceRepository = (*albumRepository)(nil)
