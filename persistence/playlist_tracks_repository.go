package persistence

import (
	. "github.com/Masterminds/squirrel"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type playlistTracksRepository struct {
	sqlRepository
	sqlRestful
	playlistId string
}

func (r *playlistRepository) Tracks(playlistId string) model.PlaylistTracksRepository {
	p := &playlistTracksRepository{}
	p.playlistId = playlistId
	p.ctx = r.ctx
	p.ormer = r.ormer
	p.tableName = "playlist_tracks"
	p.sortMappings = map[string]string{
		"id": "playlist_tracks.id",
	}
	return p
}

func (r *playlistTracksRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select().Where(Eq{"playlist_id": r.playlistId}), r.parseRestOptions(options...))
}

func (r *playlistTracksRepository) Read(id string) (interface{}, error) {
	sel := r.newSelect().
		LeftJoin("annotation on ("+
			"annotation.item_id = media_file_id"+
			" AND annotation.item_type = 'media_file'"+
			" AND annotation.user_id = '"+userId(r.ctx)+"')").
		Columns("starred", "starred_at", "play_count", "play_date", "rating", "f.*", "playlist_tracks.*").
		Join("media_file f on f.id = media_file_id").
		Where(And{Eq{"playlist_id": r.playlistId}, Eq{"id": id}})
	var trk model.PlaylistTracks
	err := r.queryOne(sel, &trk)
	return &trk, err
}

func (r *playlistTracksRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sel := r.newSelect(r.parseRestOptions(options...)).
		LeftJoin("annotation on ("+
			"annotation.item_id = media_file_id"+
			" AND annotation.item_type = 'media_file'"+
			" AND annotation.user_id = '"+userId(r.ctx)+"')").
		Columns("starred", "starred_at", "play_count", "play_date", "rating", "f.*", "playlist_tracks.*").
		Join("media_file f on f.id = media_file_id").
		Where(Eq{"playlist_id": r.playlistId})
	var res []model.PlaylistTracks
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *playlistTracksRepository) EntityName() string {
	return "playlist_tracks"
}

func (r *playlistTracksRepository) NewInstance() interface{} {
	return &model.PlaylistTracks{}
}

var _ model.PlaylistTracksRepository = (*playlistTracksRepository)(nil)
var _ model.ResourceRepository = (*playlistTracksRepository)(nil)
