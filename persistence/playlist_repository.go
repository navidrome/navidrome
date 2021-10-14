package persistence

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type playlistRepository struct {
	sqlRepository
	sqlRestful
}

type dbPlaylist struct {
	model.Playlist `structs:",flatten"`
	RawRules       string `structs:"rules"`
}

func NewPlaylistRepository(ctx context.Context, o orm.Ormer) model.PlaylistRepository {
	r := &playlistRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "playlist"
	return r
}

func (r *playlistRepository) userFilter() Sqlizer {
	user := loggedUser(r.ctx)
	if user.IsAdmin {
		return And{}
	}
	return Or{
		Eq{"public": true},
		Eq{"owner": user.UserName},
	}
}

func (r *playlistRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := Select().Where(r.userFilter())
	return r.count(sql, options...)
}

func (r *playlistRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(And{Eq{"id": id}, r.userFilter()}))
}

func (r *playlistRepository) Delete(id string) error {
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin {
		pls, err := r.Get(id)
		if err != nil {
			return err
		}
		if pls.Owner != usr.UserName {
			return rest.ErrPermissionDenied
		}
	}
	return r.delete(And{Eq{"id": id}, r.userFilter()})
}

func (r *playlistRepository) Put(p *model.Playlist) error {
	pls := dbPlaylist{Playlist: *p}
	if p.Rules != nil {
		j, err := json.Marshal(p.Rules)
		if err != nil {
			return err
		}
		pls.RawRules = string(j)
	}
	if pls.ID == "" {
		pls.CreatedAt = time.Now()
	} else {
		ok, err := r.Exists(pls.ID)
		if err != nil {
			return err
		}
		if !ok {
			return model.ErrNotAuthorized
		}
	}
	pls.UpdatedAt = time.Now()

	// Save tracks for later and set it to nil, to avoid trying to save it to the DB
	tracks := pls.Tracks
	pls.Tracks = nil

	id, err := r.put(pls.ID, pls)
	if err != nil {
		return err
	}
	p.ID = id

	// Only update tracks if they are specified
	if tracks == nil {
		return nil
	}
	return r.updateTracks(id, tracks)
}

func (r *playlistRepository) Get(id string) (*model.Playlist, error) {
	return r.findBy(And{Eq{"id": id}, r.userFilter()}, true)
}

func (r *playlistRepository) FindByPath(path string) (*model.Playlist, error) {
	return r.findBy(Eq{"path": path}, false)
}

func (r *playlistRepository) FindByID(id string) (*model.Playlist, error) {
	return r.findBy(And{Eq{"id": id}, r.userFilter()}, false)
}

func (r *playlistRepository) findBy(sql Sqlizer, includeTracks bool) (*model.Playlist, error) {
	sel := r.newSelect().Columns("*").Where(sql)
	var pls []dbPlaylist
	err := r.queryAll(sel, &pls)
	if err != nil {
		return nil, err
	}
	if len(pls) == 0 {
		return nil, model.ErrNotFound
	}

	return r.toModel(pls[0], includeTracks)
}

func (r *playlistRepository) toModel(pls dbPlaylist, includeTracks bool) (*model.Playlist, error) {
	var err error
	if strings.TrimSpace(pls.RawRules) != "" {
		r := model.SmartPlaylist{}
		err = json.Unmarshal([]byte(pls.RawRules), &r)
		if err != nil {
			return nil, err
		}
		pls.Playlist.Rules = &r
	} else {
		pls.Playlist.Rules = nil
	}
	if includeTracks {
		err = r.loadTracks(&pls)
	}
	return &pls.Playlist, err
}

func (r *playlistRepository) GetAll(options ...model.QueryOptions) (model.Playlists, error) {
	sel := r.newSelect(options...).Columns("*").Where(r.userFilter())
	var res []dbPlaylist
	err := r.queryAll(sel, &res)
	if err != nil {
		return nil, err
	}
	playlists := make(model.Playlists, len(res))
	for i, p := range res {
		pls, err := r.toModel(p, false)
		if err != nil {
			return nil, err
		}
		playlists[i] = *pls
	}
	return playlists, err
}

func (r *playlistRepository) updateTracks(id string, tracks model.MediaFiles) error {
	ids := make([]string, len(tracks))
	for i := range tracks {
		ids[i] = tracks[i].ID
	}
	return r.Tracks(id).Update(ids)
}

func (r *playlistRepository) loadTracks(pls *dbPlaylist) error {
	tracksQuery := Select().From("playlist_tracks").
		LeftJoin("annotation on ("+
			"annotation.item_id = media_file_id"+
			" AND annotation.item_type = 'media_file'"+
			" AND annotation.user_id = '"+userId(r.ctx)+"')").
		Columns("starred", "starred_at", "play_count", "play_date", "rating", "f.*").
		Join("media_file f on f.id = media_file_id").
		Where(Eq{"playlist_id": pls.ID}).OrderBy("playlist_tracks.id")
	err := r.queryAll(tracksQuery, &pls.Tracks)
	if err != nil {
		log.Error("Error loading playlist tracks", "playlist", pls.Name, "id", pls.ID)
	}
	err = r.loadMediaFileGenres(&pls.Tracks)
	return err
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
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin && pls.Owner != usr.UserName {
		return rest.ErrPermissionDenied
	}
	err := r.Put(pls)
	if err == model.ErrNotFound {
		return rest.ErrNotFound
	}
	return err
}

func (r *playlistRepository) removeOrphans() error {
	sel := Select("playlist_tracks.playlist_id as id", "p.name").From("playlist_tracks").
		Join("playlist p on playlist_tracks.playlist_id = p.id").
		LeftJoin("media_file mf on playlist_tracks.media_file_id = mf.id").
		Where(Eq{"mf.id": nil}).
		GroupBy("playlist_tracks.playlist_id")

	var pls []struct{ Id, Name string }
	err := r.queryAll(sel, &pls)
	if err != nil {
		return err
	}

	for _, pl := range pls {
		log.Debug(r.ctx, "Cleaning-up orphan tracks from playlist", "id", pl.Id, "name", pl.Name)
		del := Delete("playlist_tracks").Where(And{
			ConcatExpr("media_file_id not in (select id from media_file)"),
			Eq{"playlist_id": pl.Id},
		})
		n, err := r.executeSQL(del)
		if n == 0 || err != nil {
			return err
		}
		log.Debug(r.ctx, "Deleted tracks, now reordering", "id", pl.Id, "name", pl.Name, "deleted", n)

		// To reorganize the playlist, just add an empty list of new tracks
		tracks := r.Tracks(pl.Id)
		if _, err := tracks.Add(nil); err != nil {
			return err
		}
	}
	return nil
}

var _ model.PlaylistRepository = (*playlistRepository)(nil)
var _ rest.Repository = (*playlistRepository)(nil)
var _ rest.Persistable = (*playlistRepository)(nil)
