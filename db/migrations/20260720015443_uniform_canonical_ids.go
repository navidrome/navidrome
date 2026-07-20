package migrations

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(upUniformCanonicalIds, downUniformCanonicalIds)
}

// idColumns is the exhaustive list of Navidrome-id-bearing columns; anything absent is deliberately exempt.
var idColumns = []struct{ table, col string }{
	{"media_file", "id"}, {"media_file", "pid"}, {"media_file", "artist_id"},
	{"media_file", "album_id"}, {"media_file", "album_artist_id"}, {"media_file", "folder_id"},
	{"album", "id"}, {"album", "album_artist_id"},
	{"artist", "id"},
	{"folder", "id"}, {"folder", "parent_id"},
	{"tag", "id"},
	{"library_artist", "artist_id"},
	{"user", "id"},
	{"user_props", "user_id"},
	{"playlist", "id"}, {"playlist", "owner_id"},
	{"playlist_tracks", "playlist_id"}, {"playlist_tracks", "media_file_id"},
	{"playlist_fields", "playlist_id"},
	{"annotation", "user_id"}, {"annotation", "item_id"},
	{"bookmark", "user_id"}, {"bookmark", "item_id"},
	{"player", "id"}, {"player", "user_id"}, {"player", "transcoding_id"},
	{"transcoding", "id"},
	{"radio", "id"},
	{"share", "user_id"},
	{"scrobble_buffer", "id"}, {"scrobble_buffer", "user_id"}, {"scrobble_buffer", "media_file_id"},
	{"playqueue", "id"}, {"playqueue", "user_id"},
	{"user_library", "user_id"},
	{"scrobbles", "user_id"}, {"scrobbles", "media_file_id"},
	{"media_file_artists", "media_file_id"}, {"media_file_artists", "artist_id"},
	{"album_artists", "album_id"}, {"album_artists", "artist_id"},
	{"library_tag", "tag_id"},
}

func upUniformCanonicalIds(ctx context.Context, tx *sql.Tx) error {
	if err := buildIDMap(ctx, tx); err != nil {
		return err
	}
	for _, tc := range idColumns {
		if err := applyIDMap(ctx, tx, tc.table, tc.col); err != nil {
			return fmt.Errorf("canonicalizing %s.%s: %w", tc.table, tc.col, err)
		}
	}
	if err := rewriteListColumn(ctx, tx, "playqueue", "items"); err != nil {
		return err
	}
	if err := rewriteListColumn(ctx, tx, "share", "resource_ids"); err != nil {
		return err
	}
	if err := rewriteJSONColumn(ctx, tx, "plugin", "users", canonicalizePluginUsers); err != nil {
		return err
	}
	if err := rewriteJSONColumn(ctx, tx, "playlist", "rules", canonicalizePlaylistRules); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, "DROP TABLE _id_map")
	return err
}

// buildIDMap stages old->new pairs for every id that changes, indexed for the update joins.
func buildIDMap(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx,
		"CREATE TEMP TABLE _id_map (old_id TEXT PRIMARY KEY, new_id TEXT NOT NULL) WITHOUT ROWID")
	if err != nil {
		return err
	}
	ins, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO _id_map (old_id, new_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer ins.Close()
	for _, tc := range idColumns {
		if err := collectColumn(ctx, tx, ins, tc.table, tc.col); err != nil {
			return fmt.Errorf("collecting %s.%s: %w", tc.table, tc.col, err)
		}
	}
	return nil
}

func collectColumn(ctx context.Context, tx *sql.Tx, ins *sql.Stmt, table, col string) error {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(
		"SELECT DISTINCT %[2]s FROM %[1]s WHERE %[2]s IS NOT NULL", table, col))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var old string
		if err := rows.Scan(&old); err != nil {
			return err
		}
		if newID := canonicalID(old); newID != old {
			if _, err := ins.ExecContext(ctx, old, newID); err != nil {
				return err
			}
		}
	}
	return rows.Err()
}

