package migrations

import (
	"context"
	"database/sql"
	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
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

		sessionSecret = "encrypted-session-secret-sentinel"
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
			CREATE TABLE playlist (id text, owner_id text, rules text);
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
			CREATE TABLE user_library (user_id text, library_id integer);
			CREATE TABLE scrobbles (id integer, media_file_id text, user_id text);
			CREATE TABLE media_file_artists (media_file_id text, artist_id text);
			CREATE TABLE album_artists (album_id text, artist_id text);
			CREATE TABLE library_tag (tag_id text, library_id integer);
			CREATE TABLE plugin (id text, users text);
			CREATE TABLE property (id text primary key, value text);
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
		// uuid id, random owner, smart-playlist rules with an embedded inPlaylist id and a sibling operator
		seed(`INSERT INTO playlist VALUES (?, ?, ?)`, uuidOld, randOld, `{"all":[{"inPlaylist":{"id":"`+uuidOld+`"}},{"inTheLast":{"lastPlayed":30}}]}`)
		seed(`INSERT INTO annotation VALUES (?, ?, 'media_file')`, randOld, legacyOld)
		seed(`INSERT INTO playqueue VALUES (?, ?, ?)`, randOld, randOld, legacyOld+","+hashID)
		seed(`INSERT INTO share VALUES (?, ?, ?, 'Album Foo...')`, shareID, randOld, legacyOld+","+uuidOld)
		seed(`INSERT INTO user_library VALUES (?, 1)`, randOld)
		seed(`INSERT INTO scrobbles VALUES (1, ?, ?)`, legacyOld, randOld)
		seed(`INSERT INTO media_file_artists VALUES (?, ?)`, legacyOld, hashID)
		seed(`INSERT INTO album_artists VALUES (?, ?)`, legacyOld, hashID)
		seed(`INSERT INTO library_tag VALUES (?, 1)`, hashID)
		seed(`INSERT INTO plugin VALUES ('lastfm', ?)`, `["`+randOld+`","`+hashID+`"]`)
		seed(`INSERT INTO plugin VALUES ('empty', '[]')`) // exempt: empty user list untouched
		// malformed JSON in both a plugin list and a playlist rule: must pass through byte-for-byte
		seed(`INSERT INTO plugin VALUES ('broken', 'not-json')`)
		seed(`INSERT INTO playlist VALUES (?, ?, '{broken')`, hashID, hashID)
		seed(`INSERT INTO property VALUES (?, ?)`, consts.JWTSecretKey, sessionSecret)
	})

	JustBeforeEach(func() {
		var err error
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
		Expect(get(`SELECT id FROM playlist WHERE owner_id='` + randNew + `'`)).To(Equal(uuidNew))
		Expect(get(`SELECT owner_id FROM playlist WHERE id='` + uuidNew + `'`)).To(Equal(randNew))
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

	It("canonicalizes junction and membership tables", func() {
		Expect(get(`SELECT user_id FROM user_library`)).To(Equal(randNew))
		Expect(get(`SELECT media_file_id FROM scrobbles`)).To(Equal(legacyNew))
		Expect(get(`SELECT user_id FROM scrobbles`)).To(Equal(randNew))
		Expect(get(`SELECT media_file_id FROM media_file_artists`)).To(Equal(legacyNew))
		Expect(get(`SELECT artist_id FROM media_file_artists`)).To(Equal(hashID)) // hash-family, unchanged
		Expect(get(`SELECT album_id FROM album_artists`)).To(Equal(legacyNew))
		Expect(get(`SELECT artist_id FROM album_artists`)).To(Equal(hashID)) // hash-family, unchanged
		Expect(get(`SELECT tag_id FROM library_tag`)).To(Equal(hashID))      // hash-family, unchanged
	})

	It("rewrites JSON columns element-wise", func() {
		Expect(get(`SELECT users FROM plugin WHERE id='lastfm'`)).To(Equal(`["` + randNew + `","` + hashID + `"]`))
		Expect(get(`SELECT id FROM plugin WHERE id='lastfm'`)).To(Equal("lastfm")) // plugin name, untouched

		var rules map[string]any
		Expect(json.Unmarshal([]byte(get(`SELECT rules FROM playlist WHERE id='`+uuidNew+`'`)), &rules)).To(Succeed())
		all, ok := rules["all"].([]any)
		Expect(ok).To(BeTrue())
		Expect(all).To(HaveLen(2))
		inPl := all[0].(map[string]any)["inPlaylist"].(map[string]any)
		Expect(inPl["id"]).To(Equal(uuidNew))
	})

	It("preserves sibling operators alongside a rewritten inPlaylist id", func() {
		var rules map[string]any
		Expect(json.Unmarshal([]byte(get(`SELECT rules FROM playlist WHERE id='`+uuidNew+`'`)), &rules)).To(Succeed())
		all := rules["all"].([]any)
		var sawInPlaylist, sawInTheLast bool
		for _, e := range all {
			op := e.(map[string]any)
			if pl, ok := op["inPlaylist"].(map[string]any); ok {
				Expect(pl["id"]).To(Equal(uuidNew))
				sawInPlaylist = true
			}
			if last, ok := op["inTheLast"].(map[string]any); ok {
				Expect(last["lastPlayed"]).To(Equal(float64(30)))
				sawInTheLast = true
			}
		}
		Expect(sawInPlaylist).To(BeTrue())
		Expect(sawInTheLast).To(BeTrue())
	})

	It("leaves exempt JSON rows untouched", func() {
		Expect(get(`SELECT users FROM plugin WHERE id='empty'`)).To(Equal("[]"))
	})

	It("passes malformed JSON columns through byte-for-byte", func() {
		Expect(get(`SELECT users FROM plugin WHERE id='broken'`)).To(Equal("not-json"))
		Expect(get(`SELECT rules FROM playlist WHERE id='` + hashID + `'`)).To(Equal("{broken"))
	})

	rescanCount := func() int {
		var count int
		ExpectWithOffset(1, db.QueryRow(
			`SELECT count(*) FROM property WHERE id = ?`, consts.FullScanAfterMigrationFlagKey).Scan(&count)).To(Succeed())
		return count
	}

	It("does not force a full rescan for the default PID config", func() {
		Expect(rescanCount()).To(Equal(0))
	})

	Context("with a legacy PID configuration", func() {
		BeforeEach(func() {
			prev := conf.Server.PID.Album
			conf.Server.PID.Album = "album_legacy"
			DeferCleanup(func() { conf.Server.PID.Album = prev })
		})

		It("forces a full rescan so composite pids are rewritten", func() {
			Expect(rescanCount()).To(Equal(1))
		})
	})

	propCount := func(key string) int {
		var count int
		ExpectWithOffset(1, db.QueryRow(`SELECT count(*) FROM property WHERE id = ?`, key).Scan(&count)).To(Succeed())
		return count
	}

	It("rotates the session secret to the public key", func() {
		Expect(propCount(consts.JWTSecretKey)).To(Equal(0))
		Expect(get(`SELECT value FROM property WHERE id = '` + consts.JWTPublicSecretKey + `'`)).To(Equal(sessionSecret))
	})

	Context("with no stored session secret", func() {
		BeforeEach(func() {
			_, err := db.Exec(`DELETE FROM property WHERE id = ?`, consts.JWTSecretKey)
			Expect(err).ToNot(HaveOccurred())
		})

		It("is a no-op", func() {
			Expect(propCount(consts.JWTSecretKey)).To(Equal(0))
			Expect(propCount(consts.JWTPublicSecretKey)).To(Equal(0))
		})
	})
})
