package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"regexp"
	"slices"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests guard against sort options silently losing index support: adding or
// changing a sort mapping, or dropping/renaming an index in a migration, must not
// reintroduce full-table temp B-tree sorts on the large tables. Those are
// catastrophic on big libraries but invisible on dev-sized ones, which is how the
// unindexed album/artist song sorts went unnoticed for years.
//
// Every sort mapping is checked automatically: the real ORDER BY is built via
// buildSortOrder (both directions) and verified with EXPLAIN QUERY PLAN against
// the migrated test schema. The planner's choice is deterministic even on an
// empty table. A sort passes when the plan has no full "USE TEMP B-TREE FOR
// ORDER BY" step; an incremental sort of tie groups ("... FOR LAST TERM OF ORDER
// BY") is fine, as it only sorts rows with equal leading columns.
//
// A new sort mapping therefore fails this test until a matching index is created.
// The only escape hatch is exceptions, for sorts that genuinely cannot be
// served by a table index (random, annotation-join columns, JSON expressions):
// declaring one requires writing down the reason, making the trade-off visible in
// review. The checks run with the default config: PreferSortTags=true rewrites
// mappings to coalesce expressions with no matching indexes (used by ~0.1% of
// installations, per insights), and is out of scope here.
var _ = Describe("Sort index coverage", func() {
	conn := db.Db()

	type repoCase struct {
		table   string
		newRepo func(ctx context.Context) *sqlRepository
		// sort mapping -> reason it cannot be served by an index
		exceptions map[string]string
	}

	cases := []repoCase{
		{
			table: "media_file",
			newRepo: func(ctx context.Context) *sqlRepository {
				return &NewMediaFileRepository(ctx, GetDBXBuilder()).(*mediaFileRepository).sqlRepository
			},
			exceptions: map[string]string{
				"random":     "not a column sort",
				"starred_at": "sorts on annotation join columns",
				"rated_at":   "sorts on annotation join columns",
				"play_count": "sorts on annotation join columns",
				"play_date":  "sorts on annotation join columns",
				"rating":     "sorts on annotation join columns",
				"comment":    "UI-sortable but rarely used; not worth an index",
			},
		},
		{
			table: "album",
			newRepo: func(ctx context.Context) *sqlRepository {
				return &NewAlbumRepository(ctx, GetDBXBuilder()).(*albumRepository).sqlRepository
			},
			exceptions: map[string]string{
				"random":     "not a column sort",
				"starred_at": "sorts on annotation join columns",
				"rated_at":   "sorts on annotation join columns",
				"max_year":   "coalesce expression over original_date/max_year, no expression index",
			},
		},
		{
			table: "artist",
			newRepo: func(ctx context.Context) *sqlRepository {
				return &NewArtistRepository(ctx, GetDBXBuilder()).(*artistRepository).sqlRepository
			},
			exceptions: map[string]string{ //nolint:gosec // G101 false positive, same as the artist sortMappings
				"starred_at":             "sorts on annotation join columns",
				"rated_at":               "sorts on annotation join columns",
				"song_count":             "JSON expression over stats column",
				"album_count":            "JSON expression over stats column",
				"size":                   "JSON expression over stats column",
				"maincredit_song_count":  "aggregate over JSON stats",
				"maincredit_album_count": "aggregate over JSON stats",
				"maincredit_size":        "aggregate over JSON stats",
			},
		},
	}

	newCtx := func() context.Context {
		ctx := log.NewContext(GinkgoT().Context())
		return request.WithUser(ctx, model.User{ID: "userid"})
	}

	for _, c := range cases {
		It(fmt.Sprintf("uses an index for every sort mapping on %s", c.table), func() {
			r := c.newRepo(newCtx())
			for _, sort := range slices.Sorted(maps.Keys(r.sortMappings)) {
				if _, ok := c.exceptions[sort]; ok {
					continue
				}
				for _, dir := range []string{"asc", "desc"} {
					orderBy := r.buildSortOrder(sort, dir)
					Expect(checkSortUsesIndex(conn, c.table, orderBy)).To(Succeed(),
						"sort %q (%s) on table %q needs an index. Create one matching its ORDER BY, or, if it cannot be served by an index, add it to exceptions with the reason",
						sort, dir, c.table)
				}
			}
		})

		It(fmt.Sprintf("has no stale exceptions entries for %s", c.table), func() {
			r := c.newRepo(newCtx())
			for _, sort := range slices.Sorted(maps.Keys(c.exceptions)) {
				Expect(r.sortMappings).To(HaveKey(sort),
					"exceptions entry %q on table %q does not match any sort mapping - remove it", sort, c.table)
			}
		})
	}

	It("uses an index for recently_added when RecentlyAddedByModTime is enabled", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.RecentlyAddedByModTime = true
		for _, c := range cases[:2] { // media_file and album
			r := c.newRepo(newCtx())
			for _, dir := range []string{"asc", "desc"} {
				orderBy := r.buildSortOrder("recently_added", dir)
				Expect(checkSortUsesIndex(conn, c.table, orderBy)).To(Succeed(),
					"sort recently_added (%s) on table %q", dir, c.table)
			}
		}
	})
})

// Matches the full-sort step only: incremental tie-group sorts are reported as
// "USE TEMP B-TREE FOR LAST TERM OF ORDER BY" (or "LAST N TERMS") and are allowed.
var fullTempBTreeSort = regexp.MustCompile(`USE TEMP B-TREE FOR ORDER BY`)

func checkSortUsesIndex(conn *sql.DB, table, orderBy string) error {
	rows, err := conn.Query(fmt.Sprintf("explain query plan select * from %s order by %s limit 15", table, orderBy))
	if err != nil {
		return fmt.Errorf("explain query plan failed for order by %q: %w", orderBy, err)
	}
	defer rows.Close()

	var details []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			return err
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if slices.ContainsFunc(details, fullTempBTreeSort.MatchString) {
		return fmt.Errorf("no index satisfies ORDER BY %s - plan: %v", orderBy, details)
	}
	return nil
}
