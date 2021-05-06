package persistence

import (
	"context"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

type genreTypeRepository struct {
	sqlRepository
}

func NewGenreTypeRepository(ctx context.Context, o orm.Ormer) model.GenreTypeRepository {
	r := &genreTypeRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "genre_type"
	return r
}

func (r *genreTypeRepository) GetGenres(itemID string, itemType string) ([]string, error) {
	all := r.newSelect().Columns("genre_id").Where(And{Eq{"item_id": itemID}, Eq{"item_type": itemType}}).OrderBy("genre_id")
	var genres model.GenreTypes
	err := r.queryAll(all, &genres)
	if err != nil {
		log.Error("Error querying genres from", itemType, "ID", itemID, err)
		return nil, err
	}
	genre_ids := make([]string, len(genres))
	for i := range genre_ids {
		genre_ids[i] = genres[i].GenreID
	}
	return genre_ids, nil
}

func (r *genreTypeRepository) EntityName() string {
	return "genre_type"
}

func (r *genreTypeRepository) NewInstance() interface{} {
	return &model.GenreType{}
}

func (r *genreTypeRepository) Refresh(ids ...string) error {
	chunks := utils.BreakUpStringSlice(ids, 100)
	for _, chunk := range chunks {
		err := r.refresh(chunk...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *genreTypeRepository) refresh(cids ...string) error {
	var ids []string
	for _, id := range cids {
		ids = append(ids, strings.Split(id, ",")...)
	}
	del := Delete(r.tableName).Where(Eq{"genre_id": ids})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}

	sel := Select(`genre as genre_id, id as item_id`).
		From("media_file").
		Where(Eq{"genre": ids})
	err = r.insertRelations(sel, "media_file")
	if err != nil {
		return err
	}

	sel = Select(`f.genre as genre_id, a.id as item_id`).
		From("media_file as f").
		Distinct().
		LeftJoin("album a on f.album_id = a.id").
		Where(Eq{"f.genre": ids})
	err = r.insertRelations(sel, "album")
	if err != nil {
		return err
	}

	sel = Select(`f.genre as genre_id, a.id as item_id`).
		From("media_file as f").
		Distinct().
		LeftJoin("artist a on f.artist_id = a.id").
		Where(Eq{"f.genre": ids})
	err = r.insertRelations(sel, "artist")
	if err != nil {
		return err
	}

	return nil
}

func (r *genreTypeRepository) insertRelations(sel SelectBuilder, itemType string) error {
	var genres model.GenreTypes
	err := r.queryAll(sel, &genres)
	if err != nil {
		return err
	}

	for i := 0; i < len(genres); {
		ins := Insert(r.tableName).Columns("genre_id", "item_id", "item_type")
		for j := 0; j < 50 && i < len(genres); i, j = i+1, j+1 {
			genres[i].ItemType = itemType
			ins = ins.Values(genres[i].GenreID, genres[i].ItemID, genres[i].ItemType)
		}
		_, err = r.executeSQL(ins)
		if err != nil {
			return err
		}
	}
	return nil
}

var _ model.GenreTypeRepository = (*genreTypeRepository)(nil)
