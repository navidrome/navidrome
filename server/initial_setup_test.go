package server

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type userErrors struct {
	put, countAll error
}

func withUserErrors(ds model.DataStore, errs userErrors) model.DataStore {
	return persistencetest.Stub(ds, persistencetest.Repos{User: func(r model.UserRepository) model.UserRepository {
		return &userErrorsRepo{UserRepository: r, errs: errs}
	}})
}

type userErrorsRepo struct {
	model.UserRepository
	errs userErrors
}

func (r *userErrorsRepo) Put(ctx context.Context, u *model.User) error {
	if r.errs.put != nil {
		return r.errs.put
	}
	return r.UserRepository.Put(ctx, u)
}

func (r *userErrorsRepo) CountAll(ctx context.Context, qo ...model.QueryOptions) (int64, error) {
	if r.errs.countAll != nil {
		return 0, r.errs.countAll
	}
	return r.UserRepository.CountAll(ctx, qo...)
}

var _ = Describe("initial_setup", func() {
	var ds model.DataStore
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
	})

	Describe("createInitialAdminUser", func() {
		It("creates a new admin user with specified password if User table is empty", func() {
			Expect(createInitialAdminUser(ctx, ds, "pass123")).To(Succeed())
			ur := ds.User()
			admin, err := ur.FindByUsernameWithPassword(ctx, "admin")
			Expect(err).ToNot(HaveOccurred())
			Expect(admin.Password).To(Equal("pass123"))
		})

		It("does not create a new admin user if User table is not empty", func() {
			Expect(createInitialAdminUser(ctx, ds, "first")).To(Succeed())
			ur := ds.User()
			Expect(ur.CountAll(ctx)).To(Equal(int64(1)))
			Expect(createInitialAdminUser(ctx, ds, "second")).To(Succeed())
			Expect(ur.CountAll(ctx)).To(Equal(int64(1)))
		})

		It("returns the error when the user cannot be stored", func() {
			boom := errors.New("db is down")
			Expect(createInitialAdminUser(ctx, withUserErrors(ds, userErrors{put: boom}), "pass123")).To(MatchError(boom))
		})

		It("returns the error when the user table cannot be read", func() {
			boom := errors.New("db is down")
			Expect(createInitialAdminUser(ctx, withUserErrors(ds, userErrors{countAll: boom}), "pass123")).To(MatchError(boom))
		})
	})
})

var _ = Describe("initialSetup with a real database", func() {
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
			_, _ = conn.ExecContext(context.WithoutCancel(ctx), "ROLLBACK")
			_ = conn.Close()
		}()
		DeferCleanup(func() { <-released })
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
	})

	It("completes the first-run setup when the write lock is held briefly", func() {
		holdWriteLock(300 * time.Millisecond)

		Expect(initialSetup(ctx, ds)).To(Succeed())
		Expect(ds.Property().Get(ctx, consts.InitialSetupFlagKey)).ToNot(BeEmpty())
	})

	It("does not need the write lock when setup already ran and the music folder is unchanged", func() {
		Expect(initialSetup(ctx, ds)).To(Succeed())
		const lockHeld = 1500 * time.Millisecond
		holdWriteLock(lockHeld)

		start := time.Now()
		Expect(initialSetup(ctx, ds)).To(Succeed())
		Expect(time.Since(start)).To(BeNumerically("<", lockHeld/2))
	})

	It("stores a changed music folder when the write lock is held briefly", func() {
		Expect(initialSetup(ctx, ds)).To(Succeed())
		newFolder := conf.Server().MusicFolder + "-moved"
		conf.Update(func(c *conf.Options) { c.MusicFolder = newFolder })
		holdWriteLock(300 * time.Millisecond)

		Expect(initialSetup(ctx, ds)).To(Succeed())
		lib, err := ds.Library().Get(ctx, model.DefaultLibraryID)
		Expect(err).ToNot(HaveOccurred())
		Expect(lib.Path).To(Equal(newFolder))
	})
})
