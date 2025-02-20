package persistence

import (
	"context"
	"encoding/json"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtistRepository", func() {
	var repo model.ArtistRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, model.User{ID: "userid"})
		repo = NewArtistRepository(ctx, GetDBXBuilder())
	})

	Describe("Count", func() {
		It("returns the number of artists in the DB", func() {
			Expect(repo.CountAll()).To(Equal(int64(2)))
		})
	})

	Describe("Exists", func() {
		It("returns true for an artist that is in the DB", func() {
			Expect(repo.Exists("3")).To(BeTrue())
		})
		It("returns false for an artist that is in the DB", func() {
			Expect(repo.Exists("666")).To(BeFalse())
		})
	})

	Describe("Get", func() {
		It("saves and retrieves data", func() {
			artist, err := repo.Get("2")
			Expect(err).ToNot(HaveOccurred())
			Expect(artist.Name).To(Equal(artistKraftwerk.Name))
		})
	})

	Describe("GetIndexKey", func() {
		// Note: OrderArtistName should never be empty, so we don't need to test for that
		r := artistRepository{indexGroups: utils.ParseIndexGroups(conf.Server.IndexGroups)}
		When("PreferSortTags is false", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig)
				conf.Server.PreferSortTags = false
			})
			It("returns the OrderArtistName key is SortArtistName is empty", func() {
				conf.Server.PreferSortTags = false
				a := model.Artist{SortArtistName: "", OrderArtistName: "Bar", Name: "Qux"}
				idx := GetIndexKey(&r, a)
				Expect(idx).To(Equal("B"))
			})
			It("returns the OrderArtistName key even if SortArtistName is not empty", func() {
				a := model.Artist{SortArtistName: "Foo", OrderArtistName: "Bar", Name: "Qux"}
				idx := GetIndexKey(&r, a)
				Expect(idx).To(Equal("B"))
			})
		})
		When("PreferSortTags is true", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig)
				conf.Server.PreferSortTags = true
			})
			It("returns the SortArtistName key if it is not empty", func() {
				a := model.Artist{SortArtistName: "Foo", OrderArtistName: "Bar", Name: "Qux"}
				idx := GetIndexKey(&r, a)
				Expect(idx).To(Equal("F"))
			})
			It("returns the OrderArtistName key if SortArtistName is empty", func() {
				a := model.Artist{SortArtistName: "", OrderArtistName: "Bar", Name: "Qux"}
				idx := GetIndexKey(&r, a)
				Expect(idx).To(Equal("B"))
			})
		})
	})

	Describe("GetIndex", func() {
		When("PreferSortTags is true", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.PreferSortTags = true
			})
			It("returns the index when PreferSortTags is true and SortArtistName is not empty", func() {
				// Set SortArtistName to "Foo" for Beatles
				artistBeatles.SortArtistName = "Foo"
				er := repo.Put(&artistBeatles)
				Expect(er).To(BeNil())

				idx, err := repo.GetIndex()
				Expect(err).ToNot(HaveOccurred())
				Expect(idx).To(HaveLen(2))
				Expect(idx[0].ID).To(Equal("F"))
				Expect(idx[0].Artists).To(HaveLen(1))
				Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
				Expect(idx[1].ID).To(Equal("K"))
				Expect(idx[1].Artists).To(HaveLen(1))
				Expect(idx[1].Artists[0].Name).To(Equal(artistKraftwerk.Name))

				// Restore the original value
				artistBeatles.SortArtistName = ""
				er = repo.Put(&artistBeatles)
				Expect(er).To(BeNil())
			})

			// BFR Empty SortArtistName is not saved in the DB anymore
			XIt("returns the index when PreferSortTags is true and SortArtistName is empty", func() {
				idx, err := repo.GetIndex()
				Expect(err).ToNot(HaveOccurred())
				Expect(idx).To(HaveLen(2))
				Expect(idx[0].ID).To(Equal("B"))
				Expect(idx[0].Artists).To(HaveLen(1))
				Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
				Expect(idx[1].ID).To(Equal("K"))
				Expect(idx[1].Artists).To(HaveLen(1))
				Expect(idx[1].Artists[0].Name).To(Equal(artistKraftwerk.Name))
			})
		})

		When("PreferSortTags is false", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.PreferSortTags = false
			})
			It("returns the index when SortArtistName is NOT empty", func() {
				// Set SortArtistName to "Foo" for Beatles
				artistBeatles.SortArtistName = "Foo"
				er := repo.Put(&artistBeatles)
				Expect(er).To(BeNil())

				idx, err := repo.GetIndex()
				Expect(err).ToNot(HaveOccurred())
				Expect(idx).To(HaveLen(2))
				Expect(idx[0].ID).To(Equal("B"))
				Expect(idx[0].Artists).To(HaveLen(1))
				Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
				Expect(idx[1].ID).To(Equal("K"))
				Expect(idx[1].Artists).To(HaveLen(1))
				Expect(idx[1].Artists[0].Name).To(Equal(artistKraftwerk.Name))

				// Restore the original value
				artistBeatles.SortArtistName = ""
				er = repo.Put(&artistBeatles)
				Expect(er).To(BeNil())
			})

			It("returns the index when SortArtistName is empty", func() {
				idx, err := repo.GetIndex()
				Expect(err).ToNot(HaveOccurred())
				Expect(idx).To(HaveLen(2))
				Expect(idx[0].ID).To(Equal("B"))
				Expect(idx[0].Artists).To(HaveLen(1))
				Expect(idx[0].Artists[0].Name).To(Equal(artistBeatles.Name))
				Expect(idx[1].ID).To(Equal("K"))
				Expect(idx[1].Artists).To(HaveLen(1))
				Expect(idx[1].Artists[0].Name).To(Equal(artistKraftwerk.Name))
			})
		})
	})

	Describe("dbArtist mapping", func() {
		var (
			artist *model.Artist
			dba    *dbArtist
		)

		BeforeEach(func() {
			artist = &model.Artist{ID: "1", Name: "Eddie Van Halen", SortArtistName: "Van Halen, Eddie"}
			dba = &dbArtist{Artist: artist}
		})

		Describe("PostScan", func() {
			It("parses stats and similar artists correctly", func() {
				stats := map[string]map[string]int64{
					"total":    {"s": 1000, "m": 10, "a": 2},
					"composer": {"s": 500, "m": 5, "a": 1},
				}
				statsJSON, _ := json.Marshal(stats)
				dba.Stats = string(statsJSON)
				dba.SimilarArtists = `[{"id":"2","Name":"AC/DC"},{"name":"Test;With:Sep,Chars"}]`

				err := dba.PostScan()
				Expect(err).ToNot(HaveOccurred())
				Expect(dba.Artist.Size).To(Equal(int64(1000)))
				Expect(dba.Artist.SongCount).To(Equal(10))
				Expect(dba.Artist.AlbumCount).To(Equal(2))
				Expect(dba.Artist.Stats).To(HaveLen(1))
				Expect(dba.Artist.Stats[model.RoleFromString("composer")].Size).To(Equal(int64(500)))
				Expect(dba.Artist.Stats[model.RoleFromString("composer")].SongCount).To(Equal(5))
				Expect(dba.Artist.Stats[model.RoleFromString("composer")].AlbumCount).To(Equal(1))
				Expect(dba.Artist.SimilarArtists).To(HaveLen(2))
				Expect(dba.Artist.SimilarArtists[0].ID).To(Equal("2"))
				Expect(dba.Artist.SimilarArtists[0].Name).To(Equal("AC/DC"))
				Expect(dba.Artist.SimilarArtists[1].ID).To(BeEmpty())
				Expect(dba.Artist.SimilarArtists[1].Name).To(Equal("Test;With:Sep,Chars"))
			})
		})

		Describe("PostMapArgs", func() {
			It("maps empty similar artists correctly", func() {
				m := make(map[string]any)
				err := dba.PostMapArgs(m)
				Expect(err).ToNot(HaveOccurred())
				Expect(m).To(HaveKeyWithValue("similar_artists", "[]"))
			})

			It("maps similar artists and full text correctly", func() {
				artist.SimilarArtists = []model.Artist{
					{ID: "2", Name: "AC/DC"},
					{Name: "Test;With:Sep,Chars"},
				}
				m := make(map[string]any)
				err := dba.PostMapArgs(m)
				Expect(err).ToNot(HaveOccurred())
				Expect(m).To(HaveKeyWithValue("similar_artists", `[{"id":"2","name":"AC/DC"},{"name":"Test;With:Sep,Chars"}]`))
				Expect(m).To(HaveKeyWithValue("full_text", " eddie halen van"))
			})

			It("does not override empty sort_artist_name and mbz_artist_id", func() {
				m := map[string]any{
					"sort_artist_name": "",
					"mbz_artist_id":    "",
				}
				err := dba.PostMapArgs(m)
				Expect(err).ToNot(HaveOccurred())
				Expect(m).ToNot(HaveKey("sort_artist_name"))
				Expect(m).ToNot(HaveKey("mbz_artist_id"))
			})
		})
	})
})
