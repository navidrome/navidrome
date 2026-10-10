package migrations

import (
	"context"
	"database/sql"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("upRequeueSmartPlaylistCovers", func() {
	var db *sql.DB
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		var err error
		db, err = sql.Open("sqlite3", "file::memory:")
		Expect(err).ToNot(HaveOccurred())
		db.SetMaxOpenConns(1) // non-shared :memory: — every new conn is a fresh empty DB
		DeferCleanup(func() { _ = db.Close() })

		_, err = db.Exec(`
			CREATE TABLE playlist (id text, rules text, song_count integer);
			CREATE TABLE item_artwork (item_kind text, item_id text, image_type text, hash text NOT NULL DEFAULT '',
				PRIMARY KEY (item_kind, item_id, image_type));
			CREATE TABLE artwork_queue (item_kind text NOT NULL, item_id text NOT NULL, image_type text NOT NULL DEFAULT 'primary',
				priority integer NOT NULL DEFAULT 0, attempts integer NOT NULL DEFAULT 0,
				retry_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP, enqueued_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
				trace jsonb NOT NULL DEFAULT '[]', PRIMARY KEY (item_kind, item_id, image_type));

			INSERT INTO playlist VALUES
				('absent', '{"all":[]}', 5), ('no-row', '{"all":[]}', 5), ('has-cover', '{"all":[]}', 5),
				('static-null', NULL, 5), ('static-empty', '', 5), ('empty-smart', '{"all":[]}', 0),
				('queued', '{"all":[]}', 5);
			INSERT INTO item_artwork VALUES
				('pl', 'absent', 'primary', ''), ('pl', 'has-cover', 'primary', 'h1'),
				('pl', 'static-null', 'primary', ''), ('pl', 'static-empty', 'primary', ''),
				('pl', 'empty-smart', 'primary', ''), ('pl', 'queued', 'primary', ''),
				('al', 'no-row', 'primary', '');
		`)
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Exec(`INSERT INTO artwork_queue (item_kind, item_id, priority, retry_at, enqueued_at)
			VALUES ('pl', 'queued', 100, ?, ?)`, time.Now(), time.Now())
		Expect(err).ToNot(HaveOccurred())

		tx, err := db.BeginTx(ctx, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(upRequeueSmartPlaylistCovers(ctx, tx)).To(Succeed())
		Expect(tx.Commit()).To(Succeed())
	})

	// Queued ids and priorities, limited to rows whose retry_at has passed, as DequeueBatch reads them.
	dequeueable := func() map[string]int {
		rows, err := db.QueryContext(ctx, `SELECT item_id, priority FROM artwork_queue
			WHERE item_kind = 'pl' AND retry_at <= ?`, time.Now())
		Expect(err).ToNot(HaveOccurred())
		defer rows.Close()
		res := map[string]int{}
		for rows.Next() {
			var id string
			var priority int
			Expect(rows.Scan(&id, &priority)).To(Succeed())
			res[id] = priority
		}
		Expect(rows.Err()).ToNot(HaveOccurred())
		return res
	}

	It("re-queues populated smart playlists that have no cover, ready for the next drain", func() {
		Expect(dequeueable()).To(Equal(map[string]int{
			"absent": 50,
			"no-row": 50,
			"queued": 100, // already queued: left untouched
		}))
	})
})
