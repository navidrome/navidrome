package persistencetest_test

import (
	"context"
	"errors"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var errStub = errors.New("stubbed album error")

type failingAlbumRepo struct{ model.AlbumRepository }

func (failingAlbumRepo) Get(context.Context, string) (*model.Album, error) { return nil, errStub }

type failingAlbums struct{ model.DataStore }

func (f failingAlbums) Album() model.AlbumRepository {
	return failingAlbumRepo{f.DataStore.Album()}
}

var _ = Describe("Overlay", func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistencetest.Overlay(persistence.New(db.Db()), func(ds model.DataStore) model.DataStore {
			return failingAlbums{ds}
		})
	})

	It("applies the stub outside a transaction", func() {
		_, err := ds.Album().Get(ctx, "any")
		Expect(err).To(MatchError(errStub))
	})

	DescribeTable("applies the stub inside a transaction",
		func(inTx func(block func(tx model.DataStore) error) error) {
			var err error
			Expect(inTx(func(tx model.DataStore) error {
				_, err = tx.Album().Get(ctx, "any")
				return nil
			})).To(Succeed())
			Expect(err).To(MatchError(errStub))
		},
		Entry("WithTx", func(block func(tx model.DataStore) error) error {
			return ds.WithTx(block)
		}),
		Entry("WithTxImmediate", func(block func(tx model.DataStore) error) error {
			return ds.WithTxImmediate(block)
		}),
		Entry("WithTxRetry", func(block func(tx model.DataStore) error) error {
			return ds.WithTxRetry(ctx, func(_ context.Context, tx model.DataStore) error { return block(tx) })
		}),
		Entry("a nested WithTx", func(block func(tx model.DataStore) error) error {
			return ds.WithTx(func(tx model.DataStore) error { return tx.WithTx(block) })
		}),
	)
})
