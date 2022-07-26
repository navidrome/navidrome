package persistence

import (
	"context"
	"os"
	"path/filepath"

	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
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
		repo = NewAlbumRepository(ctx, orm.NewOrm())
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

	Describe("getMinYear", func() {
		It("returns 0 when there's no valid year", func() {
			Expect(getMinYear("a b c")).To(Equal(0))
			Expect(getMinYear("")).To(Equal(0))
		})
		It("returns 0 when all values are 0", func() {
			Expect(getMinYear("0 0 0 ")).To(Equal(0))
		})
		It("returns the smallest value from the list", func() {
			Expect(getMinYear("2000 0 1800")).To(Equal(1800))
		})
	})

	Describe("getComment", func() {
		const zwsp = string('\u200b')
		It("returns empty string if there are no comments", func() {
			Expect(getComment("", "")).To(Equal(""))
		})
		It("returns empty string if comments are different", func() {
			Expect(getComment("first"+zwsp+"second", zwsp)).To(Equal(""))
		})
		It("returns comment if all comments are the same", func() {
			Expect(getComment("first"+zwsp+"first", zwsp)).To(Equal("first"))
		})
	})

	Describe("getCoverFromPath", func() {
		var testFolder, testPath, embeddedPath string
		BeforeEach(func() {
			testFolder, _ = os.MkdirTemp("", "album_persistence_tests")
			if err := os.MkdirAll(testFolder, 0777); err != nil {
				panic(err)
			}
			if _, err := os.Create(filepath.Join(testFolder, "Cover.jpeg")); err != nil {
				panic(err)
			}
			if _, err := os.Create(filepath.Join(testFolder, "FRONT.PNG")); err != nil {
				panic(err)
			}
			testPath = filepath.Join(testFolder, "somefile.test")
			embeddedPath = filepath.Join(testFolder, "somefile.mp3")
		})
		AfterEach(func() {
			_ = os.RemoveAll(testFolder)
		})

		It("returns audio file for embedded cover", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
			Expect(getCoverFromPath(testPath, embeddedPath)).To(Equal(""))
		})

		It("returns external file when no embedded cover exists", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
			Expect(getCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "Cover.jpeg")))
		})

		It("returns embedded cover even if not first choice", func() {
			conf.Server.CoverArtPriority = "something.png, embedded, cover.*, front.*"
			Expect(getCoverFromPath(testPath, embeddedPath)).To(Equal(""))
		})

		It("returns first correct match case-insensitively", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jpg, front.svg, front.png"
			Expect(getCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "FRONT.PNG")))
		})

		It("returns match for embedded pattern", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jp?g, front.png"
			Expect(getCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "Cover.jpeg")))
		})

		It("returns empty string if no match was found", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jpg, front.apng"
			Expect(getCoverFromPath(testPath, "")).To(Equal(""))
		})

		// Reset configuration to default.
		conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
	})

	Describe("getAlbumArtist", func() {
		var al refreshAlbum
		BeforeEach(func() {
			al = refreshAlbum{}
		})
		Context("Non-Compilations", func() {
			BeforeEach(func() {
				al.Compilation = false
				al.Artist = "Sparks"
				al.ArtistID = "ar-123"
			})
			It("returns the track artist if no album artist is specified", func() {
				id, name := getAlbumArtist(al)
				Expect(id).To(Equal("ar-123"))
				Expect(name).To(Equal("Sparks"))
			})
			It("returns the album artist if it is specified", func() {
				al.AlbumArtist = "Sparks Brothers"
				al.AlbumArtistID = "ar-345"
				id, name := getAlbumArtist(al)
				Expect(id).To(Equal("ar-345"))
				Expect(name).To(Equal("Sparks Brothers"))
			})
		})
		Context("Compilations", func() {
			BeforeEach(func() {
				al.Compilation = true
				al.Name = "Sgt. Pepper Knew My Father"
				al.AlbumArtistID = "ar-000"
				al.AlbumArtist = "The Beatles"
			})

			It("returns VariousArtists if there's more than one album artist", func() {
				al.AlbumArtistIds = `ar-123 ar-345`
				id, name := getAlbumArtist(al)
				Expect(id).To(Equal(consts.VariousArtistsID))
				Expect(name).To(Equal(consts.VariousArtists))
			})

			It("returns the sole album artist if they are the same", func() {
				al.AlbumArtistIds = `ar-000 ar-000`
				id, name := getAlbumArtist(al)
				Expect(id).To(Equal("ar-000"))
				Expect(name).To(Equal("The Beatles"))
			})
		})
	})
})
