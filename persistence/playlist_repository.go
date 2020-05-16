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
	res := model.Playlists{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *playlistRepository) updateTracks(id string, tracks model.MediaFiles) error {
	ids := make([]string, len(tracks))
	for i := range tracks {
		ids[i] = tracks[i].ID
	}
	return r.Tracks(id).Update(ids)
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

func (r *playlistRepository) Save(entity interface{}) (string, error) {
	pls := entity.(*model.Playlist)
	pls.Owner = loggedUser(r.ctx).UserName
	err := r.Put(pls)
	if err != nil {
		return "", err
	}
	return pls.ID, err
}

func (r *playlistRepository) Update(entity interface{}, cols ...string) error {
	pls := entity.(*model.Playlist)
	err := r.Put(pls)
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

var _ model.PlaylistRepository = (*playlistRepository)(nil)
var _ rest.Repository = (*playlistRepository)(nil)
var _ rest.Persistable = (*playlistRepository)(nil)
