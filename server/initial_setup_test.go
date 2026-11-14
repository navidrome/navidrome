package server

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type failingPutUserRepo struct {
	model.UserRepository
	err error
}

func (r *failingPutUserRepo) Put(context.Context, *model.User) error { return r.err }

func dsWithFailingPut(err error) model.DataStore {
	return &tests.MockDataStore{MockedUser: &failingPutUserRepo{UserRepository: tests.CreateMockUserRepo(), err: err}}
}

var _ = Describe("initial_setup", func() {
	var ds model.DataStore
	var ctx context.Context

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ctx = GinkgoT().Context()
	})

	Describe("createInitialAdminUser", func() {
		It("creates a new admin user with specified password if User table is empty", func() {
			Expect(createInitialAdminUser(ctx, ds, "pass123")).To(BeNil())
			ur := ds.User()
			admin, err := ur.FindByUsername(ctx, "admin")
			Expect(err).To(BeNil())
			Expect(admin.Password).To(Equal("pass123"))
		})

		It("does not create a new admin user if User table is not empty", func() {
			Expect(createInitialAdminUser(ctx, ds, "first")).To(BeNil())
			ur := ds.User()
			Expect(ur.CountAll(ctx)).To(Equal(int64(1)))
			Expect(createInitialAdminUser(ctx, ds, "second")).To(BeNil())
			Expect(ur.CountAll(ctx)).To(Equal(int64(1)))
		})

		It("returns the error when the user cannot be stored", func() {
			boom := errors.New("db is down")
			Expect(createInitialAdminUser(ctx, dsWithFailingPut(boom), "pass123")).To(MatchError(boom))
		})

		It("returns the error when the user table cannot be read", func() {
			boom := errors.New("db is down")
			ds = &tests.MockDataStore{MockedUser: &tests.MockedUserRepo{Error: boom}}
			Expect(createInitialAdminUser(ctx, ds, "pass123")).To(MatchError(boom))
		})
	})
})

// db.Db() is a process-wide singleton that the BeforeAll cleanup closes; keep these the only real-DB specs in this package.
var _ = Describe("initialSetup with a real database", Ordered, func() {
	var ctx context.Context
	var ds model.DataStore

	holdWriteLock := func(releaseAfter time.Duration) {
		conn, err := db.Db().Conn(ctx)
		Expect(err).ToNot(HaveOccurred())
		_, err = conn.ExecContext(ctx, "BEGIN IMMEDIATE")
		Expect(err).ToNot(HaveOccurred())
		released := make(chan struct{})
		go func() {
			defer close(released)
			time.Sleep(releaseAfter)
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
			_ = conn.Close()
		}()
		DeferCleanup(func() { <-released })
	}

	BeforeAll(func() {
		DeferCleanup(conf.SnapshotConfig())
		conf.Update(func(c *conf.Options) {
			c.DbPath = filepath.Join(GinkgoT().TempDir(), "initial-setup.db") + "?_journal_mode=WAL&_busy_timeout=1000"
		})
		DeferCleanup(db.Init(GinkgoT().Context()))
		ds = persistence.New(db.Db())
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
	})

	It("completes the first-run setup when the write lock is held briefly", func() {
		holdWriteLock(300 * time.Millisecond)

		Expect(initialSetup(ctx, ds)).To(Succeed())
		Expect(ds.Property().Get(ctx, consts.InitialSetupFlagKey)).ToNot(BeEmpty())
	})

	It("does not need the write lock when setup already ran and the music folder is unchanged", func() {
		holdWriteLock(1500 * time.Millisecond)

		Expect(initialSetup(ctx, ds)).To(Succeed())
	})

	It("stores a changed music folder when the write lock is held briefly", func() {
		newFolder := conf.Server().MusicFolder + "-moved"
		conf.Update(func(c *conf.Options) { c.MusicFolder = newFolder })
		holdWriteLock(300 * time.Millisecond)

		Expect(initialSetup(ctx, ds)).To(Succeed())
		lib, err := ds.Library().Get(ctx, model.DefaultLibraryID)
		Expect(err).ToNot(HaveOccurred())
		Expect(lib.Path).To(Equal(newFolder))
	})
})
