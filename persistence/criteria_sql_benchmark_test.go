package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/pocketbase/dbx"
)

const (
	benchNumArtists      = 1_000
	benchNumTracks       = 40_000
	benchNumPatterns     = 500
	benchArtistsPerTrack = 3
)

// BenchmarkSmartPlaylistRole compares role-based smart playlist query performance
// between the current implementation (merged join-table via criteria pipeline) and
// the old baseline (unmerged json_tree subqueries).
func BenchmarkSmartPlaylistRole(b *testing.B) {
	configtest.SetupConfig()
	tmpDir := b.TempDir()
	conf.Server.DbPath = filepath.Join(tmpDir, "bench-smartpl.db")
	cleanup := db.Init(context.Background())
	defer cleanup()
	log.SetLevel(log.LevelFatal)

	conn := dbx.NewFromDB(db.Db(), db.Dialect)
	ctx := log.NewContext(context.Background())
	user := model.User{ID: "bench-user", UserName: "bench", Name: "Bench User", IsAdmin: true}
	ctx = request.WithUser(ctx, user)

	setupBenchData(b, ctx, conn, user)
	criteria.AddRoles([]string{"artist"})

	// Build the criteria expression: 500 "contains artist" patterns in an OR group
	anyExprs := make(criteria.Any, benchNumPatterns)
	for i := range benchNumPatterns {
		anyExprs[i] = criteria.Contains{"artist": fmt.Sprintf("Artist %04d", i)}
	}
	expr := criteria.Criteria{Expression: anyExprs, Sort: "title", Limit: 500}

	b.Run("Current", func(b *testing.B) {
		benchmarkCriteriaPipeline(b, ctx, expr)
	})
	b.Run("Baseline_UnmergedJSONTree", func(b *testing.B) {
		benchmarkUnmergedJSONTree(b, ctx)
	})
}

// BenchmarkSmartPlaylistNegatedRole compares performance for smart playlists with many
// negated role conditions ANDed together (e.g. 500 "isNot artist" rules, issue #5511)
// between the current implementation (merged NOT EXISTS via criteria pipeline) and the
// old baseline (one separate NOT EXISTS subquery per pattern).
func BenchmarkSmartPlaylistNegatedRole(b *testing.B) {
	configtest.SetupConfig()
	tmpDir := b.TempDir()
	conf.Server.DbPath = filepath.Join(tmpDir, "bench-smartpl-neg.db")
	cleanup := db.Init(context.Background())
	defer cleanup()
	log.SetLevel(log.LevelFatal)

	conn := dbx.NewFromDB(db.Db(), db.Dialect)
	ctx := log.NewContext(context.Background())
	user := model.User{ID: "bench-user", UserName: "bench", Name: "Bench User", IsAdmin: true}
	ctx = request.WithUser(ctx, user)

	setupBenchData(b, ctx, conn, user)
	criteria.AddRoles([]string{"artist"})

	// Build the criteria expression: 500 "isNot artist" patterns in an AND group
	allExprs := make(criteria.All, benchNumPatterns)
	for i := range benchNumPatterns {
		allExprs[i] = criteria.IsNot{"artist": fmt.Sprintf("Artist %04d", i)}
	}
	expr := criteria.Criteria{Expression: allExprs, Sort: "title", Limit: 500}

	b.Run("Current", func(b *testing.B) {
		benchmarkCriteriaPipeline(b, ctx, expr)
	})
	b.Run("Baseline_UnmergedNotExists", func(b *testing.B) {
		benchmarkUnmergedNegatedJSONTree(b, ctx)
	})
}

// benchmarkUnmergedNegatedJSONTree builds the old-style query with N separate negated
// json_tree EXISTS subqueries ANDed together (the pre-optimization baseline).
func benchmarkUnmergedNegatedJSONTree(b *testing.B, ctx context.Context) {
	b.Helper()

	var sb strings.Builder
	sb.WriteString("SELECT media_file.id FROM media_file WHERE (")
	args := make([]any, 0, benchNumPatterns)
	for i := range benchNumPatterns {
		if i > 0 {
			sb.WriteString(" AND ")
		}
		sb.WriteString("not exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name' and value = ?)")
		args = append(args, fmt.Sprintf("Artist %04d", i))
	}
	sb.WriteString(") ORDER BY media_file.title LIMIT 500")

	runBenchQuery(b, ctx, sb.String(), args)
}

// benchmarkCriteriaPipeline runs the criteria through the actual production code path:
// newSmartPlaylistCriteria → Where() → ToSql(), then executes the resulting query.
func benchmarkCriteriaPipeline(b *testing.B, ctx context.Context, expr criteria.Criteria) {
	b.Helper()

	cSQL := newSmartPlaylistCriteria(expr)

	// Build the full query matching buildSmartPlaylistQuery + addCriteria
	sq := squirrel.Select("media_file.id").From("media_file")
	cond, err := cSQL.Where()
	if err != nil {
		b.Fatal(err)
	}
	sq = sq.Where(cond)
	if expr.Limit > 0 {
		sq = sq.Limit(uint64(expr.Limit))
	}
	if order := cSQL.OrderBy(); order != "" {
		sq = sq.OrderBy(order)
	}

	query, args, err := sq.PlaceholderFormat(squirrel.Question).ToSql()
	if err != nil {
		b.Fatal(err)
	}

	runBenchQuery(b, ctx, query, args)
}