func applyIDMap(ctx context.Context, tx *sql.Tx, table, col string) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(
		`UPDATE %[1]s SET %[2]s = (SELECT new_id FROM _id_map WHERE old_id = %[1]s.%[2]s)
		 WHERE %[2]s IN (SELECT old_id FROM _id_map)`, table, col))
	return err
}

// rewriteListColumn canonicalizes comma-separated id lists element-wise.
func rewriteListColumn(ctx context.Context, tx *sql.Tx, table, col string) error {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(
		"SELECT rowid, %[2]s FROM %[1]s WHERE ifnull(%[2]s, '') <> ''", table, col))
	if err != nil {
		return err
	}
	type change struct {
		rowid int64
		val   string
	}
	var changes []change
	for rows.Next() {
		var rowid int64
		var val string
		if err := rows.Scan(&rowid, &val); err != nil {
			_ = rows.Close()
			return err
		}
		parts := strings.Split(val, ",")
		changed := false
		for i, p := range parts {
			if n := canonicalID(p); n != p {
				parts[i] = n
				changed = true
			}
		}
		if changed {
			changes = append(changes, change{rowid, strings.Join(parts, ",")})
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()
	for _, c := range changes {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(
			"UPDATE %s SET %s = ? WHERE rowid = ?", table, col), c.val, c.rowid); err != nil {
			return err
		}
	}
	return nil
}

// rewriteJSONColumn applies transform to each non-empty JSON cell, updating only rows it changed.
func rewriteJSONColumn(ctx context.Context, tx *sql.Tx, table, col string, transform func(string) (string, bool)) error {
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(
		"SELECT rowid, %[2]s FROM %[1]s WHERE ifnull(%[2]s, '') <> ''", table, col))
	if err != nil {
		return err
	}
	type change struct {
		rowid int64
		val   string
	}
	var changes []change
	for rows.Next() {
		var rowid int64
		var val string
		if err := rows.Scan(&rowid, &val); err != nil {
			_ = rows.Close()
			return err
		}
		if newVal, ok := transform(val); ok {
			changes = append(changes, change{rowid, newVal})
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()
	for _, c := range changes {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(
			"UPDATE %s SET %s = ? WHERE rowid = ?", table, col), c.val, c.rowid); err != nil {
			return err
		}
	}
	return nil
}

// canonicalizePluginUsers maps a JSON array of user ids; malformed JSON passes through untouched.
func canonicalizePluginUsers(s string) (string, bool) {
	var users []string
	if err := json.Unmarshal([]byte(s), &users); err != nil {
		return s, false
	}
	changed := false
	for i, u := range users {
		if n := canonicalID(u); n != u {
			users[i] = n
			changed = true
		}
	}
	if !changed {
		return s, false
	}
	out, err := json.Marshal(users)
	if err != nil {
		return s, false
	}
	return string(out), true
}

// canonicalizePlaylistRules rewrites inPlaylist/notInPlaylist ids in smart-playlist criteria; malformed JSON passes through.
func canonicalizePlaylistRules(s string) (string, bool) {
	var root map[string]any
	if err := json.Unmarshal([]byte(s), &root); err != nil {
		return s, false
	}
	if !canonicalizeRulesNode(root) {
		return s, false
	}
	out, err := json.Marshal(root)
	if err != nil {
		return s, false
	}
	return string(out), true
}

// canonicalizeRulesNode walks the criteria tree, canonicalizing the id of any inPlaylist/notInPlaylist object.
func canonicalizeRulesNode(node any) bool {
	changed := false
	switch v := node.(type) {
	case map[string]any:
		for k, val := range v {
			if lk := strings.ToLower(k); lk == "inplaylist" || lk == "notinplaylist" {
				if obj, ok := val.(map[string]any); ok {
					if id, ok := obj["id"].(string); ok {
						if n := canonicalID(id); n != id {
							obj["id"] = n
							changed = true
						}
					}
				}
			}
			if canonicalizeRulesNode(val) {
				changed = true
			}
		}
	case []any:
		for _, item := range v {
			if canonicalizeRulesNode(item) {
				changed = true
			}
		}
	}
	return changed
}

func downUniformCanonicalIds(ctx context.Context, tx *sql.Tx) error {
	return nil // irreversible data migration
}
