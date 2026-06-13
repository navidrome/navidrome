package model_test

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetEntityByID", func() {
	var (
		ctx context.Context
		ds  *tests.MockDataStore
	)

	BeforeEach(func() {
		ctx = context.Background()
		ds = &tests.MockDataStore{}
	})

	It("returns a Folder when found", func() {
		folder := model.Folder{ID: "folder-1", Name: "Jazz"}
		ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{folder})

		entity, err := model.GetEntityByID(ctx, ds, "folder-1")
		Expect(err).ToNot(HaveOccurred())
		Expect(entity).To(BeAssignableToTypeOf(&model.Folder{}))
		Expect(entity.(*model.Folder).ID).To(Equal("folder-1"))
	})

	It("returns a Folder before trying Artist for the same ID", func() {
		folder := model.Folder{ID: "shared-id", Name: "Folder"}
		artist := model.Artist{ID: "shared-id", Name: "Artist"}
		ds.Folder(ctx).(*tests.MockFolderRepo).SetData([]model.Folder{folder})
		ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{artist})

		entity, err := model.GetEntityByID(ctx, ds, "shared-id")
		Expect(err).ToNot(HaveOccurred())
		Expect(entity).To(BeAssignableToTypeOf(&model.Folder{}))
	})

	It("returns an Artist when no Folder matches", func() {
		artist := model.Artist{ID: "artist-1", Name: "Kraftwerk"}
		ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{artist})

		entity, err := model.GetEntityByID(ctx, ds, "artist-1")
		Expect(err).ToNot(HaveOccurred())
		Expect(entity).To(BeAssignableToTypeOf(&model.Artist{}))
	})

	It("returns an Album when no Folder or Artist matches", func() {
		album := model.Album{ID: "album-1", Name: "Radioactivity"}
		ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{album})

		entity, err := model.GetEntityByID(ctx, ds, "album-1")
		Expect(err).ToNot(HaveOccurred())
		Expect(entity).To(BeAssignableToTypeOf(&model.Album{}))
	})

	It("returns an error when no entity is found", func() {
		_, err := model.GetEntityByID(ctx, ds, "nonexistent")
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})
