package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
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
	sql, args, err := r.toSql(sq)
	if err != nil {
		return nil, err
	}
	var res model.Genres
	_, err = r.ormer.Raw(sql, args).QueryRows(&res)
	return res, err
}
