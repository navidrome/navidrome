package persistence

import (
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// PlaylistRepository methods to handle smart playlists, which are defined by criteria and automatically populated
// based on their rules. The main method is refreshSmartPlaylist, which evaluates the criteria and updates the playlist
// tracks accordingly. It also handles refreshing dependent playlists when a smart playlist references other playlists
// in its criteria. To optimize performance, it only refreshes when necessary based on the last evaluated time and
// configured refresh delay.

// refreshSmartPlaylist evaluates the criteria of a smart playlist and updates its tracks accordingly.
func (r *playlistRepository) refreshSmartPlaylist(pls *model.Playlist) bool {
	usr := loggedUser(r.ctx)
	if !r.shouldRefreshSmartPlaylist(pls, usr) {
		return false
	}

	log.Debug(r.ctx, "Refreshing smart playlist", "playlist", pls.Name, "id", pls.ID)
	start := time.Now()

	del := Delete("playlist_tracks").Where(Eq{"playlist_id": pls.ID})
	if _, err := r.executeSQL(del); err != nil {
		log.Error(r.ctx, "Error deleting old smart playlist tracks", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	rulesSQL := newSmartPlaylistCriteria(*pls.Rules, withSmartPlaylistOwner(*usr))

	if !r.refreshChildPlaylists(pls, rulesSQL) {
		return false
	}

	if err := r.resolvePercentageLimit(pls, &rulesSQL, usr.ID); err != nil {
		return false
	}

	sq := r.buildSmartPlaylistQuery(pls, rulesSQL, usr.ID)
	sq, err := r.addCriteria(sq, rulesSQL)
	if err != nil {
		log.Error(r.ctx, "Error building smart playlist criteria", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	insSql := Insert("playlist_tracks").Columns("id", "playlist_id", "media_file_id").Select(sq)
	if _, err = r.executeSQL(insSql); err != nil {
		log.Error(r.ctx, "Error refreshing smart playlist tracks", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	if err = r.refreshCounters(pls); err != nil {
		log.Error(r.ctx, "Error updating smart playlist stats", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	// Reuse the stamp refreshCounters just wrote, so evaluated_at and updated_at agree
	now := pls.UpdatedAt
	updSql := Update(r.tableName).Set("evaluated_at", now).Where(Eq{"id": pls.ID})
	if _, err = r.executeSQL(updSql); err != nil {
		log.Error(r.ctx, "Error updating smart playlist", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}
	pls.EvaluatedAt = &now

	log.Debug(r.ctx, "Refreshed playlist", "playlist", pls.Name, "id", pls.ID, "numTracks", pls.SongCount, "elapsed", time.Since(start))
	return true
}

// shouldRefreshSmartPlaylist determines if a smart playlist needs to be refreshed based on its type, last evaluated
// time, and ownership.
func (r *playlistRepository) shouldRefreshSmartPlaylist(pls *model.Playlist, usr *model.User) bool {
	if !pls.IsSmartPlaylist() {
		return false
	}
	if pls.EvaluatedAt != nil && time.Since(*pls.EvaluatedAt) < pls.RefreshDelay() {
		return false
	}
	if pls.OwnerID != usr.ID {
		log.Trace(r.ctx, "Not refreshing smart playlist from other user", "playlist", pls.Name, "id", pls.ID)
		return false
	}
	return true
}

// refreshChildPlaylists handles refreshing any child playlists that are referenced in the smart playlist criteria.
// Returns false if child playlists could not be loaded (DB error), signaling the parent refresh should abort.
func (r *playlistRepository) refreshChildPlaylists(pls *model.Playlist, rulesSQL smartPlaylistCriteria) bool {
	childPlaylistIds := rulesSQL.ChildPlaylistIds()
	childPlaylistPaths := rulesSQL.ChildPlaylistPaths()
	if len(childPlaylistIds) == 0 || len(childPlaylistPaths) == 0 {
		return true
	}

	childPlaylists, err := r.GetAll(model.QueryOptions{Filters: Or{Eq{"playlist.id": childPlaylistIds}, Eq{"playlist.path": childPlaylistPaths}}})
	if err != nil {
		log.Error(r.ctx, "Error loading child playlists for smart playlist refresh", "playlist", pls.Name, "id", pls.ID, "childIds", childPlaylistIds, err)
		return false
	}

	found := make(map[string]struct{}, len(childPlaylists)*2)
	for i := range childPlaylists {
		found[childPlaylists[i].ID] = struct{}{}
		found[childPlaylists[i].Path] = struct{}{}
		r.refreshSmartPlaylist(&childPlaylists[i])
	}
	for _, id := range childPlaylistIds {
		if _, ok := found[id]; !ok {
			log.Warn(r.ctx, "Referenced playlist is not accessible to smart playlist owner", "playlist", pls.Name, "id", pls.ID, "childId", id, "ownerId", pls.OwnerID)
		}
	}

	for _, path := range childPlaylistPaths {
		if _, ok := found[path]; !ok {
			log.Warn(r.ctx, "Referenced playlist is not accessible to smart playlist owner", "playlist", pls.Name, "id", pls.ID, "path", path, "ownerId", pls.OwnerID)
		}
	}
	return true
}

// resolvePercentageLimit calculates the actual limit for a smart playlist criteria that uses a percentage-based limit.
func (r *playlistRepository) resolvePercentageLimit(pls *model.Playlist, rulesSQL *smartPlaylistCriteria, userID string) error {
	if !rulesSQL.IsPercentageLimit() {
		return nil
	}

	countSq := Select("count(*) as count").From("media_file")
	countSq = rulesSQL.applyExpressionJoins(countSq, userID)
	countSq = r.applyLibraryFilter(countSq, "media_file")

	cond, err := rulesSQL.where()
	if err != nil {
		log.Error(r.ctx, "Error building smart playlist criteria", "playlist", pls.Name, "id", pls.ID, err)
		return err
	}
	countSq = countSq.Where(cond)

	var res struct{ Count int64 }
	if err = r.queryOne(countSq, &res); err != nil {
		log.Error(r.ctx, "Error counting matching tracks for percentage limit", "playlist", pls.Name, "id", pls.ID, err)
		return err
	}

	rulesSQL.ResolveLimit(res.Count)
	log.Debug(r.ctx, "Resolved percentage limit", "playlist", pls.Name, "percent", rulesSQL.LimitPercent, "totalMatching", res.Count, "resolvedLimit", rulesSQL.Limit)
	return nil
}

// buildSmartPlaylistQuery constructs the SQL query to select media files matching the smart playlist criteria,
// including the joins its fields require and library filtering.
func (r *playlistRepository) buildSmartPlaylistQuery(pls *model.Playlist, rulesSQL smartPlaylistCriteria, userID string) SelectBuilder {
	orderBy := rulesSQL.orderBy()
	sq := Select("row_number() over (order by "+orderBy+") as id", "'"+pls.ID+"' as playlist_id", "media_file.id as media_file_id").
		From("media_file")
	sq = rulesSQL.applyRequiredJoins(sq, userID)
	sq = r.applyLibraryFilter(sq, "media_file")
	return sq
}

// addCriteria applies the where conditions, limit, offset, and order by clauses to the SQL query based on the
// smart playlist criteria.
func (r *playlistRepository) addCriteria(sql SelectBuilder, cSQL smartPlaylistCriteria) (SelectBuilder, error) {
	cond, err := cSQL.where()
	if err != nil {
		return sql, err
	}
	sql = sql.Where(cond)
	if cSQL.Criteria.Limit > 0 {
		sql = sql.Limit(uint64(cSQL.Criteria.Limit)).Offset(uint64(cSQL.Criteria.Offset))
	}
	if order := cSQL.orderBy(); order != "" {
		sql = sql.OrderBy(order)
	}
	return sql, nil
}
