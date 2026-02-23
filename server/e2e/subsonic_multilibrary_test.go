package e2e

import (
	"fmt"
	"testing/fstest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/metrics"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/core/storage/storagetest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/scanner"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Multi-Library Support", Ordered, func() {
	var lib2 model.Library
	var adminWithLibs model.User // admin reloaded with both libraries
	var userLib1Only model.User  // non-admin with lib1 access only

	BeforeAll(func() {
		conf.Server.EnableSharing = true
		setupTestDB()

		// Create a second FakeFS with Classical music content
		classical := template(_t{
			"albumartist": "Ludwig van Beethoven",
			"artist":      "Ludwig van Beethoven",
			"album":       "Symphony No. 9",
			"year":        1824,
			"genre":       "Classical",
		})
		classicalFS := storagetest.FakeFS{}
		classicalFS.SetFiles(fstest.MapFS{
			"Classical/Beethoven/Symphony No. 9/01 - Allegro ma non troppo.mp3": classical(track(1, "Allegro ma non troppo")),
			"Classical/Beethoven/Symphony No. 9/02 - Ode to Joy.mp3":            classical(track(2, "Ode to Joy")),
		})
		storagetest.Register("fake2", &classicalFS)

		// Create the second library in the DB (Put auto-assigns admin users)
		lib2 = model.Library{ID: 2, Name: "Classical Library", Path: "fake2:///classical"}
		Expect(ds.Library(ctx).Put(&lib2)).To(Succeed())

		// Reload admin user to get both libraries in the Libraries field
		loadedAdmin, err := ds.User(ctx).FindByUsername(adminUser.UserName)
		Expect(err).ToNot(HaveOccurred())
		adminWithLibs = *loadedAdmin

		// Run incremental scan to import lib2 content (lib1 files unchanged → skipped)
		s := scanner.New(ctx, ds, artwork.NoopCacheWarmer(), events.NoopBroker(),
			playlists.NewPlaylists(ds), metrics.NewNoopInstance())
		_, err = s.ScanAll(ctx, false)
		Expect(err).ToNot(HaveOccurred())

		// Create a non-admin user with access only to lib1
		userLib1Only = model.User{
			ID:          "multilib-user-1",
			UserName:    "lib1user",
			Name:        "Lib1 User",
			IsAdmin:     false,
			NewPassword: "password",
		}
		Expect(ds.User(ctx).Put(&userLib1Only)).To(Succeed())
		Expect(ds.User(ctx).SetUserLibraries(userLib1Only.ID, []int{lib.ID})).To(Succeed())

		loadedUser, err := ds.User(ctx).FindByUsername(userLib1Only.UserName)
		Expect(err).ToNot(HaveOccurred())
		userLib1Only.Libraries = loadedUser.Libraries
	})

	Describe("getMusicFolders", func() {
		It("returns both libraries for admin user", func() {
			resp := doReqWithUser(adminWithLibs, "getMusicFolders")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.MusicFolders.Folders).To(HaveLen(2))

			names := make([]string, len(resp.MusicFolders.Folders))
			for i, f := range resp.MusicFolders.Folders {
				names[i] = f.Name
			}
			Expect(names).To(ConsistOf("Music Library", "Classical Library"))
		})
	})

	Describe("getArtists - library filtering", func() {
		It("returns only lib1 artists when musicFolderId=1", func() {
			resp := doReqWithUser(adminWithLibs, "getArtists", "musicFolderId", fmt.Sprintf("%d", lib.ID))

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Artist).ToNot(BeNil())

			var artistNames []string
			for _, idx := range resp.Artist.Index {
				for _, a := range idx.Artists {
					artistNames = append(artistNames, a.Name)
				}
			}
			Expect(artistNames).To(ContainElements("The Beatles", "Led Zeppelin", "Miles Davis"))
			Expect(artistNames).ToNot(ContainElement("Ludwig van Beethoven"))
		})

		It("returns only lib2 artists when musicFolderId=2", func() {
			resp := doReqWithUser(adminWithLibs, "getArtists", "musicFolderId", fmt.Sprintf("%d", lib2.ID))

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Artist).ToNot(BeNil())

			var artistNames []string
			for _, idx := range resp.Artist.Index {
				for _, a := range idx.Artists {
					artistNames = append(artistNames, a.Name)
				}
			}
			Expect(artistNames).To(ContainElement("Ludwig van Beethoven"))
			Expect(artistNames).ToNot(ContainElements("The Beatles", "Led Zeppelin", "Miles Davis"))
		})

		It("returns artists from all libraries when no musicFolderId is specified", func() {
			resp := doReqWithUser(adminWithLibs, "getArtists")

			Expect(resp.Status).To(Equal(responses.StatusOK))

			var artistNames []string
			for _, idx := range resp.Artist.Index {
				for _, a := range idx.Artists {
					artistNames = append(artistNames, a.Name)
				}
			}
			Expect(artistNames).To(ContainElements("The Beatles", "Led Zeppelin", "Miles Davis", "Ludwig van Beethoven"))
		})
	})

	Describe("getAlbumList - library filtering", func() {
		It("returns only lib1 albums when musicFolderId=1", func() {
			resp := doReqWithUser(adminWithLibs, "getAlbumList", "type", "alphabeticalByName", "musicFolderId", fmt.Sprintf("%d", lib.ID))

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(6))
			for _, a := range resp.AlbumList.Album {
				Expect(a.Title).ToNot(Equal("Symphony No. 9"))
			}
		})

		It("returns only lib2 albums when musicFolderId=2", func() {
			resp := doReqWithUser(adminWithLibs, "getAlbumList", "type", "alphabeticalByName", "musicFolderId", fmt.Sprintf("%d", lib2.ID))

			Expect(resp.AlbumList).ToNot(BeNil())
			Expect(resp.AlbumList.Album).To(HaveLen(1))
			Expect(resp.AlbumList.Album[0].Title).To(Equal("Symphony No. 9"))
		})
	})

	Describe("search3 - library filtering", func() {
		It("does not find lib1 content when searching in lib2 only", func() {
			resp := doReqWithUser(adminWithLibs, "search3", "query", "Beatles", "musicFolderId", fmt.Sprintf("%d", lib2.ID))

			Expect(resp.SearchResult3).ToNot(BeNil())
			Expect(resp.SearchResult3.Artist).To(BeEmpty())
			Expect(resp.SearchResult3.Album).To(BeEmpty())
			Expect(resp.SearchResult3.Song).To(BeEmpty())
		})

		It("finds lib2 content when searching in lib2", func() {
			resp := doReqWithUser(adminWithLibs, "search3", "query", "Beethoven", "musicFolderId", fmt.Sprintf("%d", lib2.ID))

			Expect(resp.SearchResult3).ToNot(BeNil())
			Expect(resp.SearchResult3.Artist).ToNot(BeEmpty())
			Expect(resp.SearchResult3.Artist[0].Name).To(Equal("Ludwig van Beethoven"))
		})
	})

	Describe("Cross-library playlists", Ordered, func() {
		var playlistID string
		var lib1SongID, lib2SongID string

		BeforeAll(func() {
			// Look up one song from each library
			lib1Songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"media_file.library_id": lib.ID},
				Max:     1, Sort: "title",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(lib1Songs).ToNot(BeEmpty())
			lib1SongID = lib1Songs[0].ID

			lib2Songs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"media_file.library_id": lib2.ID},
				Max:     1, Sort: "title",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(lib2Songs).ToNot(BeEmpty())
			lib2SongID = lib2Songs[0].ID
		})

		It("admin creates a playlist with songs from both libraries", func() {
			resp := doReqWithUser(adminWithLibs, "createPlaylist",
				"name", "Cross-Library Playlist", "songId", lib1SongID, "songId", lib2SongID)

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Playlist).ToNot(BeNil())
			Expect(resp.Playlist.SongCount).To(Equal(int32(2)))
			Expect(resp.Playlist.Entry).To(HaveLen(2))
			playlistID = resp.Playlist.Id
		})

		It("admin makes the playlist public", func() {
			resp := doReqWithUser(adminWithLibs, "updatePlaylist",
				"playlistId", playlistID, "public", "true")

			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("non-admin user with lib1 only sees only lib1 tracks in the playlist", func() {
			resp := doReqWithUser(userLib1Only, "getPlaylist", "id", playlistID)

			Expect(resp.Playlist).ToNot(BeNil())
			// The playlist has 2 songs total, but the non-admin user only has access to lib1
			Expect(resp.Playlist.Entry).To(HaveLen(1))
			Expect(resp.Playlist.Entry[0].Id).To(Equal(lib1SongID))
		})
	})

	Describe("Cross-library shares", Ordered, func() {
		var lib2AlbumID string

		BeforeAll(func() {
			lib2Albums, err := ds.Album(ctx).GetAll(model.QueryOptions{
				Filters: squirrel.Eq{"album.library_id": lib2.ID},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(lib2Albums).ToNot(BeEmpty())
			lib2AlbumID = lib2Albums[0].ID
		})

		It("admin creates a share for a lib2 album", func() {
			resp := doReqWithUser(adminWithLibs, "createShare",
				"id", lib2AlbumID, "description", "Classical album share")

			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(resp.Shares).ToNot(BeNil())
			Expect(resp.Shares.Share).To(HaveLen(1))

			share := resp.Shares.Share[0]
			Expect(share.Description).To(Equal("Classical album share"))
			Expect(share.Entry).ToNot(BeEmpty())
			Expect(share.Entry[0].Title).To(Equal("Symphony No. 9"))
		})
	})

	Describe("Library access control", func() {
		It("returns error when non-admin user requests inaccessible library", func() {
			resp := doReqWithUser(userLib1Only, "getArtists", "musicFolderId", fmt.Sprintf("%d", lib2.ID))

			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})

		It("non-admin user sees only their library's content without musicFolderId", func() {
			resp := doReqWithUser(userLib1Only, "getArtists")

			Expect(resp.Status).To(Equal(responses.StatusOK))

			var artistNames []string
			for _, idx := range resp.Artist.Index {
				for _, a := range idx.Artists {
					artistNames = append(artistNames, a.Name)
				}
			}
			Expect(artistNames).To(ContainElements("The Beatles", "Led Zeppelin", "Miles Davis"))
			Expect(artistNames).ToNot(ContainElement("Ludwig van Beethoven"))
		})

		It("non-admin user search returns only their library's content", func() {
			resp := doReqWithUser(userLib1Only, "search3", "query", "Beethoven")

			Expect(resp.SearchResult3).ToNot(BeNil())
			Expect(resp.SearchResult3.Artist).To(BeEmpty(), "userLib1Only should not see Beethoven (lib2)")
			Expect(resp.SearchResult3.Album).To(BeEmpty())
			Expect(resp.SearchResult3.Song).To(BeEmpty())
		})

		It("non-admin user search finds content from their library", func() {
			resp := doReqWithUser(userLib1Only, "search3", "query", "Beatles")

			Expect(resp.SearchResult3).ToNot(BeNil())
			Expect(resp.SearchResult3.Artist).ToNot(BeEmpty(), "userLib1Only should find Beatles (lib1)")
		})
	})
})
