package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/utils/slice"
)

type playlistRepository struct {
	sqlRepository
	sqlRestful
}

type dbPlaylist struct {
	model.Playlist `structs:",flatten"`
	RawRules       string `structs:"rules" orm:"column(rules)"`
}

func NewPlaylistRepository(ctx context.Context, o orm.QueryExecutor) model.PlaylistRepository {
	r := &playlistRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "playlist"
	r.filterMappings = map[string]filterFunc{
		"q":     playlistFilter,
		"smart": smartPlaylistFilter,
	}
	return r
}

func playlistFilter(_ string, value interface{}) Sqlizer {
	return Or{
		substringFilter("playlist.name", value),
		substringFilter("playlist.comment", value),
	}
}

func smartPlaylistFilter(string, interface{}) Sqlizer {
	return Or{
		Eq{"rules": ""},
		Eq{"rules": nil},
	}
}

func (r *playlistRepository) userFilter() Sqlizer {
	user := loggedUser(r.ctx)
	if user.IsAdmin {
		return And{}
	}
	return Or{
		Eq{"public": true},
		Eq{"owner_id": user.ID},
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
		if pls.OwnerID != usr.ID {
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

	id, err := r.put(pls.ID, pls)
	if err != nil {
		return err
	}
	p.ID = id

	if p.IsSmartPlaylist() {
		r.refreshSmartPlaylist(p)
		return nil
	}
	// Only update tracks if they were specified
	if len(pls.Tracks) > 0 {
		return r.updateTracks(id, p.MediaFiles())
	}
	return r.refreshCounters(&pls.Playlist)
}

func (r *playlistRepository) Get(id string) (*model.Playlist, error) {
	return r.findBy(And{Eq{"playlist.id": id}, r.userFilter()})
}

func (r *playlistRepository) GetWithTracks(id string, refreshSmartPlaylist bool) (*model.Playlist, error) {
	pls, err := r.Get(id)
	if err != nil {
		return nil, err
	}
	if refreshSmartPlaylist {
		r.refreshSmartPlaylist(pls)
	}
	tracks, err := r.loadTracks(Select().From("playlist_tracks"), id)
	if err != nil {
		log.Error(r.ctx, "Error loading playlist tracks ", "playlist", pls.Name, "id", pls.ID, err)
		return nil, err
	}
	pls.Tracks = tracks
	return pls, nil
}

func (r *playlistRepository) FindByPath(path string) (*model.Playlist, error) {
	return r.findBy(Eq{"path": path})
}

func (r *playlistRepository) findBy(sql Sqlizer) (*model.Playlist, error) {
	sel := r.selectPlaylist().Where(sql)
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
		var c criteria.Criteria
		err = json.Unmarshal([]byte(pls.RawRules), &c)
		if err != nil {
			return nil, err
		}
		pls.Playlist.Rules = &c
	} else {
		pls.Playlist.Rules = nil
	}
	return &pls.Playlist, err
}

func (r *playlistRepository) GetAll(options ...model.QueryOptions) (model.Playlists, error) {
	sel := r.selectPlaylist(options...).Where(r.userFilter())
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

func (r *playlistRepository) selectPlaylist(options ...model.QueryOptions) SelectBuilder {
	return r.newSelect(options...).Join("user on user.id = owner_id").
		Columns(r.tableName+".*", "user.user_name as owner_name")
}

func (r *playlistRepository) refreshSmartPlaylist(pls *model.Playlist) bool {
	// Only refresh if it is a smart playlist and was not refreshed in the last 5 seconds
	if !pls.IsSmartPlaylist() || time.Since(pls.EvaluatedAt) < 5*time.Second {
		return false
	}

	// Never refresh other users' playlists
	usr := loggedUser(r.ctx)
	if pls.OwnerID != usr.ID {
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
	rules := *pls.Rules
	sql := Select("row_number() over (order by "+rules.OrderBy()+") as id", "'"+pls.ID+"' as playlist_id", "media_file.id as media_file_id").
		From("media_file").LeftJoin("annotation on (" +
		"annotation.item_id = media_file.id" +
		" AND annotation.item_type = 'media_file'" +
		" AND annotation.user_id = '" + userId(r.ctx) + "')").
		LeftJoin("media_file_genres ag on media_file.id = ag.media_file_id").
		LeftJoin("genre on ag.genre_id = genre.id").GroupBy("media_file.id")
	sql = r.addCriteria(sql, rules)
	insSql := Insert("playlist_tracks").Columns("id", "playlist_id", "media_file_id").Select(sql)
	_, err = r.executeSQL(insSql)
	if err != nil {
		log.Error(r.ctx, "Error refreshing smart playlist tracks", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	// Update playlist stats
	err = r.refreshCounters(pls)
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

	log.Debug(r.ctx, "Refreshed playlist", "playlist", pls.Name, "id", pls.ID, "numTracks", pls.SongCount, "elapsed", time.Since(start))

	return true
}

func (r *playlistRepository) addCriteria(sql SelectBuilder, c criteria.Criteria) SelectBuilder {
	sql = sql.Where(c.ToSql())
	if c.Limit > 0 {
		sql = sql.Limit(uint64(c.Limit)).Offset(uint64(c.Offset))
	}
	if order := c.OrderBy(); order != "" {
		sql = sql.OrderBy(order)
	}
	return sql
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

	return r.addTracks(playlistId, 1, mediaFileIds)
}

func (r *playlistRepository) addTracks(playlistId string, startingPos int, mediaFileIds []string) error {
	// Break the track list in chunks to avoid hitting SQLITE_MAX_FUNCTION_ARG limit
	chunks := slice.BreakUp(mediaFileIds, 200)

	// Add new tracks, chunk by chunk
	pos := startingPos
	for i := range chunks {
		ins := Insert("playlist_tracks").Columns("playlist_id", "media_file_id", "id")
		for _, t := range chunks[i] {
			ins = ins.Values(playlistId, t, pos)
			pos++
		}
		_, err := r.executeSQL(ins)
		if err != nil {
			return err
		}
	}

	return r.refreshCounters(&model.Playlist{ID: playlistId})
}

// RefreshStatus updates total playlist duration, size and count
func (r *playlistRepository) refreshCounters(pls *model.Playlist) error {
	statsSql := Select("sum(duration) as duration", "sum(size) as size", "count(*) as count").
		From("media_file").
		Join("playlist_tracks f on f.media_file_id = media_file.id").
		Where(Eq{"playlist_id": pls.ID})
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
		Where(Eq{"id": pls.ID})
	_, err = r.executeSQL(upd)
	if err != nil {
		return err
	}
	pls.SongCount = int(res.Count)
	pls.Duration = res.Duration
	pls.Size = int64(res.Size)
	return nil
}

func (r *playlistRepository) loadTracks(sel SelectBuilder, id string) (model.PlaylistTracks, error) {
	tracksQuery := sel.
		Columns("starred", "starred_at", "play_count", "play_date", "rating", "f.*", "playlist_tracks.*").
		LeftJoin("annotation on (" +
			"annotation.item_id = media_file_id" +
			" AND annotation.item_type = 'media_file'" +
			" AND annotation.user_id = '" + userId(r.ctx) + "')").
		Join("media_file f on f.id = media_file_id").
		Where(Eq{"playlist_id": id}).OrderBy("playlist_tracks.id")
	tracks := model.PlaylistTracks{}
	err := r.queryAll(tracksQuery, &tracks)
	for i, t := range tracks {
		tracks[i].MediaFile.ID = t.MediaFileID
	}
	return tracks, err
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
	pls.OwnerID = loggedUser(r.ctx).ID
	pls.ID = "" // Make sure we don't override an existing playlist
	err := r.Put(pls)
	if err != nil {
		return "", err
	}
	return pls.ID, err
}

func (r *playlistRepository) Update(id string, entity interface{}, cols ...string) error {
	current, err := r.Get(id)
	if err != nil {
		return err
	}
	usr := loggedUser(r.ctx)
	if !usr.IsAdmin && current.OwnerID != usr.ID {
		return rest.ErrPermissionDenied
	}
	pls := entity.(*model.Playlist)
	pls.ID = id
	pls.UpdatedAt = time.Now()
	_, err = r.put(id, pls, append(cols, "updatedAt")...)
	if errors.Is(err, model.ErrNotFound) {
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

		// Renumber the playlist if any track was removed
		if err := r.renumber(pl.Id); err != nil {
			return err
		}
	}
	return nil
}

func (r *playlistRepository) renumber(id string) error {
	var ids []string
	sql := Select("media_file_id").From("playlist_tracks").Where(Eq{"playlist_id": id}).OrderBy("id")
	err := r.queryAll(sql, &ids)
	if err != nil {
		return err
	}
	return r.updatePlaylist(id, ids)
}

func (r *playlistRepository) isWritable(playlistId string) bool {
	usr := loggedUser(r.ctx)
	if usr.IsAdmin {
		return true
	}
	pls, err := r.Get(playlistId)
	return err == nil && pls.OwnerID == usr.ID
}

var _ model.PlaylistRepository = (*playlistRepository)(nil)
var _ rest.Repository = (*playlistRepository)(nil)
var _ rest.Persistable = (*playlistRepository)(nil)
