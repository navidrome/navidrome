package persistence_test

import (
	"context"

	"github.com/beego/beego/v2/client/orm"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PublisherRepository", func() {
	var repo model.PublisherRepository

	BeforeEach(func() {
		repo = persistence.NewPublisherRepository(log.NewContext(context.TODO()), orm.NewOrm())
	})

	Describe("GetAll()", func() {
		It("returns all records", func() {
			publishers, err := repo.GetAll()
			Expect(err).To(BeNil())
			Expect(publishers).To(ConsistOf(
				model.Publisher{ID: "pn-1", Name: "Kling Klang", AlbumCount: 1, SongCount: 2},
				model.Publisher{ID: "pn-2", Name: "Apple", AlbumCount: 3, SongCount: 3},
			))
		})
	})
	Describe("Put()", Ordered, func() {
		It("does not insert existing publisher names", func() {
			g := model.Publisher{Name: "Apple"}
			err := repo.Put(&g)
			Expect(err).To(BeNil())
			Expect(g.ID).To(Equal("pn-2"))

			publishers, _ := repo.GetAll()
			Expect(publishers).To(HaveLen(2))
		})

		It("insert non-existent publisher names", func() {
			g := model.Publisher{Name: "Armada"}
			err := repo.Put(&g)
			Expect(err).To(BeNil())

			// ID is a uuid
			_, err = uuid.Parse(g.ID)
			Expect(err).To(BeNil())

			publishers, _ := repo.GetAll()
			Expect(publishers).To(HaveLen(3))
			Expect(publishers).To(ContainElement(model.Publisher{ID: g.ID, Name: "Armada", AlbumCount: 0, SongCount: 0}))
		})
	})
})
