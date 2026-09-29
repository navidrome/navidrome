package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/text/unicode/norm"
)

// PlaylistRepository methods to handle smart playlists, which are defined by criteria and automatically populated
// based on their rules. The main method is refreshSmartPlaylist, which evaluates the criteria and updates the playlist
// tracks accordingly. It also handles refreshing dependent playlists when a smart playlist references other playlists
// in its criteria. To optimize performance, it only refreshes when necessary based on the last evaluated time and
// configured refresh delay.

func (r *playlistRepository) Evaluate(ctx context.Context, id string) error {
	var res dbPlaylist
	if err := r.queryOne(ctx, r.selectPlaylist(ctx).Where(Eq{"playlist.id": id}), &res); err != nil {
		return err
	}
	pls := res.Playlist
	ownerCtx, err := r.ownerContext(ctx, pls.OwnerID)
	if err != nil {
		return err
	}
	pls.EvaluatedAt = nil
	if !r.refreshSmartPlaylist(ownerCtx, &pls) {
		return fmt.Errorf("evaluating smart playlist %s", id)
	}
	return nil
}

// refreshSmartPlaylist evaluates the criteria of a smart playlist and updates its tracks accordingly.
func (r *playlistRepository) refreshSmartPlaylist(ctx context.Context, pls *model.Playlist) bool {
	return r.refreshSmartPlaylistTree(ctx, pls, map[string]struct{}{})
}

