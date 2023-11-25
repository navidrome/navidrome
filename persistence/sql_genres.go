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

func (r *sqlRepository) updateGenres(id string, tableName string, genres model.Genres) error {
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

func (r *sqlRepository) loadMediaFileGenres(mfs *model.MediaFiles) error {
	var ids []string
	m := map[string]*model.MediaFile{}
	for i := range *mfs {
		mf := &(*mfs)[i]
		ids = append(ids, mf.ID)
		m[mf.ID] = mf
	}

	return slice.RangeByChunks(ids, 900, func(ids []string) error {
		sql := Select("g.*", "mg.media_file_id").From("genre g").Join("media_file_genres mg on mg.genre_id = g.id").
			Where(Eq{"mg.media_file_id": ids}).OrderBy("mg.media_file_id", "mg.rowid")
		var genres []struct {
			model.Genre
			MediaFileId string
		}

		err := r.queryAll(sql, &genres)
		if err != nil {
			return err
		}
		for _, g := range genres {
			mf := m[g.MediaFileId]
			mf.Genres = append(mf.Genres, g.Genre)
		}
		return nil
	})
}

func (r *sqlRepository) loadAlbumGenres(mfs *model.Albums) error {
	var ids []string
	m := map[string]*model.Album{}
	for i := range *mfs {
		mf := &(*mfs)[i]
		ids = append(ids, mf.ID)
		m[mf.ID] = mf
	}

	return slice.RangeByChunks(ids, 900, func(ids []string) error {
		sql := Select("g.*", "ag.album_id").From("genre g").Join("album_genres ag on ag.genre_id = g.id").
			Where(Eq{"ag.album_id": ids}).OrderBy("ag.album_id", "ag.rowid")
		var genres []struct {
			model.Genre
			AlbumId string
		}

		err := r.queryAll(sql, &genres)
		if err != nil {
			return err
		}
		for _, g := range genres {
			mf := m[g.AlbumId]
			mf.Genres = append(mf.Genres, g.Genre)
		}
		return nil
	})
}

func (r *sqlRepository) loadArtistGenres(mfs *model.Artists) error {
	var ids []string
	m := map[string]*model.Artist{}
	for i := range *mfs {
		mf := &(*mfs)[i]
		ids = append(ids, mf.ID)
		m[mf.ID] = mf
	}

	return slice.RangeByChunks(ids, 900, func(ids []string) error {
		sql := Select("g.*", "ag.artist_id").From("genre g").Join("artist_genres ag on ag.genre_id = g.id").
			Where(Eq{"ag.artist_id": ids}).OrderBy("ag.artist_id", "ag.rowid")
		var genres []struct {
			model.Genre
			ArtistId string
		}

		err := r.queryAll(sql, &genres)
		if err != nil {
			return err
		}
		for _, g := range genres {
			mf := m[g.ArtistId]
			mf.Genres = append(mf.Genres, g.Genre)
		}
		return nil
	})
}
