package persistence

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	NewId string = "123-456-789"
)

var _ = Describe("RadioRepository", func() {
	var repo model.RadioRepository

	Describe("Admin User", func() {
		BeforeEach(func() {
			ctx := log.NewContext(context.TODO())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			repo = NewRadioRepository(ctx, NewDBXBuilder(db.Db()))
			_ = repo.Put(&radioWithHomePage)
		})

		AfterEach(func() {
			all, _ := repo.GetAll()

			for _, radio := range all {
				_ = repo.Delete(radio.ID)
			}

			for i := range testRadios {
				r := testRadios[i]
				err := repo.Put(&r)
				if err != nil {
					panic(err)
				}
			}
		})

		Describe("Count", func() {
			It("returns the number of radios in the DB", func() {
				Expect(repo.CountAll()).To(Equal(int64(2)))
			})
		})

		Describe("Delete", func() {
			It("deletes existing item", func() {
				err := repo.Delete(radioWithHomePage.ID)

				Expect(err).To(BeNil())

				_, err = repo.Get(radioWithHomePage.ID)
				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("Get", func() {
			It("returns an existing item", func() {
				res, err := repo.Get(radioWithHomePage.ID)

				Expect(err).To(BeNil())
				Expect(res.ID).To(Equal(radioWithHomePage.ID))
			})

			It("errors when missing", func() {
				_, err := repo.Get("notanid")

				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("GetAll", func() {
			It("returns all items from the DB", func() {
				all, err := repo.GetAll()
				Expect(err).To(BeNil())
				Expect(all[0].ID).To(Equal(radioWithoutHomePage.ID))
				Expect(all[1].ID).To(Equal(radioWithHomePage.ID))
			})
		})

		Describe("Put", func() {
			It("successfully updates item", func() {
				err := repo.Put(&model.Radio{
					ID:        radioWithHomePage.ID,
					Name:      "New Name",
					StreamUrl: "https://example.com:4533/app",
				})

				Expect(err).To(BeNil())

				item, err := repo.Get(radioWithHomePage.ID)
				Expect(err).To(BeNil())

				Expect(item.HomePageUrl).To(Equal(""))
			})

			It("successfully creates item", func() {
				err := repo.Put(&model.Radio{
					Name:      "New radio",
					StreamUrl: "https://example.com:4533/app",
				})

				Expect(err).To(BeNil())
				Expect(repo.CountAll()).To(Equal(int64(3)))

				all, err := repo.GetAll()
				Expect(err).To(BeNil())
				Expect(all[2].StreamUrl).To(Equal("https://example.com:4533/app"))
			})
		})
	})

	Describe("Regular User", func() {
		BeforeEach(func() {
			ctx := log.NewContext(context.TODO())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: false})
			repo = NewRadioRepository(ctx, NewDBXBuilder(db.Db()))
		})

		Describe("Count", func() {
			It("returns the number of radios in the DB", func() {
				Expect(repo.CountAll()).To(Equal(int64(2)))
			})
		})

		Describe("Delete", func() {
			It("fails to delete items", func() {
				err := repo.Delete(radioWithHomePage.ID)

				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})

		Describe("Get", func() {
			It("returns an existing item", func() {
				res, err := repo.Get(radioWithHomePage.ID)

				Expect(err).To((BeNil()))
				Expect(res.ID).To(Equal(radioWithHomePage.ID))
			})

			It("errors when missing", func() {
				_, err := repo.Get("notanid")

				Expect(err).To(MatchError(model.ErrNotFound))
			})
		})

		Describe("GetAll", func() {
			It("returns all items from the DB", func() {
				all, err := repo.GetAll()
				Expect(err).To(BeNil())
				Expect(all[0].ID).To(Equal(radioWithoutHomePage.ID))
				Expect(all[1].ID).To(Equal(radioWithHomePage.ID))
			})
		})

		Describe("Put", func() {
			It("fails to update item", func() {
				err := repo.Put(&model.Radio{
					ID:        radioWithHomePage.ID,
					Name:      "New Name",
					StreamUrl: "https://example.com:4533/app",
				})

				Expect(err).To(Equal(rest.ErrPermissionDenied))
			})
		})
	})
})
