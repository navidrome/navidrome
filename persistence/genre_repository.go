package persistence

import (
	"context"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

type genreRepository struct {
	sqlRepository
	sqlRestful
}

func NewGenreRepository(ctx context.Context, o orm.Ormer) model.GenreRepository {
	r := &genreRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "media_file"
	return r
}

func (r *genreRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select(), r.parseRestOptions(options...))
}

func (r *genreRepository) EntityName() string {
	return "genre"
}

func (r *genreRepository) NewInstance() interface{} {
	return &model.Genre{}
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

func (r *genreRepository) GetAll() (model.Genres, error) {
	sq := Select("genre as name", "count(distinct album_id) as album_count", "count(distinct id) as song_count").
		From("media_file").GroupBy("genre")
	res := model.Genres{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *genreRepository) Refresh(ids ...string) error {
	chunks := utils.BreakUpStringSlice(ids, 100)
	for _, chunk := range chunks {
		err := r.refresh(chunk...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *genreRepository) refresh(cids ...string) error {
	var ids []string
	for _, id := range cids {
		ids = append(ids, strings.Split(id, ",")...)
	}
	type refreshGenre struct {
		model.Genre
		CurrentId string
	}
	var genres []refreshGenre
	sel := Select(`f.genre as name, count(distinct f.album_id) as album_count, count(distinct f.id) as song_count,
		g.name as current_id`).
		From("media_file f").
		LeftJoin("genre g on f.genre = g.name").
		Where(Eq{"f.genre": ids}).GroupBy("f.genre")
	err := r.queryAll(sel, &genres)
	if err != nil {
		return err
	}

	toInsert := 0
	toUpdate := 0

	for _, g := range genres {
		if g.CurrentId != "" {
			toUpdate++
		} else {
			toInsert++
		}

		_, err := r.put(g.Name, g.Genre)
		if err != nil {
			return err
		}
	}
	if toInsert > 0 {
		log.Debug(r.ctx, "Inserted new genres", "totalInserted", toInsert)
	}
	if toUpdate > 0 {
		log.Debug(r.ctx, "Updated genres", "totalUpdated", toUpdate)
	}
	return err
}

var _ model.GenreRepository = (*genreRepository)(nil)
