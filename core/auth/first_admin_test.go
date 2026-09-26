package auth_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CreateFirstAdmin", Ordered, func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeAll(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.DbPath = filepath.Join(GinkgoT().TempDir(), "first-admin.db") + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"
		DeferCleanup(db.Init(GinkgoT().Context()))
		ds = persistence.New(db.Db())
	})

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		_, err := db.Db().ExecContext(ctx, "delete from user")
		Expect(err).ToNot(HaveOccurred())
	})

	create := func(name string) (*model.User, error) {
		return auth.CreateFirstAdmin(ctx, ds, name, "secret", nil)
	}

	It("creates an admin with a title-cased name and returns it with its id", func() {
		u, err := create("john")
		Expect(err).ToNot(HaveOccurred())
		Expect(u.ID).ToNot(BeEmpty())
		Expect(u.IsAdmin).To(BeTrue())
		Expect(u.Name).To(Equal("John"))

		stored, err := ds.User().FindByUsernameWithPassword(ctx, "john")
		Expect(err).ToNot(HaveOccurred())
		Expect(stored.Password).To(Equal("secret"))
	})

	It("refuses once any user exists", func() {
		_, err := create("first")
		Expect(err).ToNot(HaveOccurred())
		_, err = create("second")
		Expect(err).To(MatchError(auth.ErrSetupComplete))
	})

	It("runs then in the same transaction, rolling the user back when it fails", func() {
		boom := errors.New("boom")
		var seen string
		_, err := auth.CreateFirstAdmin(ctx, ds, "john", "secret", func(tx model.DataStore, u *model.User) error {
			seen = u.ID
			Expect(tx.User().CountAll(ctx)).To(Equal(int64(1)))
			return boom
		})
		Expect(err).To(MatchError(boom))
		Expect(seen).ToNot(BeEmpty())
		Expect(ds.User().CountAll(ctx)).To(BeZero())
	})

	It("lets exactly one of two concurrent setups win", func() {
		var wg sync.WaitGroup
		errs := make([]error, 2)
		for i, name := range []string{"racer-a", "racer-b"} {
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				_, errs[i] = auth.CreateFirstAdmin(ctx, slowCountDS{ds}, name, "secret", nil)
			}()
		}
		wg.Wait()
		Expect(errs).To(ContainElement(BeNil()))
		Expect(errs).To(ContainElement(MatchError(auth.ErrSetupComplete)))
		Expect(ds.User().CountAll(ctx)).To(Equal(int64(1)))
	})
})

type slowCountDS struct{ model.DataStore }

func (d slowCountDS) User() model.UserRepository { return slowCountUsers{d.DataStore.User()} }

func (d slowCountDS) WithTxImmediate(block func(tx model.DataStore) error, scope ...string) error {
	return d.DataStore.WithTxImmediate(func(tx model.DataStore) error { return block(slowCountDS{tx}) }, scope...)
}

type slowCountUsers struct{ model.UserRepository }

// Holds the transaction open after counting, so an unlocked count would interleave with the other racer.
func (u slowCountUsers) CountAll(ctx context.Context, opts ...model.QueryOptions) (int64, error) {
	n, err := u.UserRepository.CountAll(ctx, opts...)
	time.Sleep(50 * time.Millisecond)
	return n, err
}