// The visited set stops playlists that reference each other from recursing forever.
func (r *playlistRepository) refreshSmartPlaylistTree(ctx context.Context, pls *model.Playlist, visited map[string]struct{}) bool {
	if _, seen := visited[pls.ID]; seen {
		log.Trace(ctx, "Skipping already visited smart playlist", "playlist", pls.Name, "id", pls.ID)
		return false
	}
	visited[pls.ID] = struct{}{}

	usr := loggedUser(ctx)
	if !r.shouldRefreshSmartPlaylist(ctx, pls, usr) {
		return false
	}

	log.Debug(ctx, "Refreshing smart playlist", "playlist", pls.Name, "id", pls.ID)
	start := time.Now()

	rulesSQL := newSmartPlaylistCriteria(*pls.NormalizedRules(), withSmartPlaylistOwner(*usr))

	if !r.refreshChildPlaylists(ctx, pls, rulesSQL, visited) {
		return false
	}

	if err := r.resolvePercentageLimit(ctx, pls, &rulesSQL, usr.ID); err != nil {
		return false
	}

	sq, err := r.addCriteria(r.buildSmartPlaylistQuery(ctx, rulesSQL, usr.ID), rulesSQL)
	if err != nil {
		log.Error(ctx, "Error building smart playlist criteria", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	// Evaluate the criteria before writing, so the write lock is only held for the short replace below
	var ids []string
	if err = r.queryAllSlice(ctx, sq, &ids); err != nil && !errors.Is(err, model.ErrNotFound) {
		log.Error(ctx, "Error evaluating smart playlist criteria", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	err = r.inTx(func(tx *playlistRepository) error { return tx.replaceSmartPlaylistTracks(ctx, pls, ids) })
	if err != nil {
		log.Error(ctx, "Error refreshing smart playlist tracks", "playlist", pls.Name, "id", pls.ID, err)
		return false
	}

	log.Debug(ctx, "Refreshed playlist", "playlist", pls.Name, "id", pls.ID, "numTracks", pls.SongCount, "elapsed", time.Since(start))
	return true
}

func (r *playlistRepository) replaceSmartPlaylistTracks(ctx context.Context, pls *model.Playlist, ids []string) error {
	if _, err := r.executeSQL(ctx, Delete("playlist_tracks").Where(Eq{"playlist_id": pls.ID})); err != nil {
		return err
	}
	if len(ids) > 0 {
		idsJSON, err := json.Marshal(ids)
		if err != nil {
			return err
		}
		ins := Expr("INSERT INTO playlist_tracks (id, playlist_id, media_file_id) SELECT key + 1, ?, value FROM json_each(?)",
			pls.ID, string(idsJSON))
		if _, err = r.executeSQL(ctx, ins); err != nil {
			return err
		}
	}
	if err := r.refreshCounters(ctx, pls); err != nil {
		return err
	}
	// Reuse the stamp refreshCounters just wrote, so evaluated_at and updated_at agree
	now := pls.UpdatedAt
	if _, err := r.executeSQL(ctx, Update(r.tableName).Set("evaluated_at", now).Where(Eq{"id": pls.ID})); err != nil {
		return err
	}
	pls.EvaluatedAt = &now
	return nil
}

// shouldRefreshSmartPlaylist determines if a smart playlist needs to be refreshed based on its type, last evaluated
// time, and ownership.
func (r *playlistRepository) shouldRefreshSmartPlaylist(ctx context.Context, pls *model.Playlist, usr *model.User) bool {
	if !pls.IsSmartPlaylist() {
		return false
	}
	if pls.EvaluatedAt != nil && time.Since(*pls.EvaluatedAt) < pls.RefreshDelay() {
		return false
	}
	if pls.OwnerID != usr.ID {
		log.Trace(ctx, "Not refreshing smart playlist from other user", "playlist", pls.Name, "id", pls.ID)
		return false
	}
	return true
}

// refreshChildPlaylists handles refreshing any child playlists that are referenced in the smart playlist criteria.
// Returns false if child playlists could not be loaded (DB error), signaling the parent refresh should abort.
func (r *playlistRepository) refreshChildPlaylists(ctx context.Context, pls *model.Playlist, rulesSQL smartPlaylistCriteria, visited map[string]struct{}) bool {
	childPlaylistIds := rulesSQL.ChildPlaylistIds()
	childPlaylistPaths := rulesSQL.ChildPlaylistPaths()
	if len(childPlaylistIds) == 0 && len(childPlaylistPaths) == 0 {
		return true
	}

	var conditions Or
	if len(childPlaylistIds) > 0 {
		conditions = append(conditions, Eq{"playlist.id": childPlaylistIds})
	}
	if len(childPlaylistPaths) > 0 {
		lookupPaths := slices.Concat(slice.Map(childPlaylistPaths, pathVariants)...)
		conditions = append(conditions, Eq{"playlist.path": lookupPaths})
	}

	childPlaylists, err := r.GetAll(ctx, model.QueryOptions{Filters: conditions})
	if err != nil {
		log.Error(ctx, "Error loading child playlists for smart playlist refresh", "playlist", pls.Name, "id", pls.ID, "childIds", childPlaylistIds, "childPaths", childPlaylistPaths, err)
		return false
	}

	found := make(map[string]struct{}, len(childPlaylists)*2)
	for i := range childPlaylists {
		found[childPlaylists[i].ID] = struct{}{}
		if childPlaylists[i].Path != "" {
			found[norm.NFC.String(childPlaylists[i].Path)] = struct{}{}
		}
		r.refreshSmartPlaylistTree(ctx, &childPlaylists[i], visited)
	}
	for _, id := range childPlaylistIds {
		if _, ok := found[id]; !ok {
			log.Warn(ctx, "Referenced playlist is not accessible to smart playlist owner", "playlist", pls.Name, "id", pls.ID, "childId", id, "ownerId", pls.OwnerID)
		}
	}

	for _, path := range childPlaylistPaths {
		if _, ok := found[norm.NFC.String(path)]; !ok {
			log.Warn(ctx, "Referenced playlist is not accessible to smart playlist owner", "playlist", pls.Name, "id", pls.ID, "path", path, "ownerId", pls.OwnerID)
		}
	}
	return true
}

// resolvePercentageLimit calculates the actual limit for a smart playlist criteria that uses a percentage-based limit.
func (r *playlistRepository) resolvePercentageLimit(ctx context.Context, pls *model.Playlist, rulesSQL *smartPlaylistCriteria, userID string) error {
	if !rulesSQL.IsPercentageLimit() {
		return nil
	}

	countSq := Select("count(*) as count").From("media_file")
	countSq = rulesSQL.applyExpressionJoins(countSq, userID)
	countSq = r.applyLibraryFilter(ctx, countSq, "media_file")

	cond, err := rulesSQL.where()
	if err != nil {
		log.Error(ctx, "Error building smart playlist criteria", "playlist", pls.Name, "id", pls.ID, err)
		return err
	}
	countSq = countSq.Where(cond)

	var res struct{ Count int64 }
	if err = r.queryOne(ctx, countSq, &res); err != nil {
		log.Error(ctx, "Error counting matching tracks for percentage limit", "playlist", pls.Name, "id", pls.ID, err)
		return err
	}

	rulesSQL.ResolveLimit(res.Count)
	log.Debug(ctx, "Resolved percentage limit", "playlist", pls.Name, "percent", rulesSQL.LimitPercent, "totalMatching", res.Count, "resolvedLimit", rulesSQL.Limit)
	return nil
}

// buildSmartPlaylistQuery constructs the SQL query to select the ids of media files matching the smart playlist
// criteria, including the joins its fields require and library filtering.
func (r *playlistRepository) buildSmartPlaylistQuery(ctx context.Context, rulesSQL smartPlaylistCriteria, userID string) SelectBuilder {
	sq := Select("media_file.id").From("media_file")
	sq = rulesSQL.applyRequiredJoins(sq, userID)
	sq = r.applyLibraryFilter(ctx, sq, "media_file")
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
