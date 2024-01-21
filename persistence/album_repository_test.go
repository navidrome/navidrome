package persistence

import (
	"context"

	"github.com/fatih/structs"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AlbumRepository", func() {
	var repo model.AlbumRepository

	BeforeEach(func() {
		ctx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe"})
		repo = NewAlbumRepository(ctx, getDBXBuilder())
	})

	Describe("Get", func() {
		It("returns an existent album", func() {
			Expect(repo.Get("103")).To(Equal(&albumRadioactivity))
		})
		It("returns ErrNotFound when the album does not exist", func() {
			_, err := repo.Get("666")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetAll", func() {
		It("returns all records", func() {
			Expect(repo.GetAll()).To(Equal(testAlbums))
		})

		It("returns all records sorted", func() {
			Expect(repo.GetAll(model.QueryOptions{Sort: "name"})).To(Equal(model.Albums{
				albumAbbeyRoad,
				albumRadioactivity,
				albumSgtPeppers,
			}))
		})

		It("returns all records sorted desc", func() {
			Expect(repo.GetAll(model.QueryOptions{Sort: "name", Order: "desc"})).To(Equal(model.Albums{
				albumSgtPeppers,
				albumRadioactivity,
				albumAbbeyRoad,
			}))
		})

		It("paginates the result", func() {
			Expect(repo.GetAll(model.QueryOptions{Offset: 1, Max: 1})).To(Equal(model.Albums{
				albumAbbeyRoad,
			}))
		})
	})

	Describe("dbAlbum mapping", func() {
		var a *model.Album
		BeforeEach(func() {
			a = &model.Album{ID: "1", Name: "name", ArtistID: "2"}
		})
		It("maps empty discs field", func() {
			a.Discs = model.Discs{}
			dba := dbAlbum{Album: a}

			m := structs.Map(dba)
			Expect(dba.PostMapArgs(m)).To(Succeed())
			Expect(m).To(HaveKeyWithValue("discs", `{}`))

			other := dbAlbum{Album: &model.Album{ID: "1", Name: "name"}, Discs: "{}"}
			Expect(other.PostScan()).To(Succeed())

			Expect(other.Album.Discs).To(Equal(a.Discs))
		})
		It("maps the discs field", func() {
			a.Discs = model.Discs{1: "disc1", 2: "disc2"}
			dba := dbAlbum{Album: a}

			m := structs.Map(dba)
			Expect(dba.PostMapArgs(m)).To(Succeed())
			Expect(m).To(HaveKeyWithValue("discs", `{"1":"disc1","2":"disc2"}`))

			other := dbAlbum{Album: &model.Album{ID: "1", Name: "name"}, Discs: m["discs"].(string)}
			Expect(other.PostScan()).To(Succeed())

			Expect(other.Album.Discs).To(Equal(a.Discs))
		})
	})
})
