package persistence

import (
	. "github.com/Masterminds/squirrel"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type playlistTrackRepository struct {
	sqlRepository
	sqlRestful
	playlistId string
}

func (r *playlistRepository) Tracks(playlistId string) model.PlaylistTrackRepository {
	p := &playlistTrackRepository{}
	p.playlistId = playlistId
	p.ctx = r.ctx
	p.ormer = r.ormer
	p.tableName = "playlist_tracks"
	p.sortMappings = map[string]string{
		"id": "playlist_tracks.id",
	}
	return p
}

func (r *playlistTrackRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select().Where(Eq{"playlist_id": r.playlistId}), r.parseRestOptions(options...))
}

func (r *playlistTrackRepository) Read(id string) (interface{}, error) {
	sel := r.newSelect().
		LeftJoin("annotation on ("+
			"annotation.item_id = media_file_id"+
			" AND annotation.item_type = 'media_file'"+
			" AND annotation.user_id = '"+userId(r.ctx)+"')").
		Columns("starred", "starred_at", "play_count", "play_date", "rating", "f.*", "playlist_tracks.*").
		Join("media_file f on f.id = media_file_id").
		Where(And{Eq{"playlist_id": r.playlistId}, Eq{"id": id}})
	var trk model.PlaylistTrack
	err := r.queryOne(sel, &trk)
	return &trk, err
}

func (r *playlistTrackRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sel := r.newSelect(r.parseRestOptions(options...)).
		LeftJoin("annotation on ("+
			"annotation.item_id = media_file_id"+
			" AND annotation.item_type = 'media_file'"+
			" AND annotation.user_id = '"+userId(r.ctx)+"')").
		Columns("starred", "starred_at", "play_count", "play_date", "rating", "f.*", "playlist_tracks.*").
		Join("media_file f on f.id = media_file_id").
		Where(Eq{"playlist_id": r.playlistId})
	res := model.PlaylistTracks{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *playlistTrackRepository) EntityName() string {
	return "playlist_tracks"
}

func (r *playlistTrackRepository) NewInstance() interface{} {
	return &model.PlaylistTrack{}
}

func (r *playlistTrackRepository) Add(mediaFileIds []string) error {
	log.Debug(r.ctx, "Adding songs to playlist", "playlistId", r.playlistId, "mediaFileIds", mediaFileIds)

	// Get all current tracks
	all := r.newSelect().Columns("media_file_id").Where(Eq{"playlist_id": r.playlistId}).OrderBy("id")
	var tracks model.PlaylistTracks
	err := r.queryAll(all, &tracks)
	if err != nil {
		log.Error("Error querying current tracks from playlist", "playlistId", r.playlistId, err)
		return err
	}
	ids := make([]string, len(tracks))
	for i := range tracks {
		ids[i] = tracks[i].MediaFileID
	}

	// Append new tracks
	ids = append(ids, mediaFileIds...)

	// Update tracks and playlist
	return r.Update(ids)
}

func (r *playlistTrackRepository) Update(mediaFileIds []string) error {
	// Remove old tracks
	del := Delete(r.tableName).Where(Eq{"playlist_id": r.playlistId})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}

	// Break the track list in chunks to avoid hitting SQLITE_MAX_FUNCTION_ARG limit
	numTracks := len(mediaFileIds)
	const chunkSize = 50
	var chunks [][]string
	for i := 0; i < numTracks; i += chunkSize {
		end := i + chunkSize
		if end > numTracks {
			end = numTracks
		}

		chunks = append(chunks, mediaFileIds[i:end])
	}

	// Add new tracks, chunk by chunk
	pos := 0
	for i := range chunks {
		ins := Insert(r.tableName).Columns("playlist_id", "media_file_id", "id")
		for _, t := range chunks[i] {
			ins = ins.Values(r.playlistId, t, pos)
			pos++
		}
		_, err = r.executeSQL(ins)
		if err != nil {
			return err
		}
	}

	return r.updateStats()
}

func (r *playlistTrackRepository) updateStats() error {
	// Get total playlist duration and count
	statsSql := Select("sum(duration) as duration", "count(*) as count").From("media_file").
		Join("playlist_tracks f on f.media_file_id = media_file.id").
		Where(Eq{"playlist_id": r.playlistId})
	var res struct{ Duration, Count float32 }
	err := r.queryOne(statsSql, &res)
	if err != nil {
		return err
	}

	// Update playlist's total duration and count
	upd := Update("playlist").
		Set("duration", res.Duration).
		Set("song_count", res.Count).
		Where(Eq{"id": r.playlistId})
	_, err = r.executeSQL(upd)
	return err
}

func (r *playlistTrackRepository) Delete(id string) error {
	err := r.delete(And{Eq{"playlist_id": r.playlistId}, Eq{"id": id}})
	if err != nil {
		return err
	}
	return r.updateStats()
}

var _ model.PlaylistTrackRepository = (*playlistTrackRepository)(nil)
