package persistence

import (
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

type playlistTrackRepository struct {
	sqlRepository
	sqlRestful
	playlistId   string
	playlistRepo *playlistRepository
}

func (r *playlistRepository) Tracks(playlistId string) model.PlaylistTrackRepository {
	p := &playlistTrackRepository{}
	p.playlistRepo = r
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

func (r *playlistTrackRepository) GetAll(options ...model.QueryOptions) (model.PlaylistTracks, error) {
	sel := r.newSelect(options...).
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

func (r *playlistTrackRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *playlistTrackRepository) EntityName() string {
	return "playlist_tracks"
}

func (r *playlistTrackRepository) NewInstance() interface{} {
	return &model.PlaylistTrack{}
}

func (r *playlistTrackRepository) Add(mediaFileIds []string) (int, error) {
	if !r.isWritable() {
		return 0, rest.ErrPermissionDenied
	}

	if len(mediaFileIds) > 0 {
		log.Debug(r.ctx, "Adding songs to playlist", "playlistId", r.playlistId, "mediaFileIds", mediaFileIds)
	}

	ids, err := r.getTracks()
	if err != nil {
		return 0, err
	}

	// Append new tracks
	ids = append(ids, mediaFileIds...)

	// Update tracks and playlist
	return len(mediaFileIds), r.Update(ids)
}

func (r *playlistTrackRepository) AddAlbums(albumIds []string) (int, error) {
	sq := Select("id").From("media_file").Where(Eq{"album_id": albumIds})
	return r.addMediaFileIds(sq)
}

func (r *playlistTrackRepository) AddArtists(artistIds []string) (int, error) {
	sq := Select("id").From("media_file").Where(Eq{"album_artist_id": artistIds})
	return r.addMediaFileIds(sq)
}

func (r *playlistTrackRepository) AddDiscs(discs []model.DiscID) (int, error) {
	sq := Select("id").From("media_file")
	if len(discs) == 0 {
		return 0, nil
	}
	var clauses []Sqlizer
	for _, d := range discs {
		clauses = append(clauses, And{Eq{"album_id": d.AlbumID}, Eq{"disc_number": d.DiscNumber}})
	}
	sq = sq.Where(Or(clauses))
	return r.addMediaFileIds(sq)
}

func (r *playlistTrackRepository) addMediaFileIds(sq SelectBuilder) (int, error) {
	var res []struct{ Id string }
	sq = sq.OrderBy("album_artist, album, disc_number, track_number")
	err := r.queryAll(sq, &res)
	if err != nil {
		log.Error(r.ctx, "Error getting tracks to add to playlist", err)
		return 0, err
	}
	if len(res) == 0 {
		return 0, nil
	}
	var ids []string
	for _, r := range res {
		ids = append(ids, r.Id)
	}
	return r.Add(ids)
}

func (r *playlistTrackRepository) getTracks() ([]string, error) {
	// Get all current tracks
	all := r.newSelect().Columns("media_file_id").Where(Eq{"playlist_id": r.playlistId}).OrderBy("id")
	var tracks model.PlaylistTracks
	err := r.queryAll(all, &tracks)
	if err != nil {
		log.Error("Error querying current tracks from playlist", "playlistId", r.playlistId, err)
		return nil, err
	}
	ids := make([]string, len(tracks))
	for i := range tracks {
		ids[i] = tracks[i].MediaFileID
	}
	return ids, nil
}

func (r *playlistTrackRepository) Update(mediaFileIds []string) error {
	if !r.isWritable() {
		return rest.ErrPermissionDenied
	}

	// Remove old tracks
	del := Delete(r.tableName).Where(Eq{"playlist_id": r.playlistId})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}

	// Break the track list in chunks to avoid hitting SQLITE_MAX_FUNCTION_ARG limit
	chunks := utils.BreakUpStringSlice(mediaFileIds, 50)

	// Add new tracks, chunk by chunk
	pos := 1
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
	// Get total playlist duration, size and count
	statsSql := Select("sum(duration) as duration", "sum(size) as size", "count(*) as count").
		From("media_file").
		Join("playlist_tracks f on f.media_file_id = media_file.id").
		Where(Eq{"playlist_id": r.playlistId})
	var res struct{ Duration, Size, Count float32 }
	err := r.queryOne(statsSql, &res)
	if err != nil {
		return err
	}

	// Update playlist's total duration, size and count
	upd := Update("playlist").
		Set("duration", res.Duration).
		Set("size", res.Size).
		Set("song_count", res.Count).
		Set("updated_at", time.Now()).
		Where(Eq{"id": r.playlistId})
	_, err = r.executeSQL(upd)
	return err
}

func (r *playlistTrackRepository) Delete(id string) error {
	if !r.isWritable() {
		return rest.ErrPermissionDenied
	}
	err := r.delete(And{Eq{"playlist_id": r.playlistId}, Eq{"id": id}})
	if err != nil {
		return err
	}

	// To renumber the playlist
	_, err = r.Add(nil)
	return err
}

func (r *playlistTrackRepository) Reorder(pos int, newPos int) error {
	if !r.isWritable() {
		return rest.ErrPermissionDenied
	}
	ids, err := r.getTracks()
	if err != nil {
		return err
	}
	newOrder := utils.MoveString(ids, pos-1, newPos-1)
	return r.Update(newOrder)
}

func (r *playlistTrackRepository) isWritable() bool {
	usr := loggedUser(r.ctx)
	if usr.IsAdmin {
		return true
	}
	pls, err := r.playlistRepo.Get(r.playlistId)
	return err == nil && pls.Owner == usr.UserName
}

var _ model.PlaylistTrackRepository = (*playlistTrackRepository)(nil)
