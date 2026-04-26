package persistence

import (
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
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

	now := time.Now()
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
	if pls.EvaluatedAt != nil && time.Since(*pls.EvaluatedAt) < conf.Server.SmartPlaylistRefreshDelay {
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
	if len(childPlaylistIds) == 0 {
		return true
	}

	childPlaylists, err := r.GetAll(model.QueryOptions{Filters: Eq{"playlist.id": childPlaylistIds}})
	if err != nil {
		log.Error(r.ctx, "Error loading child playlists for smart playlist refresh", "playlist", pls.Name, "id", pls.ID, "childIds", childPlaylistIds, err)
		return false
	}

	found := make(map[string]struct{}, len(childPlaylists))
	for i := range childPlaylists {
		found[childPlaylists[i].ID] = struct{}{}
		r.refreshSmartPlaylist(&childPlaylists[i])
	}
	for _, id := range childPlaylistIds {
		if _, ok := found[id]; !ok {
			log.Warn(r.ctx, "Referenced playlist is not accessible to smart playlist owner", "playlist", pls.Name, "id", pls.ID, "childId", id, "ownerId", pls.OwnerID)
		}
	}
	return true
}

// resolvePercentageLimit calculates the actual limit for a smart playlist criteria that uses a percentage-based limit.
func (r *playlistRepository) resolvePercentageLimit(pls *model.Playlist, rulesSQL *smartPlaylistCriteria, userID string) error {
	if !rulesSQL.IsPercentageLimit() {
		return nil
	}

	exprJoins := rulesSQL.ExpressionJoins()
	countSq := Select("count(*) as count").From("media_file")
	countSq = r.addMediaFileAnnotationJoin(countSq, userID)
	countSq = r.addSmartPlaylistAnnotationJoins(countSq, exprJoins, userID)
	countSq = r.applyLibraryFilter(countSq, "media_file")

	cond, err := rulesSQL.Where()
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
// including necessary joins for annotations and library filtering.
func (r *playlistRepository) buildSmartPlaylistQuery(pls *model.Playlist, rulesSQL smartPlaylistCriteria, userID string) SelectBuilder {
	orderBy := rulesSQL.OrderBy()
	sq := Select("row_number() over (order by "+orderBy+") as id", "'"+pls.ID+"' as playlist_id", "media_file.id as media_file_id").
		From("media_file")
	sq = r.addMediaFileAnnotationJoin(sq, userID)

	requiredJoins := rulesSQL.RequiredJoins()
	sq = r.addSmartPlaylistAnnotationJoins(sq, requiredJoins, userID)
	sq = r.applyLibraryFilter(sq, "media_file")
	return sq
}

// addMediaFileAnnotationJoin adds a left join to the annotation table for media files, filtering by user ID to include
// user-specific annotations in the smart playlist criteria evaluation.
func (r *playlistRepository) addMediaFileAnnotationJoin(sq SelectBuilder, userID string) SelectBuilder {
	return sq.LeftJoin("annotation on ("+
		"annotation.item_id = media_file.id"+
		" AND annotation.item_type = 'media_file'"+
		" AND annotation.user_id = ?)", userID)
}

// addSmartPlaylistAnnotationJoins adds left joins to the annotation table for albums and artists as needed based on
// the smart playlist criteria, filtering by user ID to include user-specific annotations in the evaluation.
func (r *playlistRepository) addSmartPlaylistAnnotationJoins(sq SelectBuilder, joins smartPlaylistJoinType, userID string) SelectBuilder {
	if joins.has(smartPlaylistJoinAlbumAnnotation) {
		sq = sq.LeftJoin("annotation AS album_annotation ON ("+
			"album_annotation.item_id = media_file.album_id"+
			" AND album_annotation.item_type = 'album'"+
			" AND album_annotation.user_id = ?)", userID)
	}
	if joins.has(smartPlaylistJoinArtistAnnotation) {
		sq = sq.LeftJoin("annotation AS artist_annotation ON ("+
			"artist_annotation.item_id = media_file.artist_id"+
			" AND artist_annotation.item_type = 'artist'"+
			" AND artist_annotation.user_id = ?)", userID)
	}
	return sq
}

// addCriteria applies the where conditions, limit, offset, and order by clauses to the SQL query based on the
// smart playlist criteria.
func (r *playlistRepository) addCriteria(sql SelectBuilder, cSQL smartPlaylistCriteria) (SelectBuilder, error) {
	cond, err := cSQL.Where()
	if err != nil {
		return sql, err
	}
	sql = sql.Where(cond)
	if cSQL.Criteria.Limit > 0 {
		sql = sql.Limit(uint64(cSQL.Criteria.Limit)).Offset(uint64(cSQL.Criteria.Offset))
	}
	if order := cSQL.OrderBy(); order != "" {
		sql = sql.OrderBy(order)
	}
	return sql, nil
}
