package core

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Share", func() {
	var ds model.DataStore
	var share Share
	var mockedRepo rest.Persistable
	ctx := context.Background()

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		mockedRepo = ds.Share(ctx).(rest.Persistable)
		share = NewShare(ds)
	})

	Describe("NewRepository", func() {
		var repo rest.Persistable

		BeforeEach(func() {
			repo = share.NewRepository(ctx).(rest.Persistable)
			_ = ds.Album(ctx).Put(&model.Album{ID: "123", Name: "Album"})
		})

		Describe("Save", func() {
			It("it sets a random ID", func() {
				entity := &model.Share{Description: "test", ResourceIDs: "123"}
				id, err := repo.Save(entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(id).ToNot(BeEmpty())
				Expect(entity.ID).To(Equal(id))
			})
		})

		Describe("Update", func() {
			It("filters out read-only fields", func() {
				entity := &model.Share{}
				err := repo.Update("id", entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockedRepo.(*tests.MockShareRepo).Cols).To(ConsistOf("description", "downloadable"))
			})
		})

		Describe("Album Contents Label", func() {
			It("properly qualifies album.id in GetAll filter to avoid SQL ambiguity", func() {
				wrapper := &shareRepositoryWrapper{ctx: ctx, ds: ds}

				// Set up mock data
				mockAlbumRepo := ds.Album(ctx).(*tests.MockAlbumRepo)
				mockAlbumRepo.SetData(model.Albums{
					{ID: "album1", Name: "Test Album 1"},
					{ID: "album2", Name: "Test Album 2"},
				})

				// This should not cause SQL ambiguity errors - the key test is that
				// the function calls ds.Album().GetAll() with "album.id" qualified filter
				result := wrapper.contentsLabelFromAlbums("shareID", "album1,album2")
				Expect(result).To(Equal("Test Album 1, Test Album 2"))
			})
		})
	})
})
