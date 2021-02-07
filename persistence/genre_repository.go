package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/model"
)

type genreRepository struct {
	sqlRepository
}

func NewGenreRepository(ctx context.Context, o orm.Ormer) model.GenreRepository {
	r := &genreRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "media_file"
	return r
}

func (r genreRepository) GetAll() (model.Genres, error) {
	sq := Select("genre as name", "count(distinct album_id) as album_count", "count(distinct id) as song_count").
		From("media_file").GroupBy("genre")
	res := model.Genres{}
	err := r.queryAll(sq, &res)
	return res, err
}
