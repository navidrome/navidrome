package persistence

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"github.com/navidrome/navidrome/db"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// When creating migrations that change existing columns, it is easy to miss the original collation of a column.
// These tests enforce that the required collation of the columns and indexes in the database are kept in place.
// This is important to ensure that the database can perform fast case-insensitive searches and sorts.
var _ = Describe("Collation", func() {
	conn := db.Db()
	DescribeTable("Column collation",
		func(table, column string) {
			Expect(checkCollation(conn, table, column)).To(Succeed())
		},
		Entry("artist.order_artist_name", "artist", "order_artist_name"),
		Entry("artist.sort_artist_name", "artist", "sort_artist_name"),
		Entry("album.order_album_name", "album", "order_album_name"),
		Entry("album.order_album_artist_name", "album", "order_album_artist_name"),
		Entry("album.sort_album_name", "album", "sort_album_name"),
		Entry("album.sort_album_artist_name", "album", "sort_album_artist_name"),
		Entry("media_file.order_title", "media_file", "order_title"),
		Entry("media_file.order_album_name", "media_file", "order_album_name"),
		Entry("media_file.order_artist_name", "media_file", "order_artist_name"),
		Entry("media_file.sort_title", "media_file", "sort_title"),
		Entry("media_file.sort_album_name", "media_file", "sort_album_name"),
		Entry("media_file.sort_artist_name", "media_file", "sort_artist_name"),
		Entry("radio.name", "radio", "name"),
		Entry("user.name", "user", "name"),
	)

	DescribeTable("Index collation",
		func(table, column string) {
			Expect(checkIndexUsage(conn, table, column)).To(Succeed())
		},
		Entry("artist.order_artist_name", "artist", "order_artist_name collate nocase"),
		Entry("artist.sort_artist_name", "artist", "coalesce(nullif(sort_artist_name,''),order_artist_name) collate nocase"),
		Entry("album.order_album_name", "album", "order_album_name collate nocase"),
		Entry("album.order_album_artist_name", "album", "order_album_artist_name collate nocase"),
		Entry("album.sort_album_name", "album", "coalesce(nullif(sort_album_name,''),order_album_name) collate nocase"),
		Entry("album.sort_album_artist_name", "album", "coalesce(nullif(sort_album_artist_name,''),order_album_artist_name) collate nocase"),
		Entry("media_file.order_title", "media_file", "order_title collate nocase"),
		Entry("media_file.order_album_name", "media_file", "order_album_name collate nocase"),
		Entry("media_file.order_artist_name", "media_file", "order_artist_name collate nocase"),
		Entry("media_file.sort_title", "media_file", "coalesce(nullif(sort_title,''),order_title) collate nocase"),
		Entry("media_file.sort_album_name", "media_file", "coalesce(nullif(sort_album_name,''),order_album_name) collate nocase"),
		Entry("media_file.sort_artist_name", "media_file", "coalesce(nullif(sort_artist_name,''),order_artist_name) collate nocase"),
		Entry("media_file.path", "media_file", "path collate nocase"),
		Entry("radio.name", "radio", "name collate nocase"),
		Entry("user.user_name", "user", "user_name collate nocase"),
	)
})

func checkIndexUsage(conn *sql.DB, table string, column string) error {
	rows, err := conn.Query(fmt.Sprintf(`
explain query plan select * from %[1]s
where %[2]s = 'test'
order by %[2]s`, table, column))
	if err != nil {
		return err
	}
	defer rows.Close()

	err = rows.Err()
	if err != nil {
		return err
	}

	if rows.Next() {
		var dummy int
		var detail string
		err = rows.Scan(&dummy, &dummy, &dummy, &detail)
		if err != nil {
			return nil
		}
		if ok, _ := regexp.MatchString("SEARCH.*USING INDEX", detail); ok {
			return nil
		} else {
			return fmt.Errorf("INDEX for '%s' not used: %s", column, detail)
		}
	}
	return errors.New("no rows returned")
}

func checkCollation(conn *sql.DB, table string, column string) error {
	rows, err := conn.Query(fmt.Sprintf("SELECT sql FROM sqlite_master WHERE type='table' AND tbl_name='%s'", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	err = rows.Err()
	if err != nil {
		return err
	}

	if rows.Next() {
		var res string
		err = rows.Scan(&res)
		if err != nil {
			return err
		}
		re := regexp.MustCompile(fmt.Sprintf(`(?i)\b%s\b.*varchar`, column))
		if !re.MatchString(res) {
			return fmt.Errorf("column '%s' not found in table '%s'", column, table)
		}
		re = regexp.MustCompile(fmt.Sprintf(`(?i)\b%s\b.*collate\s+NOCASE`, column))
		if re.MatchString(res) {
			return nil
		}
	} else {
		return fmt.Errorf("table '%s' not found", table)
	}
	return fmt.Errorf("column '%s' in table '%s' does not have NOCASE collation", column, table)
}
