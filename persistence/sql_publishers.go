package persistence

import (
	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

func (r sqlRepository) withPublishers(sql SelectBuilder) SelectBuilder {
	return sql.LeftJoin(r.tableName + "_publishers ap on " + r.tableName + ".id = ap." + r.tableName + "_id").
		LeftJoin("publisher on ap.publisher_id = publisher.id")
}

func (r *sqlRepository) updatePublishers(id string, tableName string, publishers model.Publishers) error {
	del := Delete(tableName + "_publishers").Where(Eq{tableName + "_id": id})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}

	if len(publishers) == 0 {
		return nil
	}
	var publisherIds []string
	for _, p := range publishers {
		publisherIds = append(publisherIds, p.ID)
	}
	err = utils.RangeByChunks(publisherIds, 100, func(ids []string) error {
		ins := Insert(tableName+"_publishers").Columns("publisher_id", tableName+"_id")
		for _, pid := range ids {
			ins = ins.Values(pid, id)
		}
		_, err = r.executeSQL(ins)
		return err
	})
	return err
}

func (r *sqlRepository) loadMediaFilePublishers(mfs *model.MediaFiles) error {
	var ids []string
	m := map[string]*model.MediaFile{}
	for i := range *mfs {
		mf := &(*mfs)[i]
		ids = append(ids, mf.ID)
		m[mf.ID] = mf
	}

	return utils.RangeByChunks(ids, 900, func(ids []string) error {
		sql := Select("p.*", "mp.media_file_id").From("publisher p").Join("media_file_publishers mp on mp.publisher_id = p.id").
			Where(Eq{"mp.media_file_id": ids}).OrderBy("mp.media_file_id", "mp.rowid")
		var publishers []struct {
			model.Publisher
			MediaFileId string
		}

		err := r.queryAll(sql, &publishers)
		if err != nil {
			return err
		}
		for _, p := range publishers {
			mf := m[p.MediaFileId]
			mf.Publishers = append(mf.Publishers, p.Publisher)
		}
		return nil
	})
}

func (r *sqlRepository) loadAlbumPublishers(mfs *model.Albums) error {
	var ids []string
	m := map[string]*model.Album{}
	for i := range *mfs {
		mf := &(*mfs)[i]
		ids = append(ids, mf.ID)
		m[mf.ID] = mf
	}

	return utils.RangeByChunks(ids, 900, func(ids []string) error {
		sql := Select("p.*", "ap.album_id").From("publisher p").Join("album_publishers ap on ap.publisher_id = p.id").
			Where(Eq{"ap.album_id": ids}).OrderBy("ap.album_id", "ap.rowid")
		var publishers []struct {
			model.Publisher
			AlbumId string
		}

		err := r.queryAll(sql, &publishers)
		if err != nil {
			return err
		}
		for _, g := range publishers {
			mf := m[g.AlbumId]
			mf.Publishers = append(mf.Publishers, g.Publisher)
		}
		return nil
	})
}

func (r *sqlRepository) loadArtistPublishers(mfs *model.Artists) error {
	var ids []string
	m := map[string]*model.Artist{}
	for i := range *mfs {
		mf := &(*mfs)[i]
		ids = append(ids, mf.ID)
		m[mf.ID] = mf
	}

	return utils.RangeByChunks(ids, 900, func(ids []string) error {
		sql := Select("p.*", "ap.artist_id").From("publisher p").Join("artist_publishers ap on ap.publisher_id = p.id").
			Where(Eq{"ap.artist_id": ids}).OrderBy("ap.artist_id", "ap.rowid")
		var publishers []struct {
			model.Publisher
			ArtistId string
		}

		err := r.queryAll(sql, &publishers)
		if err != nil {
			return err
		}
		for _, g := range publishers {
			mf := m[g.ArtistId]
			mf.Publishers = append(mf.Publishers, g.Publisher)
		}
		return nil
	})
}
