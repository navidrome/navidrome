package db_test

import (
	"context"
	"database/sql"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/singleton"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func shortTime(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, time.UTC)
}

var _ = Describe("database backups", func() {
	When("there are a few backup files", func() {
		var ctx context.Context
		var timesShuffled []time.Time

		timesDecreasingChronologically := []time.Time{
			shortTime(2024, 11, 6, 5, 11),
			shortTime(2024, 11, 6, 5, 8),
			shortTime(2024, 11, 6, 4, 32),
			shortTime(2024, 11, 6, 2, 4),
			shortTime(2024, 11, 6, 1, 52),

			shortTime(2024, 11, 5, 23, 0),
			shortTime(2024, 11, 5, 6, 4),
			shortTime(2024, 11, 4, 2, 4),
			shortTime(2024, 11, 3, 8, 5),
			shortTime(2024, 11, 2, 5, 24),
			shortTime(2024, 11, 1, 5, 24),

			shortTime(2024, 10, 31, 5, 9),
			shortTime(2024, 10, 30, 5, 9),
			shortTime(2024, 10, 23, 14, 3),
			shortTime(2024, 10, 22, 3, 6),
			shortTime(2024, 10, 11, 14, 3),

			shortTime(2024, 9, 21, 19, 5),
			shortTime(2024, 9, 3, 8, 5),

			shortTime(2024, 7, 5, 1, 1),

			shortTime(2023, 8, 2, 19, 5),

			shortTime(2021, 8, 2, 19, 5),
			shortTime(2020, 8, 2, 19, 5),
		}

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())

			tempFolder, err := os.MkdirTemp("", "navidrome_backup")
			Expect(err).ToNot(HaveOccurred())
			conf.Server.Backup.Path = conf.NewDir(tempFolder)

			DeferCleanup(func() {
				_ = os.RemoveAll(tempFolder)
			})

			timesShuffled = make([]time.Time, len(timesDecreasingChronologically))
			copy(timesShuffled, timesDecreasingChronologically)
			//nolint:gosec // shuffle order is not a security decision
			rand.Shuffle(len(timesShuffled), func(i, j int) {
				timesShuffled[i], timesShuffled[j] = timesShuffled[j], timesShuffled[i]
			})

			for _, time := range timesShuffled {
				path := BackupPath(time)
				file, err := os.Create(path)
				Expect(err).ToNot(HaveOccurred())
				_ = file.Close()
			}

			ctx = context.Background()
		})

		DescribeTable("prune", func(count, expected int) {
			conf.Server.Backup.Count = count
			pruneCount, err := Prune(ctx)
			Expect(err).ToNot(HaveOccurred())
			for idx, time := range timesDecreasingChronologically {
				_, err := os.Stat(BackupPath(time))
				shouldExist := idx < conf.Server.Backup.Count
				if shouldExist {
					Expect(err).ToNot(HaveOccurred())
				} else {
					Expect(err).To(MatchError(os.ErrNotExist))
				}
			}

			Expect(len(timesDecreasingChronologically) - pruneCount).To(Equal(expected))
		},
			Entry("preserve latest 5 backups", 5, 5),
			Entry("delete all files", 0, 0),
			Entry("preserve all files when at length", len(timesDecreasingChronologically), len(timesDecreasingChronologically)),
			Entry("preserve all files when less than count", 10000, len(timesDecreasingChronologically)))
	})

	Describe("backup and restore", Ordered, func() {
		var ctx context.Context

		BeforeAll(func() {
			ctx = context.Background()
			DeferCleanup(configtest.SetupConfig())

			conf.Server.DbPath = "file::memory:?cache=shared&_foreign_keys=on"
			DeferCleanup(Init(ctx))
		})

		BeforeEach(func() {
			tempFolder, err := os.MkdirTemp("", "navidrome_backup")
			Expect(err).ToNot(HaveOccurred())
			conf.Server.Backup.Path = conf.NewDir(tempFolder)

			DeferCleanup(func() {
				_ = os.RemoveAll(tempFolder)
			})
		})

		It("successfully backups the database", func() {
			path, err := Backup(ctx)
			Expect(err).ToNot(HaveOccurred())

			backup, err := sql.Open(Driver, path)
			Expect(err).ToNot(HaveOccurred())
			Expect(IsSchemaEmpty(ctx, backup)).To(BeFalse())
		})

		It("successfully restores the database", func() {
			path, err := Backup(ctx)
			Expect(err).ToNot(HaveOccurred())

			err = tests.ClearDB()
			Expect(err).ToNot(HaveOccurred())
			Expect(IsSchemaEmpty(ctx, Db())).To(BeTrue())

			err = Restore(ctx, path)
			Expect(err).ToNot(HaveOccurred())
			Expect(IsSchemaEmpty(ctx, Db())).To(BeFalse())
		})
	})

	Describe("backup and restore with a file-based database", Ordered, func() {
		var ctx context.Context
		var tempFolder string
		var dbFilePath string

		BeforeAll(func() {
			ctx = context.Background()
			DeferCleanup(configtest.SetupConfig())

			var err error
			tempFolder, err = os.MkdirTemp("", "navidrome_restore")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() {
				Close(ctx)
				_ = os.RemoveAll(tempFolder)
			})

			// Mimic the production DSN (consts.DefaultDbPath): a file database in WAL mode.
			dbFilePath = filepath.Join(tempFolder, "navidrome.db")
			conf.Server.DbPath = dbFilePath + "?_busy_timeout=15000&_journal_mode=WAL&_foreign_keys=on&synchronous=normal"
			// The previous container's cleanup closed the shared *sql.DB without
			// dropping the singleton; force a fresh connection for this container.
			singleton.DeleteInstance[*sql.DB]()
			DeferCleanup(Init(ctx))
		})

		It("restores data into a database whose stale WAL sidecar files were left behind", func() {
			By("seeding user data in the current database")
			_, err := Db().ExecContext(ctx, `INSERT INTO user (id, user_name, name, email, password, is_admin, created_at, updated_at)
				VALUES ('u-restore-1', 'drilladmin', 'drilladmin', 'drilladmin@example.com', 'x', 1, datetime('now'), datetime('now'))`)
			Expect(err).ToNot(HaveOccurred())

			By("creating a backup containing the user row")
			path, err := Backup(ctx)
			Expect(err).ToNot(HaveOccurred())

			By("simulating the CLI exiting without closing the pool: sidecar files stay behind")
			_, err = Db().ExecContext(ctx, "CREATE TABLE IF NOT EXISTS _restore_probe(x)")
			Expect(err).ToNot(HaveOccurred())
			singleton.DeleteInstance[*sql.DB]()

			err = tests.ClearDB()
			Expect(err).ToNot(HaveOccurred())

			By("restoring the backup")
			Expect(Restore(ctx, path)).To(Succeed())

			By("verifying the restored data is readable through a fresh connection")
			singleton.DeleteInstance[*sql.DB]()
			var userName string
			Expect(Db().QueryRowContext(ctx, "SELECT user_name FROM user WHERE id = 'u-restore-1'").Scan(&userName)).To(Succeed())
			Expect(userName).To(Equal("drilladmin"))
		})

		It("fails to restore from a backup file that does not exist, leaving the database intact", func() {
			By("seeding user data in the current database")
			_, err := Db().ExecContext(ctx, `INSERT INTO user (id, user_name, name, email, password, is_admin, created_at, updated_at)
				VALUES ('u-restore-2', 'keepme', 'keepme', 'keepme@example.com', 'x', 1, datetime('now'), datetime('now'))`)
			Expect(err).ToNot(HaveOccurred())

			By("attempting a restore from a nonexistent file")
			missingPath := filepath.Join(tempFolder, "does_not_exist.db")
			err = Restore(ctx, missingPath)
			Expect(err).To(HaveOccurred())

			By("verifying the database was not wiped")
			var userName string
			Expect(Db().QueryRowContext(ctx, "SELECT user_name FROM user WHERE id = 'u-restore-2'").Scan(&userName)).To(Succeed())
			Expect(userName).To(Equal("keepme"))
			_, statErr := os.Stat(missingPath)
			Expect(statErr).To(MatchError(os.ErrNotExist))
		})
	})
})
