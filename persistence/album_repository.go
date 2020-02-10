package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
	"github.com/dhowden/tag/mbz"
)

type albumRepository struct {
	sqlRepository
}

func NewAlbumRepository(ctx context.Context, o orm.Ormer) model.AlbumRepository {
	r := &albumRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "album"
	return r
}

func (r *albumRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(Select(), options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"id": id}))
}

func (r *albumRepository) Put(a *model.Album) error {
	_, err := r.put(a.ID, a)
	if err != nil {
		return err
	}
	return r.index(a.ID, a.Name, a.Artist, mbz.AlbumArtist)
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	return r.newSelectWithAnnotation("id", options...).Columns("*")
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	sq := r.selectAlbum().Where(Eq{"id": id})
	var res model.Album
	err := r.queryOne(sq, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *albumRepository) FindByArtist(artistId string) (model.Albums, error) {
	sq := r.selectAlbum().Where(Eq{"artist_id": artistId}).OrderBy("year")
	res := model.Albums{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	res := model.Albums{}
	err := r.queryAll(sq, &res)
	return res, err
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
	results := model.Albums{}
	err := r.queryAll(sq, &results)
	return results, err
}

func (r *albumRepository) Refresh(ids ...string) error {
	type refreshAlbum struct {
		model.Album
		CurrentId   string
		HasCoverArt bool
	}
	var albums []refreshAlbum
	sel := Select(`album_id as id, album as name, f.artist, f.album_artist, f.artist_id, f.compilation, f.genre,
		max(f.year) as year, sum(f.duration) as duration, count(*) as song_count, a.id as current_id, 
		f.id as cover_art_id, f.path as cover_art_path, f.has_cover_art`).
		From("media_file f").
		LeftJoin("album a on f.album_id = a.id").
		Where(Eq{"f.album_id": ids}).GroupBy("album_id").OrderBy("f.id")
	err := r.queryAll(sel, &albums)
	if err != nil {
		return err
	}

	toInsert := 0
	toUpdate := 0
	for _, al := range albums {
		if !al.HasCoverArt {
			al.CoverArtId = ""
		}
		if al.Compilation {
			al.AlbumArtist = "Various Artists"
		}
		if al.AlbumArtist == "" {
			al.AlbumArtist = al.Artist
		}
		al.UpdatedAt = time.Now()
		if al.CurrentId != "" {
			toUpdate++
		} else {
			toInsert++
			al.CreatedAt = time.Now()
		}
		err := r.Put(&al.Album)
		if err != nil {
			return err
		}
	}
	if toInsert > 0 {
		log.Debug(r.ctx, "Inserted new albums", "totalInserted", toInsert)
	}
	if toUpdate > 0 {
		log.Debug(r.ctx, "Updated albums", "totalUpdated", toUpdate)
	}
	return err
}

func (r *albumRepository) PurgeEmpty() error {
	del := Delete(r.tableName).Where("id not in (select distinct(album_id) from media_file)")
	c, err := r.executeSQL(del)
	if err == nil {
		if c > 0 {
			log.Debug(r.ctx, "Purged empty albums", "totalDeleted", c)
		}
	}
	return err
}

func (r *albumRepository) GetStarred(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...).Where("starred = true")
	starred := model.Albums{}
	err := r.queryAll(sq, &starred)
	return starred, err
}

func (r *albumRepository) Search(q string, offset int, size int) (model.Albums, error) {
	results := model.Albums{}
	err := r.doSearch(q, offset, size, &results, "name")
	return results, err
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
