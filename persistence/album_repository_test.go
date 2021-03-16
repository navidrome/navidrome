package persistence

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/astaxie/beego/orm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

var _ = Describe("AlbumRepository", func() {
	var repo model.AlbumRepository
	dirFS := os.DirFS(".")

	BeforeEach(func() {
		ctx := request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe"})
		repo = NewAlbumRepository(ctx, dirFS, orm.NewOrm())
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

	Describe("GetStarred", func() {
		It("returns all starred records", func() {
			Expect(repo.GetStarred(model.QueryOptions{})).To(Equal(model.Albums{
				albumRadioactivity,
			}))
		})
	})

	Describe("FindByArtist", func() {
		It("returns all records from a given ArtistID", func() {
			Expect(repo.FindByArtist("3")).To(Equal(model.Albums{
				albumSgtPeppers,
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
		It("returns first occurrence of non-empty comment", func() {
			Expect(getComment(zwsp+zwsp+"first"+zwsp+"second", zwsp)).To(Equal("first"))
		})
	})

	Describe("getCoverFromPath", func() {
		testFolder, _ := ioutil.TempDir("", "album_persistence_tests")
		if err := os.MkdirAll(testFolder, 0777); err != nil {
			panic(err)
		}
		if _, err := os.Create(filepath.Join(testFolder, "Cover.jpeg")); err != nil {
			panic(err)
		}
		if _, err := os.Create(filepath.Join(testFolder, "FRONT.PNG")); err != nil {
			panic(err)
		}

		testFS := os.DirFS(testFolder)
		testPath := "somefile.test"
		embeddedPath := "somefile.mp3"
		It("returns audio file for embedded cover", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
			Expect(getCoverFromPath(testFS, testPath, embeddedPath)).To(Equal(""))
		})

		It("returns external file when no embedded cover exists", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
			Expect(getCoverFromPath(testFS, testPath, "")).To(Equal("Cover.jpeg"))
		})

		It("returns embedded cover even if not first choice", func() {
			conf.Server.CoverArtPriority = "something.png, embedded, cover.*, front.*"
			Expect(getCoverFromPath(testFS, testPath, embeddedPath)).To(Equal(""))
		})

		It("returns first correct match case-insensitively", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jpg, front.svg, front.png"
			Expect(getCoverFromPath(testFS, testPath, "")).To(Equal("FRONT.PNG"))
		})

		It("returns match for embedded pattern", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jp?g, front.png"
			Expect(getCoverFromPath(testFS, testPath, "")).To(Equal("Cover.jpeg"))
		})

		It("returns empty string if no match was found", func() {
			conf.Server.CoverArtPriority = "embedded, cover.jpg, front.apng"
			Expect(getCoverFromPath(testFS, testPath, "")).To(Equal(""))
		})

		// Reset configuration to default.
		conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
	})
})
