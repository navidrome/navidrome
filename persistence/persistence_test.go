package persistence

import (
	"context"

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
})
