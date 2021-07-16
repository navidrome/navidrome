package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"

	"github.com/deluan/rest"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/model"
)

type genreRepository struct {
	sqlRepository
	sqlRestful
}

func NewGenreRepository(ctx context.Context, o orm.Ormer) model.GenreRepository {
	r := &genreRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "genre"
	return r
}

func (r *genreRepository) GetAll() (model.Genres, error) {
	sq := Select("*",
		"count(distinct a.album_id) as album_count",
		"count(distinct f.media_file_id) as song_count").
		From(r.tableName).
		LeftJoin("album_genres a on a.genre_id = genre.id").
		LeftJoin("media_file_genres f on f.genre_id = genre.id").
		GroupBy("genre.id")
	res := model.Genres{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *genreRepository) Put(m *model.Genre) error {
	id, err := r.put(m.ID, m)
	m.ID = id
	return err
}

func (r *genreRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select(), r.parseRestOptions(options...))
}

func (r *genreRepository) Read(id string) (interface{}, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Genre
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *genreRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sel := r.newSelect(r.parseRestOptions(options...)).Columns("*")
	res := model.Genres{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *genreRepository) EntityName() string {
	return r.tableName
}

func (r *genreRepository) NewInstance() interface{} {
	return &model.Genre{}
}

func (r *genreRepository) purgeEmpty() error {
	del := Delete(r.tableName).Where(`id in (
select genre.id from genre
left join album_genres ag on genre.id = ag.genre_id
left join artist_genres a on genre.id = a.genre_id
left join media_file_genres mfg on genre.id = mfg.genre_id
where ag.genre_id is null
and a.genre_id is null
and mfg.genre_id is null
)`)
	c, err := r.executeSQL(del)
	if err == nil {
		if c > 0 {
			log.Debug(r.ctx, "Purged unused genres", "totalDeleted", c)
		}
	}
	return err
}

var _ model.GenreRepository = (*genreRepository)(nil)
var _ model.ResourceRepository = (*genreRepository)(nil)
