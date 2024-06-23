package persistence

import (
	"context"
	"time"

	"github.com/fatih/structs"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
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
		repo = NewAlbumRepository(ctx, NewDBXBuilder(db.Db()))
	})

	Describe("Get", func() {
		var Get = func(id string) (*model.Album, error) {
			album, err := repo.Get(id)
			if album != nil {
				album.ScannedAt = time.Time{}
			}
			return album, err
		}
		It("returns an existent album", func() {
			Expect(Get("103")).To(Equal(&albumRadioactivity))
		})
		It("returns ErrNotFound when the album does not exist", func() {
			_, err := Get("666")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("GetAll", func() {
		var GetAll = func(opts ...model.QueryOptions) (model.Albums, error) {
			albums, err := repo.GetAll(opts...)
			for i := range albums {
				albums[i].ScannedAt = time.Time{}
			}
			return albums, err
		}

		It("returns all records", func() {
			Expect(GetAll()).To(Equal(testAlbums))
		})

		It("returns all records sorted", func() {
			Expect(GetAll(model.QueryOptions{Sort: "name"})).To(Equal(model.Albums{
				albumAbbeyRoad,
				albumRadioactivity,
				albumSgtPeppers,
			}))
		})

		It("returns all records sorted desc", func() {
			Expect(GetAll(model.QueryOptions{Sort: "name", Order: "desc"})).To(Equal(model.Albums{
				albumSgtPeppers,
				albumRadioactivity,
				albumAbbeyRoad,
			}))
		})

		It("paginates the result", func() {
			Expect(GetAll(model.QueryOptions{Offset: 1, Max: 1})).To(Equal(model.Albums{
				albumAbbeyRoad,
			}))
		})
	})

	Describe("dbAlbum mapping", func() {
		Describe("Album.Discs", func() {
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
		Describe("Album.Genres", func() {
			var a *model.Album
			BeforeEach(func() {
				a = &model.Album{ID: "1", Name: "name", ArtistID: "2"}
			})
			It("maps empty tags field", func() {
				dba := dbAlbum{Album: a}

				m := structs.Map(dba)
				Expect(dba.PostMapArgs(m)).To(Succeed())
				Expect(m).ToNot(HaveKey("tags"))

				other := dbAlbum{Album: &model.Album{ID: "1", Name: "name"}}
				Expect(other.PostScan()).To(Succeed())

				Expect(other.Album.Genres).To(BeNil())
			})
			It("maps the tags field", func() {
				a.Tags = model.Tags{"genre": {"rock", "pop"}}
				dba := dbAlbum{Album: a}

				m := structs.Map(dba)
				Expect(dba.PostMapArgs(m)).To(Succeed())
				Expect(m).ToNot(HaveKey("tags"))

				other := dbAlbum{Album: &model.Album{ID: "1", Name: "name"}, Tags: `[{"genre":"rock"},{"genre":"pop"},{"genre":"rock"}]`}
				Expect(other.PostScan()).To(Succeed())

				Expect(other.Album.Genre).To(Equal("rock"))
				Expect(other.Album.Genres).To(HaveLen(3))
				Expect(other.Album.Genres[0].Name).To(Equal("rock"))
				Expect(other.Album.Genres[1].Name).To(Equal("pop"))
				Expect(other.Album.Genres[2].Name).To(Equal("rock"))
			})
		})
		Describe("Album.PlayCount", func() {
			DescribeTable("normalizes play count when AlbumPlayCountMode is absolute",
				func(songCount, playCount, expected int) {
					conf.Server.AlbumPlayCountMode = consts.AlbumPlayCountModeAbsolute

					id := uuid.NewString()
					Expect(repo.Put(&model.Album{LibraryID: 1, ID: id, Name: "name", SongCount: songCount})).To(Succeed())
					for i := 0; i < playCount; i++ {
						Expect(repo.IncPlayCount(id, time.Now())).To(Succeed())
					}

					album, err := repo.Get(id)
					Expect(err).ToNot(HaveOccurred())
					Expect(album.PlayCount).To(Equal(int64(expected)))
				},
				Entry("1 song, 0 plays", 1, 0, 0),
				Entry("1 song, 4 plays", 1, 4, 4),
				Entry("3 songs, 6 plays", 3, 6, 6),
				Entry("10 songs, 6 plays", 10, 6, 6),
				Entry("70 songs, 70 plays", 70, 70, 70),
				Entry("10 songs, 50 plays", 10, 50, 50),
				Entry("120 songs, 121 plays", 120, 121, 121),
			)

			DescribeTable("normalizes play count when AlbumPlayCountMode is normalized",
				func(songCount, playCount, expected int) {
					conf.Server.AlbumPlayCountMode = consts.AlbumPlayCountModeNormalized

					id := uuid.NewString()
					Expect(repo.Put(&model.Album{LibraryID: 1, ID: id, Name: "name", SongCount: songCount})).To(Succeed())
					for i := 0; i < playCount; i++ {
						Expect(repo.IncPlayCount(id, time.Now())).To(Succeed())
					}

					album, err := repo.Get(id)
					Expect(err).ToNot(HaveOccurred())
					Expect(album.PlayCount).To(Equal(int64(expected)))
				},
				Entry("1 song, 0 plays", 1, 0, 0),
				Entry("1 song, 4 plays", 1, 4, 4),
				Entry("3 songs, 6 plays", 3, 6, 2),
				Entry("10 songs, 6 plays", 10, 6, 1),
				Entry("70 songs, 70 plays", 70, 70, 1),
				Entry("10 songs, 50 plays", 10, 50, 5),
				Entry("120 songs, 121 plays", 120, 121, 1),
			)
		})

		Describe("AllArtistIDs", func() {
			It("collects all deduplicated artist ids", func() {
				dba := dbAlbum{Album: &model.Album{
					Participations: model.Participations{
						model.RoleAlbumArtist: {{ID: "AA1", Name: "AlbumArtist1", SortArtistName: "SortAlbumArtistName1"}},
						model.RoleArtist:      {{ID: "A2", Name: "Artist2", SortArtistName: "SortArtistName2"}},
						model.RoleComposer:    {{ID: "C1", Name: "Composer1"}},
					},
				}}

				mapped := map[string]any{}
				err := dba.PostMapArgs(mapped)
				Expect(err).ToNot(HaveOccurred())
				Expect(mapped).To(HaveKeyWithValue("all_artist_ids", "A2 AA1 C1"))
			})
		})

		Describe("dbAlbums.toModels", func() {
			It("converts dbAlbums to model.Albums", func() {
				dba := dbAlbums{
					{Album: &model.Album{ID: "1", Name: "name", SongCount: 2, Annotations: model.Annotations{PlayCount: 4}}},
					{Album: &model.Album{ID: "2", Name: "name2", SongCount: 3, Annotations: model.Annotations{PlayCount: 6}}},
				}
				albums := dba.toModels()
				for i := range dba {
					Expect(albums[i].ID).To(Equal(dba[i].Album.ID))
					Expect(albums[i].Name).To(Equal(dba[i].Album.Name))
					Expect(albums[i].SongCount).To(Equal(dba[i].Album.SongCount))
					Expect(albums[i].PlayCount).To(Equal(dba[i].Album.PlayCount))
				}
			})
		})
	})
})
