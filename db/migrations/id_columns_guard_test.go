package migrations

import (
	"context"
	"database/sql"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Version of the uniform-canonical-ids migration whose idColumns list this guard protects.
const idColumnsMigrationVersion = 20270101010123

var _ = Describe("idColumns inventory", func() {
	It("covers every id-bearing column present at the migration version", func() {
		ctx := context.Background()
		db, err := sql.Open("sqlite3", "file::memory:")
		Expect(err).ToNot(HaveOccurred())
		db.SetMaxOpenConns(1) // non-shared :memory: — a second conn would be an empty DB
		DeferCleanup(func() { _ = db.Close() })
		_, err = db.ExecContext(ctx, "PRAGMA foreign_keys=off")
		Expect(err).ToNot(HaveOccurred())

		goose.SetBaseFS(os.DirFS("."))
		DeferCleanup(func() { goose.SetBaseFS(nil) })
		Expect(goose.SetDialect("sqlite3")).To(Succeed())
		Expect(goose.UpToContext(ctx, db, ".", idColumnsMigrationVersion)).To(Succeed())

		covered := map[string]bool{}
		for _, tc := range idColumns {
			covered[tc.table+"."+tc.col] = true
		}
		for _, tc := range embeddedIDColumns {
			covered[tc.table+"."+tc.col] = true
		}
		// Columns that are id-named or JSON but need no rewrite.
		exempt := map[string]string{
			"share.id":                       "public share URLs, generated separately",
			"property.id":                    "property key, not an entity id",
			"plugin.id":                      "plugin name, not an entity id",
			"media_file.participants":        "hash-family artist ids, unchanged",
			"album.participants":             "hash-family artist ids, unchanged",
			"media_file.tags":                "hash-family tag ids, unchanged",
			"album.tags":                     "hash-family tag ids, unchanged",
			"album.folder_ids":               "hash-family folder ids, unchanged",
			"artist.similar_artists":         "hash-family artist ids, unchanged",
			"media_file.search_participants": "participant names for FTS, not ids",
			"album.search_participants":      "participant names for FTS, not ids",
			"album.discs":                    "disc number -> title map",
			"media_file.lyrics":              "synced lyrics, no ids",
			"folder.image_files":             "image file names, no ids",
			"plugin.manifest":                "plugin-authored manifest, no Navidrome ids",
			"plugin.config":                  "free-form plugin config, must not be rewritten",
			"plugin.libraries":               "integer library ids",
		}

		tables, err := queryColumn(ctx, db, "SELECT name FROM sqlite_master WHERE type='table'")
		Expect(err).ToNot(HaveOccurred())

		var dangling []string
		for _, table := range tables {
			if strings.HasPrefix(table, "sqlite_") || table == "goose_db_version" || strings.Contains(table, "_fts") {
				continue
			}
			rows, err := db.QueryContext(ctx, "SELECT name, type FROM pragma_table_info(?)", table)
			Expect(err).ToNot(HaveOccurred())
			for rows.Next() {
				var name, typ string
				Expect(rows.Scan(&name, &typ)).To(Succeed())
				lname := strings.ToLower(name)
				utyp := strings.ToUpper(typ)
				// JSON can hide an id under any key, so every JSON column needs a verdict.
				isJSON := strings.Contains(utyp, "JSON")
				isIDName := lname == "id" || lname == "pid" ||
					strings.HasSuffix(lname, "_id") || strings.HasSuffix(lname, "_ids")
				if !isJSON && !isIDName {
					continue
				}
				if !isJSON && !strings.Contains(utyp, "TEXT") && !strings.Contains(utyp, "CHAR") {
					continue // INTEGER ids (rowid PKs, library.id) are not canonical ids
				}
				if strings.HasPrefix(lname, "mbz_") {
					continue // MusicBrainz UUIDs, not Navidrome ids
				}
				key := table + "." + name
				if covered[key] || exempt[key] != "" {
					continue
				}
				dangling = append(dangling, key)
			}
			Expect(rows.Err()).ToNot(HaveOccurred())
			Expect(rows.Close()).To(Succeed())
		}
		Expect(dangling).To(BeEmpty(),
			"id-bearing columns missing from idColumns; add them to the migration or exempt with a reason: %v", dangling)
	})
})

func queryColumn(ctx context.Context, db *sql.DB, query string) ([]string, error) {
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
