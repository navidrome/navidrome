package persistence

import (
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AlbumRepository", func() {
	var albumRepo *albumRepository

	BeforeEach(func() {
		ctx := request.WithUser(GinkgoT().Context(), model.User{ID: "userid", UserName: "johndoe"})
		albumRepo = NewAlbumRepository(ctx, GetDBXBuilder()).(*albumRepository)
	})

	Describe("Get", func() {
		var Get = func(id string) (*model.Album, error) {
			album, err := albumRepo.Get(id)
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
			albums, err := albumRepo.GetAll(opts...)
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
				Expect(albumRepo.Put(&model.Album{LibraryID: 1, ID: newID, Name: "name", SongCount: songCount})).To(Succeed())
				for i := 0; i < playCount; i++ {
					Expect(albumRepo.IncPlayCount(newID, time.Now())).To(Succeed())
				}

				album, err := albumRepo.Get(newID)
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
				Expect(albumRepo.Put(&model.Album{LibraryID: 1, ID: newID, Name: "name", SongCount: songCount})).To(Succeed())
				for i := 0; i < playCount; i++ {
					Expect(albumRepo.IncPlayCount(newID, time.Now())).To(Succeed())
				}

				album, err := albumRepo.Get(newID)
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
			dba = &dbAlbum{Album: &a, Participants: "{}"}
			args = make(map[string]any)
		})

		Describe("PostScan", func() {
			It("parses Discs correctly", func() {
				dba.Discs = `{"1":"disc1","2":"disc2"}`
				Expect(dba.PostScan()).To(Succeed())
				Expect(dba.Album.Discs).To(Equal(model.Discs{1: "disc1", 2: "disc2"}))
			})

			It("parses Participants correctly", func() {
				dba.Participants = `{"composer":[{"id":"1","name":"Composer 1"}],` +
					`"artist":[{"id":"2","name":"Artist 2"},{"id":"3","name":"Artist 3","subRole":"subRole"}]}`
				Expect(dba.PostScan()).To(Succeed())
				Expect(dba.Album.Participants).To(HaveLen(2))
				Expect(dba.Album.Participants).To(HaveKeyWithValue(
					model.RoleFromString("composer"),
					model.ParticipantList{{Artist: model.Artist{ID: "1", Name: "Composer 1"}}},
				))
				Expect(dba.Album.Participants).To(HaveKeyWithValue(
					model.RoleFromString("artist"),
					model.ParticipantList{{Artist: model.Artist{ID: "2", Name: "Artist 2"}}, {Artist: model.Artist{ID: "3", Name: "Artist 3"}, SubRole: "subRole"}},
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

			It("parses Paths correctly", func() {
				dba.FolderIDs = `["folder1","folder2"]`
				Expect(dba.PostScan()).To(Succeed())
				Expect(dba.Album.FolderIDs).To(Equal([]string{"folder1", "folder2"}))
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

			It("maps participants correctly", func() {
				dba.Album.Participants = model.Participants{
					model.RoleAlbumArtist: model.ParticipantList{_p("AA1", "AlbumArtist1")},
					model.RoleComposer:    model.ParticipantList{{Artist: model.Artist{ID: "C1", Name: "Composer1"}, SubRole: "composer"}},
				}
				Expect(dba.PostMapArgs(args)).To(Succeed())
				Expect(args).To(HaveKeyWithValue(
					"participants",
					`{"albumartist":[{"id":"AA1","name":"AlbumArtist1"}],`+
						`"composer":[{"id":"C1","name":"Composer1","subRole":"composer"}]}`,
				))
			})

			It("maps discs correctly", func() {
				dba.Album.Discs = model.Discs{1: "disc1", 2: "disc2"}
				Expect(dba.PostMapArgs(args)).To(Succeed())
				Expect(args).To(HaveKeyWithValue("discs", `{"1":"disc1","2":"disc2"}`))
			})

			It("maps paths correctly", func() {
				dba.Album.FolderIDs = []string{"folder1", "folder2"}
				Expect(dba.PostMapArgs(args)).To(Succeed())
				Expect(args).To(HaveKeyWithValue("folder_ids", `["folder1","folder2"]`))
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

	Describe("artistRoleFilter", func() {
		DescribeTable("creates correct SQL expressions for artist roles",
			func(filterName, artistID, expectedSQL string) {
				sqlizer := artistRoleFilter(filterName, artistID)
				sql, args, err := sqlizer.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(expectedSQL))
				Expect(args).To(Equal([]interface{}{artistID}))
			},
			Entry("artist role", "role_artist_id", "123",
				"exists (select 1 from json_tree(participants, '$.artist') where value = ?)"),
			Entry("albumartist role", "role_albumartist_id", "456",
				"exists (select 1 from json_tree(participants, '$.albumartist') where value = ?)"),
			Entry("composer role", "role_composer_id", "789",
				"exists (select 1 from json_tree(participants, '$.composer') where value = ?)"),
		)

		It("works with the actual filter map", func() {
			filters := albumFilters()

			for roleName := range model.AllRoles {
				filterName := "role_" + roleName + "_id"
				filterFunc, exists := filters[filterName]
				Expect(exists).To(BeTrue(), fmt.Sprintf("Filter %s should exist", filterName))

				sqlizer := filterFunc(filterName, "test-id")
				sql, args, err := sqlizer.ToSql()
				Expect(err).ToNot(HaveOccurred())
				Expect(sql).To(Equal(fmt.Sprintf("exists (select 1 from json_tree(participants, '$.%s') where value = ?)", roleName)))
				Expect(args).To(Equal([]interface{}{"test-id"}))
			}
		})

		It("rejects invalid roles", func() {
			sqlizer := artistRoleFilter("role_invalid_id", "123")
			_, _, err := sqlizer.ToSql()
			Expect(err).To(HaveOccurred())
		})

		It("rejects invalid filter names", func() {
			sqlizer := artistRoleFilter("invalid_name", "123")
			_, _, err := sqlizer.ToSql()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Participant Foreign Key Handling", func() {
		// albumArtistRecord represents a record in the album_artists table
		type albumArtistRecord struct {
			ArtistID string `db:"artist_id"`
			Role     string `db:"role"`
			SubRole  string `db:"sub_role"`
		}

		var artistRepo *artistRepository

		BeforeEach(func() {
			ctx := request.WithUser(GinkgoT().Context(), adminUser)
			artistRepo = NewArtistRepository(ctx, GetDBXBuilder()).(*artistRepository)
		})

		// Helper to verify album_artists records
		verifyAlbumArtists := func(albumID string, expected []albumArtistRecord) {
			GinkgoHelper()
			var actual []albumArtistRecord
			sq := squirrel.Select("artist_id", "role", "sub_role").
				From("album_artists").
				Where(squirrel.Eq{"album_id": albumID}).
				OrderBy("role", "artist_id", "sub_role")

			err := albumRepo.queryAll(sq, &actual)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(expected))
		}

		It("verifies that participant records are actually inserted into database", func() {
			// Create a real artist in the database first
			artist := &model.Artist{
				ID:              "real-artist-1",
				Name:            "Real Artist",
				OrderArtistName: "real artist",
				SortArtistName:  "Artist, Real",
			}
			err := createArtistWithLibrary(artistRepo, artist, 1)
			Expect(err).ToNot(HaveOccurred())

			// Create an album with participants that reference the real artist
			album := &model.Album{
				LibraryID:     1,
				ID:            "test-album-db-insert",
				Name:          "Test Album DB Insert",
				AlbumArtistID: "real-artist-1",
				AlbumArtist:   "Real Artist",
				Participants: model.Participants{
					model.RoleArtist: {
						{Artist: model.Artist{ID: "real-artist-1", Name: "Real Artist"}},
					},
					model.RoleComposer: {
						{Artist: model.Artist{ID: "real-artist-1", Name: "Real Artist"}, SubRole: "primary"},
					},
				},
			}

			// Insert the album
			err = albumRepo.Put(album)
			Expect(err).ToNot(HaveOccurred())

			// Verify that participant records were actually inserted into album_artists table
			expected := []albumArtistRecord{
				{ArtistID: "real-artist-1", Role: "artist", SubRole: ""},
				{ArtistID: "real-artist-1", Role: "composer", SubRole: "primary"},
			}
			verifyAlbumArtists(album.ID, expected)

			// Clean up the test artist and album created for this test
			_, _ = artistRepo.executeSQL(squirrel.Delete("artist").Where(squirrel.Eq{"id": artist.ID}))
			_, _ = albumRepo.executeSQL(squirrel.Delete("album").Where(squirrel.Eq{"id": album.ID}))
		})

		It("filters out invalid artist IDs leaving only valid participants in database", func() {
			// Create two real artists in the database
			artist1 := &model.Artist{
				ID:              "real-artist-mix-1",
				Name:            "Real Artist 1",
				OrderArtistName: "real artist 1",
			}
			artist2 := &model.Artist{
				ID:              "real-artist-mix-2",
				Name:            "Real Artist 2",
				OrderArtistName: "real artist 2",
			}
			err := createArtistWithLibrary(artistRepo, artist1, 1)
			Expect(err).ToNot(HaveOccurred())
			err = createArtistWithLibrary(artistRepo, artist2, 1)
			Expect(err).ToNot(HaveOccurred())

			// Create an album with mix of valid and invalid artist IDs
			album := &model.Album{
				LibraryID:     1,
				ID:            "test-album-mixed-validity",
				Name:          "Test Album Mixed Validity",
				AlbumArtistID: "real-artist-mix-1",
				AlbumArtist:   "Real Artist 1",
				Participants: model.Participants{
					model.RoleArtist: {
						{Artist: model.Artist{ID: "real-artist-mix-1", Name: "Real Artist 1"}},
						{Artist: model.Artist{ID: "non-existent-mix-1", Name: "Non Existent 1"}},
						{Artist: model.Artist{ID: "real-artist-mix-2", Name: "Real Artist 2"}},
					},
					model.RoleComposer: {
						{Artist: model.Artist{ID: "non-existent-mix-2", Name: "Non Existent 2"}},
						{Artist: model.Artist{ID: "real-artist-mix-1", Name: "Real Artist 1"}},
					},
				},
			}

			// This should not fail - only valid artists should be inserted
			err = albumRepo.Put(album)
			Expect(err).ToNot(HaveOccurred())

			// Verify that only valid artist IDs were inserted into album_artists table
			// Non-existent artists should be filtered out by the INNER JOIN
			expected := []albumArtistRecord{
				{ArtistID: "real-artist-mix-1", Role: "artist", SubRole: ""},
				{ArtistID: "real-artist-mix-2", Role: "artist", SubRole: ""},
				{ArtistID: "real-artist-mix-1", Role: "composer", SubRole: ""},
			}
			verifyAlbumArtists(album.ID, expected)

			// Clean up the test artists and album created for this test
			artistIDs := []string{artist1.ID, artist2.ID}
			_, _ = artistRepo.executeSQL(squirrel.Delete("artist").Where(squirrel.Eq{"id": artistIDs}))
			_, _ = albumRepo.executeSQL(squirrel.Delete("album").Where(squirrel.Eq{"id": album.ID}))
		})

		It("handles complex nested JSON with multiple roles and sub-roles", func() {
			// Create 4 artists for this test
			artists := []*model.Artist{
				{ID: "complex-artist-1", Name: "Lead Vocalist", OrderArtistName: "lead vocalist"},
				{ID: "complex-artist-2", Name: "Guitarist", OrderArtistName: "guitarist"},
				{ID: "complex-artist-3", Name: "Producer", OrderArtistName: "producer"},
				{ID: "complex-artist-4", Name: "Engineer", OrderArtistName: "engineer"},
			}

			for _, artist := range artists {
				err := createArtistWithLibrary(artistRepo, artist, 1)
				Expect(err).ToNot(HaveOccurred())
			}

			// Create album with complex participant structure
			album := &model.Album{
				LibraryID:     1,
				ID:            "test-album-complex-json",
				Name:          "Test Album Complex JSON",
				AlbumArtistID: "complex-artist-1",
				AlbumArtist:   "Lead Vocalist",
				Participants: model.Participants{
					model.RoleArtist: {
						{Artist: model.Artist{ID: "complex-artist-1", Name: "Lead Vocalist"}},
						{Artist: model.Artist{ID: "complex-artist-2", Name: "Guitarist"}, SubRole: "lead guitar"},
						{Artist: model.Artist{ID: "complex-artist-2", Name: "Guitarist"}, SubRole: "rhythm guitar"},
					},
					model.RoleProducer: {
						{Artist: model.Artist{ID: "complex-artist-3", Name: "Producer"}, SubRole: "executive"},
					},
					model.RoleEngineer: {
						{Artist: model.Artist{ID: "complex-artist-4", Name: "Engineer"}, SubRole: "mixing"},
						{Artist: model.Artist{ID: "complex-artist-4", Name: "Engineer"}, SubRole: "mastering"},
					},
				},
			}

			err := albumRepo.Put(album)
			Expect(err).ToNot(HaveOccurred())

			// Verify complex JSON structure was correctly parsed and inserted
			expected := []albumArtistRecord{
				{ArtistID: "complex-artist-1", Role: "artist", SubRole: ""},
				{ArtistID: "complex-artist-2", Role: "artist", SubRole: "lead guitar"},
				{ArtistID: "complex-artist-2", Role: "artist", SubRole: "rhythm guitar"},
				{ArtistID: "complex-artist-4", Role: "engineer", SubRole: "mastering"},
				{ArtistID: "complex-artist-4", Role: "engineer", SubRole: "mixing"},
				{ArtistID: "complex-artist-3", Role: "producer", SubRole: "executive"},
			}
			verifyAlbumArtists(album.ID, expected)

			// Clean up the test artists and album created for this test
			artistIDs := make([]string, len(artists))
			for i, artist := range artists {
				artistIDs[i] = artist.ID
			}
			_, _ = artistRepo.executeSQL(squirrel.Delete("artist").Where(squirrel.Eq{"id": artistIDs}))
			_, _ = albumRepo.executeSQL(squirrel.Delete("album").Where(squirrel.Eq{"id": album.ID}))
		})

		It("handles albums with non-existent artist IDs without constraint errors", func() {
			// Regression test for foreign key constraint error when album participants
			// contain artist IDs that don't exist in the artist table

			// Create an album with participants that reference non-existent artist IDs
			album := &model.Album{
				LibraryID:     1,
				ID:            "test-album-fk-constraints",
				Name:          "Test Album with Invalid Artist References",
				AlbumArtistID: "non-existent-artist-1",
				AlbumArtist:   "Non Existent Album Artist",
				Participants: model.Participants{
					model.RoleArtist: {
						{Artist: model.Artist{ID: "non-existent-artist-1", Name: "Non Existent Artist 1"}},
						{Artist: model.Artist{ID: "non-existent-artist-2", Name: "Non Existent Artist 2"}},
					},
					model.RoleComposer: {
						{Artist: model.Artist{ID: "non-existent-composer-1", Name: "Non Existent Composer 1"}},
						{Artist: model.Artist{ID: "non-existent-composer-2", Name: "Non Existent Composer 2"}},
					},
					model.RoleAlbumArtist: {
						{Artist: model.Artist{ID: "non-existent-album-artist-1", Name: "Non Existent Album Artist 1"}},
					},
				},
			}

			// This should not fail with foreign key constraint error
			// The updateParticipants method should handle non-existent artist IDs gracefully
			err := albumRepo.Put(album)
			Expect(err).ToNot(HaveOccurred())

			// Verify that no participant records were inserted since all artist IDs were invalid
			// The INNER JOIN with the artist table should filter out all non-existent artists
			verifyAlbumArtists(album.ID, []albumArtistRecord{})

			// Clean up the test album created for this test
			_, _ = albumRepo.executeSQL(squirrel.Delete("album").Where(squirrel.Eq{"id": album.ID}))
		})
	})
})

func _p(id, name string, sortName ...string) model.Participant {
	p := model.Participant{Artist: model.Artist{ID: id, Name: name}}
	if len(sortName) > 0 {
		p.Artist.SortArtistName = sortName[0]
	}
	return p
}
