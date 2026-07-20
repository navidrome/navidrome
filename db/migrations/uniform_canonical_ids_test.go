package migrations

import (
	"context"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("upUniformCanonicalIds", func() {
	var db *sql.DB
	var tx *sql.Tx
	ctx := context.Background()

	const (
		hashID    = "5cLJPkLA5DK2BADhoeotPk" // canonical, kept
		randOld   = "zzzzzzzzzzzzzzzzzzzzzz" // overflows -> remapped
		randNew   = "3LyqmwQBm5IRqlVjNYASwb"
		legacyOld = "e3b7fc2ae9447bbec37a13bf916e3cf6" // 32-hex -> re-encoded
		legacyNew = "6VHl3uR4kss6sUPKA8Cwnk"
		uuidOld   = "f47ac10b-58cc-4372-a567-0e02b2c3d479" // uuid -> re-encoded
		uuidNew   = "7rke2SAWaicSeSYzkhww6R"
		shareID   = "aB3xY9kQz1" // exempt family
	)

	BeforeEach(func() {
		var err error
		db, err = sql.Open("sqlite3", "file::memory:")
		Expect(err).ToNot(HaveOccurred())
		db.SetMaxOpenConns(1) // non-shared :memory: — every new conn is a fresh empty DB
		DeferCleanup(func() { _ = db.Close() })

		// Minimal fixture: every table/column idColumns and the list rewrites touch.
		_, err = db.Exec(`
			CREATE TABLE media_file (id text, pid text, artist_id text, album_id text, album_artist_id text, folder_id text, mbz_recording_id text);
			CREATE TABLE album (id text, album_artist_id text);
			CREATE TABLE artist (id text);
			CREATE TABLE folder (id text, parent_id text);
			CREATE TABLE tag (id text);
			CREATE TABLE library_artist (artist_id text);
			CREATE TABLE user (id text);
			CREATE TABLE user_props (user_id text);
			CREATE TABLE playlist (id text, owner_id text);
			CREATE TABLE playlist_tracks (playlist_id text, media_file_id text);
			CREATE TABLE playlist_fields (playlist_id text);
			CREATE TABLE annotation (user_id text, item_id text, item_type text);
			CREATE TABLE bookmark (user_id text, item_id text);
			CREATE TABLE player (id text, user_id text, transcoding_id text);
			CREATE TABLE transcoding (id text);
			CREATE TABLE radio (id text);
			CREATE TABLE share (id text, user_id text, resource_ids text, contents text);
			CREATE TABLE scrobble_buffer (id text, user_id text, media_file_id text);
			CREATE TABLE playqueue (id text, user_id text, items text);
		`)
		Expect(err).ToNot(HaveOccurred())

		seed := func(query string, args ...any) {
			_, err := db.Exec(query, args...)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
		}
		// media_file: legacy id/pid, hash artist, legacy album, mbz uuid must stay untouched
		seed(`INSERT INTO media_file VALUES (?, ?, ?, ?, '', '', ?)`, legacyOld, legacyOld, hashID, legacyOld, uuidOld)
		seed(`INSERT INTO album VALUES (?, ?)`, legacyOld, hashID)
		seed(`INSERT INTO user VALUES (?)`, randOld)
		seed(`INSERT INTO playlist VALUES (?, ?)`, uuidOld, randOld) // uuid id, random owner
		seed(`INSERT INTO annotation VALUES (?, ?, 'media_file')`, randOld, legacyOld)
		seed(`INSERT INTO playqueue VALUES (?, ?, ?)`, randOld, randOld, legacyOld+","+hashID)
		seed(`INSERT INTO share VALUES (?, ?, ?, 'Album Foo...')`, shareID, randOld, legacyOld+","+uuidOld)

		tx, err = db.Begin()
		Expect(err).ToNot(HaveOccurred())
		Expect(upUniformCanonicalIds(ctx, tx)).To(Succeed())
		Expect(tx.Commit()).To(Succeed())
	})

	get := func(query string) string {
		var s string
		ExpectWithOffset(1, db.QueryRow(query).Scan(&s)).To(Succeed())
		return s
	}

	It("canonicalizes every id family and keeps references consistent", func() {
		Expect(get(`SELECT id FROM media_file`)).To(Equal(legacyNew))
		Expect(get(`SELECT pid FROM media_file`)).To(Equal(legacyNew))
		Expect(get(`SELECT artist_id FROM media_file`)).To(Equal(hashID))
		Expect(get(`SELECT album_id FROM media_file`)).To(Equal(legacyNew))
		Expect(get(`SELECT id FROM album`)).To(Equal(legacyNew))
		Expect(get(`SELECT id FROM user`)).To(Equal(randNew))
		Expect(get(`SELECT id FROM playlist`)).To(Equal(uuidNew))
		Expect(get(`SELECT owner_id FROM playlist`)).To(Equal(randNew))
		Expect(get(`SELECT item_id FROM annotation`)).To(Equal(legacyNew))
		Expect(get(`SELECT user_id FROM annotation`)).To(Equal(randNew))
	})

	It("rewrites list columns element-wise", func() {
		Expect(get(`SELECT items FROM playqueue`)).To(Equal(legacyNew + "," + hashID))
		Expect(get(`SELECT resource_ids FROM share`)).To(Equal(legacyNew + "," + uuidNew))
	})

	It("leaves exempt columns alone", func() {
		Expect(get(`SELECT id FROM share`)).To(Equal(shareID))
		Expect(get(`SELECT contents FROM share`)).To(Equal("Album Foo..."))
		Expect(get(`SELECT mbz_recording_id FROM media_file`)).To(Equal(uuidOld))
	})
})
