package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/pocketbase/dbx"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type genreRepository struct {
	sqlRepository
	sqlRestful
}

func NewGenreRepository(ctx context.Context, db dbx.Builder) model.GenreRepository {
	r := &genreRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "genre"
	r.filterMappings = map[string]filterFunc{
		"name": containsFilter("name"),
	}
	return r
}

func (r *genreRepository) GetAll(opt ...model.QueryOptions) (model.Genres, error) {
	sq := r.newSelect(opt...).Columns(
		"genre.id",
		"genre.name",
		"coalesce(a.album_count, 0) as album_count",
		"coalesce(m.song_count, 0) as song_count",
	).
		LeftJoin("(select ag.genre_id, count(ag.album_id) as album_count from album_genres ag group by ag.genre_id) a on a.genre_id = genre.id").
		LeftJoin("(select mg.genre_id, count(mg.media_file_id) as song_count from media_file_genres mg group by mg.genre_id) m on m.genre_id = genre.id")
	res := model.Genres{}
	err := r.queryAll(sq, &res)
	return res, err
}

// Put is an Upsert operation, based on the name of the genre: If the name already exists, returns its ID, or else
// insert the new genre in the DB and returns its new created ID.
func (r *genreRepository) Put(m *model.Genre) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	sql := Insert("genre").Columns("id", "name").Values(m.ID, m.Name).
		Suffix("on conflict (name) do update set name=excluded.name returning id")
	resp := model.Genre{}
	err := r.queryOne(sql, &resp)
	if err != nil {
		return err
	}
	m.ID = resp.ID
	return nil
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
