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
		putAlbum(ctx, ds, "a1", "One")
		entity, err := model.GetEntityByID(ctx, ds, "a1")
		Expect(err).ToNot(HaveOccurred())
		Expect(entity).To(BeAssignableToTypeOf(&model.Album{}))
		Expect(entity.(*model.Album).ID).To(Equal("a1"))
	})

	It("returns ErrNotFound when no entity matches", func() {
		putAlbum(ctx, ds, "a1", "One")
		_, err := model.GetEntityByID(ctx, ds, "missing")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("propagates unexpected repository errors instead of reporting not-found", func() {
		putAlbum(ctx, ds, "a1", "One")
		ds = persistencetest.Overlay(ds, func(ds model.DataStore) model.DataStore {
			return failingAlbums{ds}
		})
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
		putAlbum(ctx, ds, "a1", "")
		kind, err := model.GetEntityKindByID(ctx, ds, "a1")
		Expect(err).ToNot(HaveOccurred())
		Expect(kind).To(Equal(model.KindAlbumArtwork))
	})

	It("returns ErrNotFound when no entity matches", func() {
		_, err := model.GetEntityKindByID(ctx, ds, "missing")
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})

func putAlbum(ctx context.Context, ds model.DataStore, id, name string) {
	GinkgoHelper()
	Expect(ds.Album().Put(ctx, &model.Album{ID: id, Name: name, LibraryID: model.DefaultLibraryID})).To(Succeed())
}

var errAlbumGet = errors.New("album lookup failed")

type failingAlbums struct{ model.DataStore }

func (f failingAlbums) Album() model.AlbumRepository { return failingAlbumRepo{f.DataStore.Album()} }

type failingAlbumRepo struct{ model.AlbumRepository }

func (failingAlbumRepo) Get(context.Context, string) (*model.Album, error) { return nil, errAlbumGet }
