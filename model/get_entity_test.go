package model_test

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

var _ = Describe("GetEntityByID", func() {
	var ds model.DataStore
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
	})

	It("returns the entity matching the id", func() {
		persistencetest.PutAlbums(ctx, ds, model.Album{ID: "a1", Name: "One"})
		entity, err := model.GetEntityByID(ctx, ds, "a1")
		Expect(err).ToNot(HaveOccurred())
		Expect(entity).To(BeAssignableToTypeOf(&model.Album{}))
		Expect(entity.(*model.Album).ID).To(Equal("a1"))
	})

	It("returns ErrNotFound when no entity matches", func() {
		persistencetest.PutAlbums(ctx, ds, model.Album{ID: "a1", Name: "One"})
		_, err := model.GetEntityByID(ctx, ds, "missing")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("propagates unexpected repository errors instead of reporting not-found", func() {
		persistencetest.PutAlbums(ctx, ds, model.Album{ID: "a1", Name: "One"})
		ds = persistencetest.Stub(ds, persistencetest.Repos{Album: func(r model.AlbumRepository) model.AlbumRepository {
			return failingAlbumRepo{r}
		}})
		_, err := model.GetEntityByID(ctx, ds, "a1")
		Expect(err).To(MatchError(errAlbumGet))
	})
})

var _ = Describe("GetEntityKindByID", func() {
	var ds model.DataStore
	var ctx context.Context

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
	})

	It("returns the artwork kind for the matching id", func() {
		persistencetest.PutAlbums(ctx, ds, model.Album{ID: "a1"})
		kind, err := model.GetEntityKindByID(ctx, ds, "a1")
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(model.KindAlbumArtwork))
	})

	It("returns ErrNotFound when no entity matches", func() {
		_, err := model.GetEntityKindByID(ctx, ds, "missing")
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})

var errAlbumGet = errors.New("album lookup failed")

type failingAlbumRepo struct{ model.AlbumRepository }

func (failingAlbumRepo) Get(context.Context, string) (*model.Album, error) { return nil, errAlbumGet }
