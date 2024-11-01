package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
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
				album.ImportedAt = time.Time{}
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
				albums[i].ImportedAt = time.Time{}
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

	Describe("Album.PlayCount", func() {
		// Implementation is in withAnnotation() method
		DescribeTable("normalizes play count when AlbumPlayCountMode is absolute",
			func(songCount, playCount, expected int) {
				conf.Server.AlbumPlayCountMode = consts.AlbumPlayCountModeAbsolute

				newID := id.NewRandom()
				Expect(repo.Put(&model.Album{LibraryID: 1, ID: newID, Name: "name", SongCount: songCount})).To(Succeed())
				for i := 0; i < playCount; i++ {
					Expect(repo.IncPlayCount(newID, time.Now())).To(Succeed())
				}

				album, err := repo.Get(newID)
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

				newID := id.NewRandom()
				Expect(repo.Put(&model.Album{LibraryID: 1, ID: newID, Name: "name", SongCount: songCount})).To(Succeed())
				for i := 0; i < playCount; i++ {
					Expect(repo.IncPlayCount(newID, time.Now())).To(Succeed())
				}

				album, err := repo.Get(newID)
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

	Describe("dbAlbum mapping", func() {
		var (
			a    model.Album
			dba  *dbAlbum
			args map[string]any
		)

		BeforeEach(func() {
			a = al(model.Album{ID: "1", Name: "name"})
			dba = &dbAlbum{Album: &a, ParticipantIDs: "{}"}
			args = make(map[string]any)
		})

		Describe("PostScan", func() {
			It("parses Discs correctly", func() {
				dba.Discs = `{"1":"disc1","2":"disc2"}`
				Expect(dba.PostScan()).To(Succeed())
				Expect(dba.Album.Discs).To(Equal(model.Discs{1: "disc1", 2: "disc2"}))
			})

			It("parses ParticipantIDs correctly", func() {
				dba.ParticipantIDs = `{"composer":["1"],"artist":["2","3"]}`
				Expect(dba.PostScan()).To(Succeed())
				Expect(dba.Album.Participations).To(HaveLen(2))
				Expect(dba.Album.Participations).To(HaveKeyWithValue(
					model.RoleFromString("composer"),
					[]model.Artist{{ID: "1"}},
				))
				Expect(dba.Album.Participations).To(HaveKeyWithValue(
					model.RoleFromString("artist"),
					[]model.Artist{{ID: "2"}, {ID: "3"}},
				))
			})

			It("parses Tags correctly", func() {
				dba.Tags = `{"genre":[{"id":"1","value":"rock"},{"id":"2","value":"pop"}],"mood":[{"id":"3","value":"happy"}]}`
				Expect(dba.PostScan()).To(Succeed())
				Expect(dba.Album.Tags).To(HaveKeyWithValue(
					model.TagName("mood"), []string{"happy"},
				))
				Expect(dba.Album.Tags).To(HaveKeyWithValue(
					model.TagName("genre"), []string{"rock", "pop"},
				))
				Expect(dba.Album.Genre).To(Equal("rock"))
				Expect(dba.Album.Genres).To(HaveLen(2))
			})
		})

		Describe("PostMapArgs", func() {
			It("maps full_text correctly", func() {
				Expect(dba.PostMapArgs(args)).To(Succeed())
				Expect(args).To(HaveKeyWithValue("full_text", " name"))
			})

			It("maps tags correctly", func() {
				dba.Album.Tags = model.Tags{"genre": {"rock", "pop"}, "mood": {"happy"}}
				Expect(dba.PostMapArgs(args)).To(Succeed())
				Expect(args).To(HaveKeyWithValue("tags",
					`{"genre":[{"id":"5qDZoz1FBC36K73YeoJ2lF","value":"rock"},{"id":"4H0KjnlS2ob9nKLL0zHOqB",`+
						`"value":"pop"}],"mood":[{"id":"1F4tmb516DIlHKFT1KzE1Z","value":"happy"}]}`,
				))
			})

			It("maps participant_ids correctly", func() {
				dba.Album.Participations = model.Participations{
					model.RoleAlbumArtist: {{ID: "AA1", Name: "AlbumArtist1"}},
					model.RoleComposer:    {{ID: "C1", Name: "Composer1"}},
				}
				Expect(dba.PostMapArgs(args)).To(Succeed())
				Expect(args).To(HaveKeyWithValue(
					"participant_ids",
					`{"album_artist":["AA1"],"composer":["C1"]}`,
				))
			})

			It("maps discs correctly", func() {
				dba.Album.Discs = model.Discs{1: "disc1", 2: "disc2"}
				Expect(dba.PostMapArgs(args)).To(Succeed())
				Expect(args).To(HaveKeyWithValue("discs", `{"1":"disc1","2":"disc2"}`))
			})
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
