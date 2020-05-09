package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type playlistRepository struct {
	sqlRepository
	sqlRestful
}

func NewPlaylistRepository(ctx context.Context, o orm.Ormer) model.PlaylistRepository {
	r := &playlistRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "playlist"
	return r
}

func (r *playlistRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(Select(), options...)
}

func (r *playlistRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"id": id}))
}

func (r *playlistRepository) Delete(id string) error {
	del := Delete("playlist_tracks").Where(Eq{"playlist_id": id})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}
	return r.delete(Eq{"id": id})
}

func (r *playlistRepository) Put(p *model.Playlist) error {
	if p.ID == "" {
		p.CreatedAt = time.Now()
	}
	p.UpdatedAt = time.Now()

	// Save tracks for later and set it to nil, to avoid trying to save it to the DB
	tracks := p.Tracks
	p.Tracks = nil

	id, err := r.put(p.ID, p)
	if err != nil {
		return err
	}
	err = r.updateTracks(id, tracks)
	return err
}

func (r *playlistRepository) Get(id string) (*model.Playlist, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var pls model.Playlist
	err := r.queryOne(sel, &pls)
	if err != nil {
		return nil, err
	}
	err = r.loadTracks(&pls)
	return &pls, err
}

func (r *playlistRepository) GetAll(options ...model.QueryOptions) (model.Playlists, error) {
	sel := r.newSelect(options...).Columns("*")
	var res model.Playlists
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	// TODO Maybe the tracks are not required when retrieving all playlists?
	err = r.loadAllTracks(res)
	return res, err
}

func (r *playlistRepository) Tracks(playlistId string) model.PlaylistTracksRepository {
	p := &playlistTracksRepository{}
	p.playlistId = playlistId
	p.ctx = r.ctx
	p.ormer = r.ormer
	p.tableName = "playlist_tracks"
	return p
}

func (r *playlistRepository) updateTracks(id string, tracks model.MediaFiles) error {
	// Remove old tracks
	del := Delete("playlist_tracks").Where(Eq{"playlist_id": id})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}

	// Add new tracks
	for i, t := range tracks {
		ins := Insert("playlist_tracks").Columns("playlist_id", "media_file_id", "id").
			Values(id, t.ID, i)
		_, err = r.executeSQL(ins)
		if err != nil {
			return err
		}
	}

	// Get total playlist duration and count
	statsSql := Select("sum(duration) as duration", "count(*) as count").From("media_file").
		Join("playlist_tracks f on f.media_file_id = media_file.id").
		Where(Eq{"playlist_id": id})
	var res struct{ Duration, Count float32 }
	err = r.queryOne(statsSql, &res)
	if err != nil {
		return err
	}

	// Update total playlist duration and count
	upd := Update(r.tableName).
		Set("duration", res.Duration).
		Set("song_count", res.Count).
		Where(Eq{"id": id})
	_, err = r.executeSQL(upd)
	return err
}

func (r *playlistRepository) loadAllTracks(all model.Playlists) error {
	for i := range all {
		if err := r.loadTracks(&all[i]); err != nil {
			return err
		}
	}
	return nil
}

func (r *playlistRepository) loadTracks(pls *model.Playlist) (err error) {
	tracksQuery := Select().From("playlist_tracks").
		LeftJoin("annotation on ("+
			"annotation.item_id = media_file_id"+
			" AND annotation.item_type = 'media_file'"+
			" AND annotation.user_id = '"+userId(r.ctx)+"')").
		Columns("starred", "starred_at", "play_count", "play_date", "rating", "f.*").
		Join("media_file f on f.id = media_file_id").
		Where(Eq{"playlist_id": pls.ID}).OrderBy("playlist_tracks.id")
	err = r.queryAll(tracksQuery, &pls.Tracks)
	if err != nil {
		log.Error("Error loading playlist tracks", "playlist", pls.Name, "id", pls.ID)
	}
	return
}

func (r *playlistRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *playlistRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *playlistRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *playlistRepository) EntityName() string {
	return "playlist"
}

func (r *playlistRepository) NewInstance() interface{} {
	return &model.Playlist{}
}

var _ model.PlaylistRepository = (*playlistRepository)(nil)
var _ model.ResourceRepository = (*playlistRepository)(nil)
