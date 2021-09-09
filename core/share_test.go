package core

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Share", func() {
	var ds model.DataStore
	var share Share
	var mockedRepo rest.Persistable

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		mockedRepo = ds.Share(context.Background()).(rest.Persistable)
		share = NewShare(ds)
	})

	Describe("NewRepository", func() {
		var repo rest.Persistable

		BeforeEach(func() {
			repo = share.NewRepository(context.Background()).(rest.Persistable)
		})

		Describe("Save", func() {
			It("it adds a random name", func() {
				entity := &model.Share{Description: "test"}
				id, err := repo.Save(entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(id).ToNot(BeEmpty())
				Expect(entity.Name).ToNot(BeEmpty())
			})
		})

		Describe("Update", func() {
			It("filters out read-only fields", func() {
				entity := "entity"
				err := repo.Update(entity)
				Expect(err).ToNot(HaveOccurred())
				Expect(mockedRepo.(*tests.MockShareRepo).Entity).To(Equal("entity"))
				Expect(mockedRepo.(*tests.MockShareRepo).Cols).To(ConsistOf("description"))
			})
		})
	})
})
