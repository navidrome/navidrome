package persistence

import (
	"context"

	"github.com/beego/beego/v2/client/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RadioInfoRepository", func() {
	var repo model.RadioInfoRepository
	BeforeEach(func() {
		ctx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: true})
		repo = NewRadioInfoRepository(ctx, orm.NewOrm())
	})

	Describe("Get", func() {
		It("returns an existing info", func() {
			Expect(repo.Get("23456")).To(Equal(&fullRadioWithoutMatch))
		})
		It("returns ErrNotFound when the album does not exist", func() {
			_, err := repo.Get("161")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetAll", func() {
		It("returns all records", func() {
			Expect(repo.GetAll()).To(Equal(testRadioInfo))
		})

		It("returns all records sorted", func() {
			Expect(repo.GetAll(model.QueryOptions{Sort: "name"})).To(Equal(model.RadioInfos{
				fullRadioWithoutMatch,
				fullRadioWithMatch,
			}))
		})

		It("returns all records sorted desc", func() {
			Expect(repo.GetAll(model.QueryOptions{Sort: "bitrate", Order: "desc"})).To(Equal(model.RadioInfos{
				fullRadioWithMatch,
				fullRadioWithoutMatch,
			}))
		})

		It("paginates the result", func() {
			Expect(repo.GetAll(model.QueryOptions{Offset: 1, Max: 1})).To(Equal(model.RadioInfos{
				fullRadioWithoutMatch,
			}))
		})

		Describe("ReadAll", func() {
			It("returns all records with tag 1", func() {
				Expect(repo.ReadAll(rest.QueryOptions{
					Sort: "country",
					Filters: map[string]interface{}{
						"tags": "tag1",
					},
				})).To(Equal(model.RadioInfos{
					fullRadioWithoutMatch,
					fullRadioWithMatch,
				}))
			})

			It("returns all records with tag 3 and tag 1", func() {
				Expect(repo.ReadAll(rest.QueryOptions{
					Sort: "country",
					Filters: map[string]interface{}{
						"tags": "tag3,tag1",
					},
				})).To(Equal(model.RadioInfos{
					fullRadioWithMatch,
				}))
			})

			It("parses existing filter correctly", func() {
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"existing": true,
					},
				})).To(Equal(model.RadioInfos{}))
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"existing": "true",
					},
				})).To(Equal(model.RadioInfos{}))
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"existing": false,
					},
				})).To(Equal(model.RadioInfos{
					fullRadioWithMatch,
					fullRadioWithoutMatch,
				}))
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"existing": "false",
					},
				})).To(Equal(model.RadioInfos{
					fullRadioWithMatch,
					fullRadioWithoutMatch,
				}))
			})

			It("searches for part of a name", func() {
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"name": "nate",
					},
				})).To(Equal(model.RadioInfos{fullRadioWithoutMatch}))
			})

			It("handles http(s) filters", func() {
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"https": true,
					},
				})).To(Equal(model.RadioInfos{
					fullRadioWithMatch,
				}))
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"https": "true",
					},
				})).To(Equal(model.RadioInfos{
					fullRadioWithMatch,
				}))
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"https": false,
					},
				})).To(Equal(model.RadioInfos{
					fullRadioWithoutMatch,
				}))
				Expect(repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"https": "false",
					},
				})).To(Equal(model.RadioInfos{
					fullRadioWithoutMatch,
				}))
			})
		})
	})
})
