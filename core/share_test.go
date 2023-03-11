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
	})
})
