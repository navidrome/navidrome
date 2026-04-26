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

func (r *playlistRepository) refreshSmartPlaylist(pls *model.Playlist) bool {
	// Only refresh if it is a smart playlist and was not refreshed within the interval provided by the refresh delay config
	if !pls.IsSmartPlaylist() || (pls.EvaluatedAt != nil && time.Since(*pls.EvaluatedAt) < conf.Server.SmartPlaylistRefreshDelay) {
		return false
	}

	// Never refresh other users' playlists
	usr := loggedUser(r.ctx)
	if pls.OwnerID != usr.ID {
		log.Trace(r.ctx, "Not refreshing smart playlist from other user", "playlist", pls.Name, "id", pls.ID)
		return false
	}

	log.Debug(r.ctx, "Refreshing smart playlist", "playlist", pls.Name, "id", pls.ID)
	start := time.Now()

	// Remove old tracks
	del := Delete("playlist_tracks").Where(Eq{"playlist_id": pls.ID})
	_, err := r.executeSQL(del)
	if err != nil {
		log.Error(r.ctx, "Error deleting old smart playlist tracks", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	// Re-populate playlist based on Smart Playlist criteria
	rulesSQL := newSmartPlaylistCriteria(*pls.Rules, withSmartPlaylistOwner(pls.OwnerID, usr.IsAdmin))

	// If the playlist depends on other playlists, recursively refresh them first
	childPlaylistIds := rulesSQL.ChildPlaylistIds()
	if len(childPlaylistIds) > 0 {
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
	}

	orderBy := rulesSQL.OrderBy()
	sq := Select("row_number() over (order by "+orderBy+") as id", "'"+pls.ID+"' as playlist_id", "media_file.id as media_file_id").
		From("media_file").LeftJoin("annotation on ("+
		"annotation.item_id = media_file.id"+
		" AND annotation.item_type = 'media_file'"+
		" AND annotation.user_id = ?)", usr.ID)

	// Conditionally join album/artist annotation tables only when referenced by criteria or sort
	requiredJoins := rulesSQL.RequiredJoins()
	sq = r.addSmartPlaylistAnnotationJoins(sq, requiredJoins, usr.ID)

	// Only include media files from libraries the user has access to
	sq = r.applyLibraryFilter(sq, "media_file")

	// Resolve percentage-based limit to an absolute number before applying criteria
	if rulesSQL.IsPercentageLimit() {
		// Use only expression-based joins for the COUNT query (sort joins are unnecessary)
		exprJoins := rulesSQL.ExpressionJoins()
		countSq := Select("count(*) as count").From("media_file").
			LeftJoin("annotation on ("+
				"annotation.item_id = media_file.id"+
				" AND annotation.item_type = 'media_file'"+
				" AND annotation.user_id = ?)", usr.ID)
		countSq = r.addSmartPlaylistAnnotationJoins(countSq, exprJoins, usr.ID)
		countSq = r.applyLibraryFilter(countSq, "media_file")
		cond, err := rulesSQL.Where()
		if err != nil {
			log.Error(r.ctx, "Error building smart playlist criteria", "playlist", pls.Name, "id", pls.ID, err)
			return false
		}
		countSq = countSq.Where(cond)

		var res struct{ Count int64 }
		err = r.queryOne(countSq, &res)
		if err != nil {
			log.Error(r.ctx, "Error counting matching tracks for percentage limit", "playlist", pls.Name, "id", pls.ID, err)
			return false
		}
		rulesSQL.ResolveLimit(res.Count)
		log.Debug(r.ctx, "Resolved percentage limit", "playlist", pls.Name, "percent", rulesSQL.LimitPercent, "totalMatching", res.Count, "resolvedLimit", rulesSQL.Limit)
	}

	// Apply the criteria rules
	sq, err = r.addCriteria(sq, rulesSQL)
	if err != nil {
		log.Error(r.ctx, "Error building smart playlist criteria", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}
	insSql := Insert("playlist_tracks").Columns("id", "playlist_id", "media_file_id").Select(sq)
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
	now := time.Now()
	updSql := Update(r.tableName).Set("evaluated_at", now).Where(Eq{"id": pls.ID})
	_, err = r.executeSQL(updSql)
	if err != nil {
		log.Error(r.ctx, "Error updating smart playlist", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	pls.EvaluatedAt = &now

	log.Debug(r.ctx, "Refreshed playlist", "playlist", pls.Name, "id", pls.ID, "numTracks", pls.SongCount, "elapsed", time.Since(start))

	return true
}

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