// benchmarkUnmergedJSONTree builds the old-style query with N separate json_tree EXISTS
// subqueries (the pre-optimization baseline).
func benchmarkUnmergedJSONTree(b *testing.B, ctx context.Context) {
	b.Helper()

	var sb strings.Builder
	sb.WriteString("SELECT media_file.id FROM media_file WHERE (")
	args := make([]any, 0, benchNumPatterns)
	for i := range benchNumPatterns {
		if i > 0 {
			sb.WriteString(" OR ")
		}
		sb.WriteString("exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name' and value LIKE ?)")
		args = append(args, fmt.Sprintf("%%Artist %04d%%", i))
	}
	sb.WriteString(") ORDER BY media_file.title LIMIT 500")

	runBenchQuery(b, ctx, sb.String(), args)
}

func runBenchQuery(b *testing.B, ctx context.Context, query string, args []any) {
	b.Helper()
	sqlDB := db.Db()
	b.ResetTimer()
	for range b.N {
		rows, err := sqlDB.QueryContext(ctx, query, args...)
		if err != nil {
			b.Fatal(err)
		}
		for rows.Next() {
			var id string
			_ = rows.Scan(&id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			b.Fatal(err)
		}
	}
}

func setupBenchData(b *testing.B, ctx context.Context, conn *dbx.DB, user model.User) {
	b.Helper()

	sqlDB := db.Db()

	ur := NewUserRepository(ctx, conn)
	if err := ur.Put(&user); err != nil {
		b.Fatal(err)
	}
	if err := ur.SetUserLibraries(user.ID, []int{1}); err != nil {
		b.Fatal(err)
	}

	tx, err := sqlDB.Begin()
	if err != nil {
		b.Fatal(err)
	}

	// Create artists
	artistStmt, err := tx.Prepare("INSERT INTO artist (id, name) VALUES (?, ?)")
	if err != nil {
		b.Fatal(err)
	}
	for i := range benchNumArtists {
		if _, err := artistStmt.Exec(fmt.Sprintf("artist-%04d", i), fmt.Sprintf("Artist %04d", i)); err != nil {
			b.Fatal(err)
		}
	}
	artistStmt.Close()

	// Ensure folder exists
	folderID := "bench-folder"
	if _, err := tx.Exec("INSERT OR IGNORE INTO folder (id, library_id, path, name, parent_id) VALUES (?, 1, '.', '.', '')", folderID); err != nil {
		b.Fatal(err)
	}

	// Create media files with participants JSON, cycling through artists
	mfStmt, err := tx.Prepare(`INSERT INTO media_file (id, path, title, album, artist, artist_id, album_id,
		duration, year, size, suffix, tags, participants, lyrics, library_id, folder_id, pid, codec)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		b.Fatal(err)
	}

	// Populate media_file_artists join table
	mfaStmt, err := tx.Prepare("INSERT INTO media_file_artists (media_file_id, artist_id, role, sub_role) VALUES (?, ?, ?, ?)")
	if err != nil {
		b.Fatal(err)
	}

	for i := range benchNumTracks {
		trackID := fmt.Sprintf("track-%05d", i)

		// Assign benchArtistsPerTrack artists to each track, cycling through the pool
		artistEntries := make([]map[string]string, benchArtistsPerTrack)
		for a := range benchArtistsPerTrack {
			artistIdx := (i + a) % benchNumArtists
			artistEntries[a] = map[string]string{
				"id":   fmt.Sprintf("artist-%04d", artistIdx),
				"name": fmt.Sprintf("Artist %04d", artistIdx),
			}
		}
		primaryArtistIdx := i % benchNumArtists
		primaryArtistID := fmt.Sprintf("artist-%04d", primaryArtistIdx)
		primaryArtistName := fmt.Sprintf("Artist %04d", primaryArtistIdx)

		participants := map[string][]map[string]string{"artist": artistEntries}
		participantsJSON, _ := json.Marshal(participants)

		if _, err := mfStmt.Exec(
			trackID,
			fmt.Sprintf("music/%s.mp3", trackID),
			fmt.Sprintf("Track %05d", i),
			"Bench Album",
			primaryArtistName,
			primaryArtistID,
			"bench-album",
			180, 2024, 5000000, "mp3",
			"{}",
			string(participantsJSON),
			"[]",
			1, folderID, trackID, "mp3",
		); err != nil {
			b.Fatal(err)
		}

		// Insert all artist associations into the join table
		for a := range benchArtistsPerTrack {
			artistIdx := (i + a) % benchNumArtists
			artistID := fmt.Sprintf("artist-%04d", artistIdx)
			if _, err := mfaStmt.Exec(trackID, artistID, "artist", ""); err != nil {
				b.Fatal(err)
			}
		}
	}
	mfStmt.Close()
	mfaStmt.Close()

	if err := tx.Commit(); err != nil {
		b.Fatal(err)
	}

	b.Logf("Setup complete: %d artists, %d tracks (%d artists/track), %d patterns",
		benchNumArtists, benchNumTracks, benchArtistsPerTrack, benchNumPatterns)
}
