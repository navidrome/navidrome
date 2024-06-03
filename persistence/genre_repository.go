package persistence

import (
	"context"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model/metadata"
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
	r.tableName = "tag"
	r.filterMappings = map[string]filterFunc{
		"name": containsFilter("name"),
	}
	return r
}

func (r *genreRepository) selectGenre(opt ...model.QueryOptions) SelectBuilder {
	sq := Select().From("tag").
		Columns(
			"tag.id",
			"tag.tag_value as name",
			"coalesce(a.album_count, 0) as album_count",
			"coalesce(m.song_count, 0) as song_count",
		).
		LeftJoin("(select it.tag_id, count(it.item_id) as album_count from item_tags it where item_type = 'album' group by it.tag_id) a on a.tag_id = tag.id").
		LeftJoin("(select it.tag_id, count(it.item_id) as song_count from item_tags it where item_type = 'media_file' group by it.tag_id) m on m.tag_id = tag.id").
		Where(Eq{"tag.tag_name": metadata.Genre})
	sq = r.applyOptions(sq, opt...)
	sq = r.applyFilters(sq, opt...)
	return sq
}

func (r *genreRepository) GetAll(opt ...model.QueryOptions) (model.Genres, error) {
	sq := r.selectGenre(opt...)
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
	return r.count(r.selectGenre(), r.parseRestOptions(options...))
}

func (r *genreRepository) Read(id string) (interface{}, error) {
	sel := r.selectGenre().Columns("*").Where(Eq{"id": id})
	var res model.Genre
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *genreRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
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
