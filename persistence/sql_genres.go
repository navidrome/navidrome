package persistence

import (
	"slices"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
)

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

	for chunk := range slices.Chunk(ids, 900) {
		sql := Select("genre.*", tableName+"_id as item_id").From("genre").
			Join(tableName+"_genres ig on genre.id = ig.genre_id").
			OrderBy(tableName+"_id", "ig.rowid").Where(Eq{tableName + "_id": chunk})

		var genres []struct {
			model.Genre
			ItemID string
		}
		if err := r.queryAll(sql, &genres); err != nil {
			return err
		}
		for _, g := range genres {
			appendGenre(items[g.ItemID], g.Genre)
		}
	}
	return nil
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
