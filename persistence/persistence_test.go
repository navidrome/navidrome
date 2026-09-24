package persistence

import (
	"context"
	"errors"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SQLStore", func() {
	var ds model.DataStore
	var ctx context.Context
	BeforeEach(func() {
		ds = New(db.Db())
		ctx = context.Background()
	})
	Describe("WithTx", func() {
		Context("When block returns nil", func() {
			It("commits changes to the DB", func() {
				err := ds.WithTx(func(tx model.DataStore) error {
					pl := tx.Player(ctx)
					err := pl.Put(&model.Player{ID: "666", UserId: "userid"})
					Expect(err).ToNot(HaveOccurred())

					pr := tx.Property(ctx)
					err = pr.Put("777", "value")
					Expect(err).ToNot(HaveOccurred())
					return nil
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(ds.Player(ctx).Get("666")).To(Equal(&model.Player{ID: "666", UserId: "userid", Username: "userid"}))
				Expect(ds.Property(ctx).Get("777")).To(Equal("value"))
			})
		})
		Context("When block returns an error", func() {
			It("rollbacks changes to the DB", func() {
				err := ds.WithTx(func(tx model.DataStore) error {
					pr := tx.Property(ctx)
					err := pr.Put("999", "value")
					Expect(err).ToNot(HaveOccurred())

					// Will fail as it is missing the UserName
					pl := tx.Player(ctx)
					err = pl.Put(&model.Player{ID: "888"})
					Expect(err).To(HaveOccurred())
					return err
				})
				Expect(err).To(HaveOccurred())
				_, err = ds.Property(ctx).Get("999")
				Expect(err).To(MatchError(model.ErrNotFound))
				_, err = ds.Player(ctx).Get("888")
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})
	})

	Describe("WithTxRetry", func() {
		busy := sqlite3.Error{Code: sqlite3.ErrBusy}
		BeforeEach(func() {
			DeferCleanup(func(d time.Duration) { txRetryDelay = d }, txRetryDelay)
			txRetryDelay = 0
		})

		It("reruns a busy transaction from a clean rollback", func() {
			var attempts []bool
			err := ds.WithTxRetry(ctx, func(ctx context.Context, tx model.DataStore) error {
				attempts = append(attempts, hasBusyRetry(ctx))
				Expect(tx.Property(ctx).Put("retry-key", "attempt")).To(Succeed())
				if len(attempts) < 3 {
					return busy
				}
				return nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(attempts).To(Equal([]bool{true, true, true}))
			Expect(ds.Property(ctx).Get("retry-key")).To(Equal("attempt"))
		})

		It("gives up after the last retry, which is not marked as retried", func() {
			var attempts []bool
			err := ds.WithTxRetry(ctx, func(ctx context.Context, _ model.DataStore) error {
				attempts = append(attempts, hasBusyRetry(ctx))
				return busy
			})
			Expect(db.IsBusy(err)).To(BeTrue())
			Expect(attempts).To(Equal([]bool{true, true, true, false}))
		})

		It("does not rerun on other errors", func() {
			calls := 0
			err := ds.WithTxRetry(ctx, func(context.Context, model.DataStore) error {
				calls++
				return sqlite3.Error{Code: sqlite3.ErrConstraint}
			})
			Expect(err).To(HaveOccurred())
			Expect(calls).To(Equal(1))
		})

		It("does not rerun when called inside a transaction", func() {
			calls := 0
			err := ds.WithTx(func(tx model.DataStore) error {
				return tx.WithTxRetry(ctx, func(context.Context, model.DataStore) error {
					calls++
					return busy
				})
			})
			Expect(db.IsBusy(err)).To(BeTrue())
			Expect(calls).To(Equal(1))
		})

		It("joins the enclosing transaction instead of opening another", func() {
			rollback := errors.New("rollback")
			err := ds.WithTx(func(tx model.DataStore) error {
				Expect(tx.Property(ctx).Put("outer-key", "v")).To(Succeed())
				Expect(tx.WithTxRetry(ctx, func(ctx context.Context, inner model.DataStore) error {
					Expect(inner.Property(ctx).Get("outer-key")).To(Equal("v"))
					return inner.Property(ctx).Put("inner-key", "v")
				})).To(Succeed())
				return rollback
			})
			Expect(err).To(MatchError(rollback))
			_, err = ds.Property(ctx).Get("inner-key")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})
})
