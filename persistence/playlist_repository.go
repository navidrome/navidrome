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
	"github.com/navidrome/navidrome/utils"
)

type playlistRepository struct {
	sqlRepository
	sqlRestful
}

type dbPlaylist struct {
	model.Playlist `structs:",flatten"`
	RawRules       string `structs:"rules" orm:"column(rules)"`
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
	if p.IsSmartPlaylist() {
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
	return r.updateTracks(id, p.MediaFiles())
}

func (r *playlistRepository) Get(id string) (*model.Playlist, error) {
	return r.findBy(And{Eq{"id": id}, r.userFilter()})
}

func (r *playlistRepository) GetWithTracks(id string) (*model.Playlist, error) {
	pls, err := r.findBy(And{Eq{"id": id}, r.userFilter()})
	if err != nil {
		return nil, err
	}
	r.refreshSmartPlaylist(pls)
	if err := r.loadTracks(pls); err != nil {
		log.Error(r.ctx, "Error loading playlist tracks ", "playlist", pls.Name, "id", pls.ID, err)
		return nil, err
	}
	return pls, nil
}

func (r *playlistRepository) FindByPath(path string) (*model.Playlist, error) {
	return r.findBy(Eq{"path": path})
}

func (r *playlistRepository) findBy(sql Sqlizer) (*model.Playlist, error) {
	sel := r.newSelect().Columns("*").Where(sql)
	var pls []dbPlaylist
	err := r.queryAll(sel, &pls)
	if err != nil {
		return nil, err
	}
	if len(pls) == 0 {
		return nil, model.ErrNotFound
	}

	return r.toModel(pls[0])
}

func (r *playlistRepository) toModel(pls dbPlaylist) (*model.Playlist, error) {
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
		pls, err := r.toModel(p)
		if err != nil {
			return nil, err
		}
		playlists[i] = *pls
	}
	return playlists, err
}

func (r *playlistRepository) refreshSmartPlaylist(pls *model.Playlist) bool {
	// Only refresh if it is a smart playlist and was not refreshed in the last 5 seconds
	if !pls.IsSmartPlaylist() || time.Since(pls.EvaluatedAt) < 5*time.Second {
		return false
	}

	log.Debug(r.ctx, "Refreshing smart playlist", "playlist", pls.Name, "id", pls.ID)
	start := time.Now()

	// Remove old tracks
	del := Delete("playlist_tracks").Where(Eq{"playlist_id": pls.ID})
	_, err := r.executeSQL(del)
	if err != nil {
		return false
	}

	// Re-populate playlist based on Smart Playlist criteria
	sp := smartPlaylist(*pls.Rules)
	sql := Select("row_number() over (order by "+sp.OrderBy()+") as id", "'"+pls.ID+"' as playlist_id", "media_file.id as media_file_id").
		From("media_file").LeftJoin("annotation on (" +
		"annotation.item_id = media_file.id" +
		" AND annotation.item_type = 'media_file'" +
		" AND annotation.user_id = '" + userId(r.ctx) + "')")
	sql = sp.AddCriteria(sql)
	insSql := Insert("playlist_tracks").Columns("id", "playlist_id", "media_file_id").Select(sql)
	c, err := r.executeSQL(insSql)
	if err != nil {
		log.Error(r.ctx, "Error refreshing smart playlist tracks", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	// Update playlist stats
	err = r.updateStats(pls.ID)
	if err != nil {
		log.Error(r.ctx, "Error updating smart playlist stats", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	// Update when the playlist was last refreshed (for cache purposes)
	updSql := Update(r.tableName).Set("evaluated_at", time.Now()).Where(Eq{"id": pls.ID})
	_, err = r.executeSQL(updSql)
	if err != nil {
		log.Error(r.ctx, "Error updating smart playlist", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	log.Debug(r.ctx, "Refreshed playlist", "playlist", pls.Name, "id", pls.ID, "numTracks", c, "elapsed", time.Since(start))

	return true
}

func (r *playlistRepository) updateTracks(id string, tracks model.MediaFiles) error {
	ids := make([]string, len(tracks))
	for i := range tracks {
		ids[i] = tracks[i].ID
	}
	return r.updatePlaylist(id, ids)
}

func (r *playlistRepository) updatePlaylist(playlistId string, mediaFileIds []string) error {
	if !r.isWritable(playlistId) {
		return rest.ErrPermissionDenied
	}

	// Remove old tracks
	del := Delete("playlist_tracks").Where(Eq{"playlist_id": playlistId})
	_, err := r.executeSQL(del)
	if err != nil {
		return err
	}

	// Break the track list in chunks to avoid hitting SQLITE_MAX_FUNCTION_ARG limit
	chunks := utils.BreakUpStringSlice(mediaFileIds, 50)

	// Add new tracks, chunk by chunk
	pos := 1
	for i := range chunks {
		ins := Insert("playlist_tracks").Columns("playlist_id", "media_file_id", "id")
		for _, t := range chunks[i] {
			ins = ins.Values(playlistId, t, pos)
			pos++
		}
		_, err = r.executeSQL(ins)
		if err != nil {
			return err
		}
	}

	return r.updateStats(playlistId)
}

// updateStats updates total playlist duration, size and count
func (r *playlistRepository) updateStats(playlistId string) error {
	statsSql := Select("sum(duration) as duration", "sum(size) as size", "count(*) as count").
		From("media_file").
		Join("playlist_tracks f on f.media_file_id = media_file.id").
		Where(Eq{"playlist_id": playlistId})
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
		Where(Eq{"id": playlistId})
	_, err = r.executeSQL(upd)
	return err
}

func (r *playlistRepository) loadTracks(pls *model.Playlist) error {
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
		log.Error(r.ctx, "Error loading playlist tracks", "playlist", pls.Name, "id", pls.ID, err)
	}
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

func (r *playlistRepository) isWritable(playlistId string) bool {
	usr := loggedUser(r.ctx)
	if usr.IsAdmin {
		return true
	}
	pls, err := r.Get(playlistId)
	return err == nil && pls.Owner == usr.UserName
}

var _ model.PlaylistRepository = (*playlistRepository)(nil)
var _ rest.Repository = (*playlistRepository)(nil)
var _ rest.Persistable = (*playlistRepository)(nil)
