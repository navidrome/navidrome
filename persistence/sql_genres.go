package persistence

import (
	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

func (r sqlRepository) withGenres(sql SelectBuilder) SelectBuilder {
	return sql.LeftJoin(r.tableName + "_genres ag on " + r.tableName + ".id = ag." + r.tableName + "_id").
		LeftJoin("genre on ag.genre_id = genre.id")
}

func (r *sqlRepository) updateGenres(id string, genres model.Genres) error {
	tableName := r.getTableName()
	del := Delete(tableName + "_genres").Where(Eq{tableName + "_id": id})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}

	if len(genres) == 0 {
		return nil
	}
	var genreIds []string
	for _, g := range genres {
		genreIds = append(genreIds, g.ID)
	}
	err = slice.RangeByChunks(genreIds, 100, func(ids []string) error {
		ins := Insert(tableName+"_genres").Columns("genre_id", tableName+"_id")
		for _, gid := range ids {
			ins = ins.Values(gid, id)
		}
		_, err = r.executeSQL(ins)
		return err
	})
	return err
}

type baseRepository interface {
	queryAll(SelectBuilder, any, ...model.QueryOptions) error
	getTableName() string
}

type modelWithGenres interface {
	model.Album | model.Artist | model.MediaFile
}

func getID[T modelWithGenres](item T) string {
	switch v := any(item).(type) {
	case model.Album:
		return v.ID
	case model.Artist:
		return v.ID
	case model.MediaFile:
		return v.ID
	}
	return ""
}

func appendGenre[T modelWithGenres](item *T, genre model.Genre) {
	switch v := any(item).(type) {
	case *model.Album:
		v.Genres = append(v.Genres, genre)
	case *model.Artist:
		v.Genres = append(v.Genres, genre)
	case *model.MediaFile:
		v.Genres = append(v.Genres, genre)
	}
}

func loadGenres[T modelWithGenres](r baseRepository, ids []string, items map[string]*T) error {
	tableName := r.getTableName()
	return slice.RangeByChunks(ids, 900, func(ids []string) error {
		sql := Select("genre.*", tableName+"_id as item_id").From("genre").
			Join(tableName+"_genres ig on genre.id = ig.genre_id").
			OrderBy(tableName+"_id", "ig.rowid").Where(Eq{tableName + "_id": ids})

		var genres []struct {
			model.Genre
			ItemID string
		}
		err := r.queryAll(sql, &genres)
		if err != nil {
			return err
		}
		for _, g := range genres {
			appendGenre(items[g.ItemID], g.Genre)
		}
		return nil
	})
}

func loadAllGenres[T modelWithGenres](r baseRepository, items []T) error {
	// Map references to items by ID and collect all IDs
	m := map[string]*T{}
	var ids []string
	for i := range items {
		item := &(items)[i]
		id := getID(*item)
		ids = append(ids, id)
		m[id] = item
	}

	return loadGenres(r, ids, m)
}
