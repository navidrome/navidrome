package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

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

// BenchmarkSmartPlaylistRole compares role-based smart playlist query performance across
// three strategies: merged join-table, merged json_tree, and unmerged json_tree (baseline).
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

	patterns := make([]string, benchNumPatterns)
	for i := range benchNumPatterns {
		patterns[i] = fmt.Sprintf("%%Artist %04d%%", i)
	}

	b.Run("MergedJoinTable", func(b *testing.B) {
		benchmarkMergedJoinTable(b, ctx, patterns)
	})
	b.Run("MergedJSONTree", func(b *testing.B) {
		benchmarkMergedJSONTree(b, ctx, patterns)
	})
	b.Run("UnmergedJSONTree", func(b *testing.B) {
		benchmarkUnmergedJSONTree(b, ctx, patterns)
	})
}

// benchmarkMergedJoinTable: batched EXISTS with patterns ORed inside each batch, using media_file_artists join.
func benchmarkMergedJoinTable(b *testing.B, ctx context.Context, patterns []string) {
	b.Helper()

	var sb strings.Builder
	var args []any
	sb.WriteString("SELECT media_file.id FROM media_file WHERE (")
	for start := 0; start < len(patterns); start += jsonCondBatchSize {
		end := min(start+jsonCondBatchSize, len(patterns))
		batch := patterns[start:end]
		if start > 0 {
			sb.WriteString(" OR ")
		}
		sb.WriteString("exists (select 1 from media_file_artists mfa join artist on artist.id = mfa.artist_id where mfa.media_file_id = media_file.id and mfa.role = ? and (")
		args = append(args, "artist")
		for i, p := range batch {
			if i > 0 {
				sb.WriteString(" OR ")
			}
			sb.WriteString("artist.name LIKE ?")
			args = append(args, p)
		}
		sb.WriteString("))")
	}
	sb.WriteString(") ORDER BY media_file.title LIMIT 500")

	runBenchQuery(b, ctx, sb.String(), args)
}

// benchmarkMergedJSONTree: batched EXISTS with patterns ORed inside each batch, using json_tree.
func benchmarkMergedJSONTree(b *testing.B, ctx context.Context, patterns []string) {
	b.Helper()

	var sb strings.Builder
	var args []any
	sb.WriteString("SELECT media_file.id FROM media_file WHERE (")
	for start := 0; start < len(patterns); start += jsonCondBatchSize {
		end := min(start+jsonCondBatchSize, len(patterns))
		batch := patterns[start:end]
		if start > 0 {
			sb.WriteString(" OR ")
		}
		sb.WriteString("exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name' and (")
		for i, p := range batch {
			if i > 0 {
				sb.WriteString(" OR ")
			}
			sb.WriteString("value LIKE ?")
			args = append(args, p)
		}
		sb.WriteString("))")
	}
	sb.WriteString(") ORDER BY media_file.title LIMIT 500")

	runBenchQuery(b, ctx, sb.String(), args)
}

// benchmarkUnmergedJSONTree: N separate EXISTS subqueries (the old baseline approach).
func benchmarkUnmergedJSONTree(b *testing.B, ctx context.Context, patterns []string) {
	b.Helper()

	var sb strings.Builder
	sb.WriteString("SELECT media_file.id FROM media_file WHERE (")
	args := make([]any, 0, len(patterns))
	for i, p := range patterns {
		if i > 0 {
			sb.WriteString(" OR ")
		}
		sb.WriteString("exists (select 1 from json_tree(media_file.participants, '$.artist') where key='name' and value LIKE ?)")
		args = append(args, p)
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
		if err := rows.Err(); err != nil {
			b.Fatal(err)
		}
		rows.Close()
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
