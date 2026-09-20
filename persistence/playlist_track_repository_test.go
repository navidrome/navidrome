package persistence

import (
	"context"
	"strconv"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// sqliteMaxVariables is SQLITE_MAX_VARIABLE_NUMBER as compiled into the driver
const sqliteMaxVariables = 32766

var _ = Describe("PlaylistTrackRepository", func() {
	var repo model.PlaylistTrackRepository

	BeforeEach(func() {
		ctx := log.NewContext(GinkgoT().Context())
		ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
		repo = NewPlaylistRepository(ctx, GetDBXBuilder()).Tracks(plsBest.ID, true)
	})

	Describe("GetCursor", func() {
		It("yields the same tracks as GetAll", func() {
			opts := model.QueryOptions{Sort: "id"}
			want, err := repo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(want).To(HaveLen(2))

			Expect(collectCursor(repo.GetCursor(opts))).To(Equal([]model.PlaylistTrack(want)))
		})

		It("honors Max and Offset", func() {
			opts := model.QueryOptions{Sort: "id", Max: 1, Offset: 1}
			want, err := repo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(want).To(HaveLen(1))

			Expect(collectCursor(repo.GetCursor(opts))).To(Equal([]model.PlaylistTrack(want)))
		})
	})

	Describe("GetAll", func() {
		It("returns every row under a random sort, despite the integer id", func() {
			// playlist_tracks.id is an INTEGER, so SEEDEDRAND drops every row unless it is cast to
			// TEXT, and it fails silently: no error, just no rows.
			all, err := repo.GetAll(model.QueryOptions{Sort: "random"})
			Expect(err).ToNot(HaveOccurred())
			Expect(all).To(HaveLen(2), "a random sort must not silently drop rows")

			got, err := repo.GetAll(model.QueryOptions{Sort: "random", Max: 1})
			Expect(err).ToNot(HaveOccurred())
			Expect(got).To(HaveLen(1))
		})
	})

	Describe("CountAll", func() {
		It("returns the number of tracks in the playlist", func() {
			Expect(repo.CountAll()).To(Equal(int64(2)))
		})

		It("ignores Max and Offset", func() {
			Expect(repo.CountAll(model.QueryOptions{Max: 1, Offset: 1})).To(Equal(int64(2)))
		})
	})

	Describe("GetMediaFileIDs", func() {
		It("returns the song ids in playlist order", func() {
			Expect(repo.GetMediaFileIDs(model.QueryOptions{Sort: "id"})).
				To(Equal([]string{songDayInALife.ID, songRadioactivity.ID}))
		})

		It("honors Max and Offset", func() {
			Expect(repo.GetMediaFileIDs(model.QueryOptions{Sort: "id", Max: 1, Offset: 1})).
				To(Equal([]string{songRadioactivity.ID}))
		})
	})

	Describe("Insert", func() {
		var tracks model.PlaylistTrackRepository

		BeforeEach(func() {
			ctx := log.NewContext(GinkgoT().Context())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			plsRepo := NewPlaylistRepository(ctx, GetDBXBuilder())
			pls := model.Playlist{Name: "Insert", OwnerID: "userid", OwnerName: "userid"}
			Expect(plsRepo.Put(&pls)).To(Succeed())
			DeferCleanup(func() { Expect(plsRepo.Delete(pls.ID)).To(Succeed()) })

			tracks = plsRepo.Tracks(pls.ID, false)
			Expect(tracks.Add([]string{songDayInALife.ID, songRadioactivity.ID})).To(Equal(2))
		})

		order := func() []string {
			ids, err := tracks.GetMediaFileIDs(model.QueryOptions{Sort: "id"})
			Expect(err).ToNot(HaveOccurred())
			return ids
		}

		DescribeTable("inserts before a 1-based position, keeping the new tracks' order",
			func(pos int, want func() []string) {
				Expect(tracks.Insert([]string{songComeTogether.ID, songAntenna.ID}, pos)).To(Equal(2))
				Expect(order()).To(Equal(want()))
				Expect(tracks.CountAll()).To(Equal(int64(4)))
			},
			Entry("in the middle", 2, func() []string {
				return []string{songDayInALife.ID, songComeTogether.ID, songAntenna.ID, songRadioactivity.ID}
			}),
			Entry("at the start, for zero or less", 0, func() []string {
				return []string{songComeTogether.ID, songAntenna.ID, songDayInALife.ID, songRadioactivity.ID}
			}),
			Entry("at the end, past the last position", 9, func() []string {
				return []string{songDayInALife.ID, songRadioactivity.ID, songComeTogether.ID, songAntenna.ID}
			}),
		)

		It("renumbers positions contiguously", func() {
			Expect(tracks.Insert([]string{songComeTogether.ID}, 1)).To(Equal(1))
			all, err := tracks.GetAll(model.QueryOptions{Sort: "id"})
			Expect(err).ToNot(HaveOccurred())
			Expect([]string{all[0].ID, all[1].ID, all[2].ID}).To(Equal([]string{"1", "2", "3"}))
		})
	})

	Describe("Reorder", func() {
		var tracks model.PlaylistTrackRepository

		BeforeEach(func() {
			ctx := log.NewContext(GinkgoT().Context())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			plsRepo := NewPlaylistRepository(ctx, GetDBXBuilder())
			pls := model.Playlist{Name: "Reorder", OwnerID: "userid", OwnerName: "userid"}
			Expect(plsRepo.Put(&pls)).To(Succeed())
			DeferCleanup(func() { Expect(plsRepo.Delete(pls.ID)).To(Succeed()) })

			tracks = plsRepo.Tracks(pls.ID, false)
			Expect(tracks.Add([]string{songDayInALife.ID, songRadioactivity.ID, songComeTogether.ID})).To(Equal(3))
		})

		rows := func() ([]string, []string) {
			all, err := tracks.GetAll(model.QueryOptions{Sort: "id"})
			Expect(err).ToNot(HaveOccurred())
			var ids, songs []string
			for _, t := range all {
				ids = append(ids, t.ID)
				songs = append(songs, t.MediaFileID)
			}
			return ids, songs
		}

		DescribeTable("clamps the destination to the playlist",
			func(newPos int, want func() []string) {
				Expect(tracks.Reorder(1, newPos)).To(Succeed())
				ids, songs := rows()
				Expect(ids).To(Equal([]string{"1", "2", "3"}))
				Expect(songs).To(Equal(want()))
			},
			Entry("past the end moves to the end", 9, func() []string {
				return []string{songRadioactivity.ID, songComeTogether.ID, songDayInALife.ID}
			}),
			Entry("below 1 stays first", -4, func() []string {
				return []string{songDayInALife.ID, songRadioactivity.ID, songComeTogether.ID}
			}),
		)

		DescribeTable("rejects a source position outside the playlist, leaving rows untouched",
			func(pos int) {
				Expect(tracks.Reorder(pos, 1)).To(MatchError(model.ErrNotFound))
				ids, songs := rows()
				Expect(ids).To(Equal([]string{"1", "2", "3"}))
				Expect(songs).To(Equal([]string{songDayInALife.ID, songRadioactivity.ID, songComeTogether.ID}))
			},
			Entry("past the end", 4),
			Entry("zero", 0),
		)
	})

	Describe("Delete", func() {
		var tracks model.PlaylistTrackRepository
		const numTracks = deleteChunkSize*2 + 1

		positionsUpTo := func(n int) []string {
			positions := make([]string, 0, n)
			for i := 1; i <= n; i++ {
				positions = append(positions, strconv.Itoa(i))
			}
			return positions
		}

		BeforeEach(func() {
			ctx := log.NewContext(GinkgoT().Context())
			ctx = request.WithUser(ctx, model.User{ID: "userid", UserName: "userid", IsAdmin: true})
			plsRepo := NewPlaylistRepository(ctx, GetDBXBuilder())

			pls := model.Playlist{Name: "Chunked Delete", OwnerID: "userid", OwnerName: "userid"}
			Expect(plsRepo.Put(&pls)).To(Succeed())
			DeferCleanup(func() { Expect(plsRepo.Delete(pls.ID)).To(Succeed()) })

			tracks = plsRepo.Tracks(pls.ID, false)
			songIds := make([]string, numTracks)
			for i := range songIds {
				songIds[i] = songDayInALife.ID
			}
			Expect(tracks.Add(songIds)).To(Equal(numTracks))
		})

		It("removes positions spanning several chunks, and renumbers what is left", func() {
			Expect(tracks.Delete(positionsUpTo(numTracks - 1)...)).To(Succeed())

			Expect(tracks.CountAll()).To(Equal(int64(1)))
			remaining, err := tracks.GetAll(model.QueryOptions{Sort: "id"})
			Expect(err).ToNot(HaveOccurred())
			Expect(remaining[0].ID).To(Equal("1"), "the surviving track must be renumbered to position 1")
		})

		It("accepts more ids than SQLite allows as bind variables", func() {
			Expect(tracks.Delete(positionsUpTo(sqliteMaxVariables + 100)...)).To(Succeed())

			Expect(tracks.CountAll()).To(BeZero())
		})
	})

	Describe("library access", func() {
		var otherLib model.Library
		var restrictedUser model.User
		var adminCtx, userCtx context.Context
		var userTracks model.PlaylistTrackRepository
		var plsID string

		BeforeEach(func() {
			adminCtx, otherLib, restrictedUser = restrictedFixture("pls")
			userCtx = request.WithUser(log.NewContext(GinkgoT().Context()), restrictedUser)
			db := GetDBXBuilder()

			adminMr := NewMediaFileRepository(adminCtx, db)
			Expect(adminMr.Put(&model.MediaFile{
				ID: "pls-otherlib-track", LibraryID: otherLib.ID, AlbumID: "pls-hidden-album",
				Path: "hidden/in-playlist.mp3", Title: "Hidden In Playlist",
			})).To(Succeed())
			DeferCleanup(func() { _ = adminMr.Delete("pls-otherlib-track") })

			adminPls := NewPlaylistRepository(adminCtx, db)
			pls := model.Playlist{Name: "Public Mixed", OwnerID: adminUser.ID, OwnerName: adminUser.UserName, Public: true}
			Expect(adminPls.Put(&pls)).To(Succeed())
			plsID = pls.ID
			DeferCleanup(func() { _ = adminPls.Delete(plsID) })
			Expect(adminPls.Tracks(plsID, false).Add([]string{songDayInALife.ID, "pls-otherlib-track"})).To(Equal(2))

			userTracks = NewPlaylistRepository(userCtx, db).Tracks(plsID, false)
		})

		It("Read does not return a track outside the user's libraries", func() {
			_, err := userTracks.Read("2")
			Expect(err).To(MatchError(model.ErrNotFound), "position 2 holds a track the user cannot access")
		})

		It("Read still returns a track inside the user's libraries", func() {
			trk, err := userTracks.Read("1")
			Expect(err).ToNot(HaveOccurred())
			Expect(trk.(*model.PlaylistTrack).MediaFile.ID).To(Equal(songDayInALife.ID))
		})

		It("Count excludes tracks outside the user's libraries", func() {
			Expect(userTracks.Count()).To(Equal(int64(1)), "Count must agree with the filtered listing")
		})

		It("GetAlbumIDs excludes albums outside the user's libraries", func() {
			Expect(userTracks.GetAlbumIDs()).ToNot(ContainElement("pls-hidden-album"))
		})

		Describe("Add", func() {
			var ownTracks model.PlaylistTrackRepository

			BeforeEach(func() {
				userPls := NewPlaylistRepository(userCtx, GetDBXBuilder())
				own := model.Playlist{Name: "Own Playlist", OwnerID: restrictedUser.ID, OwnerName: restrictedUser.UserName}
				Expect(userPls.Put(&own)).To(Succeed())
				DeferCleanup(func() { _ = NewPlaylistRepository(adminCtx, GetDBXBuilder()).Delete(own.ID) })
				ownTracks = userPls.Tracks(own.ID, false)
			})

			It("drops ids outside the user's libraries", func() {
				Expect(ownTracks.Add([]string{songDayInALife.ID, "pls-otherlib-track"})).To(Equal(1))
				Expect(ownTracks.GetMediaFileIDs()).To(ConsistOf(songDayInALife.ID))
			})

			It("drops them when reached through AddAlbums", func() {
				Expect(ownTracks.AddAlbums([]string{"pls-hidden-album"})).To(BeZero())
			})

			It("drops them when reached through Insert", func() {
				Expect(ownTracks.Add([]string{songDayInALife.ID})).To(Equal(1))

				Expect(ownTracks.Insert([]string{"pls-otherlib-track", songComeTogether.ID}, 1)).To(Equal(1))

				Expect(ownTracks.GetMediaFileIDs()).To(Equal([]string{songComeTogether.ID, songDayInALife.ID}))
				trks, err := ownTracks.GetAll(model.QueryOptions{Sort: "id"})
				Expect(err).ToNot(HaveOccurred())
				Expect(slice.Map(trks, func(t model.PlaylistTrack) string { return t.ID })).To(Equal([]string{"1", "2"}),
					"positions must stay contiguous when an id is dropped")
			})
		})

		Describe("Put", func() {
			storedIDs := func(id string) []string {
				ids, err := NewPlaylistRepository(adminCtx, GetDBXBuilder()).Tracks(id, false).GetMediaFileIDs()
				Expect(err).ToNot(HaveOccurred())
				return ids
			}
			put := func(ctx context.Context, owner model.User, pls *model.Playlist, ids ...string) string {
				pls.OwnerID = owner.ID
				pls.Tracks = nil
				pls.AddMediaFilesByID(ids)
				Expect(NewPlaylistRepository(ctx, GetDBXBuilder()).Put(pls)).To(Succeed())
				DeferCleanup(func() { _ = NewPlaylistRepository(adminCtx, GetDBXBuilder()).Delete(pls.ID) })
				return pls.ID
			}

			It("drops ids outside the user's libraries when creating a playlist", func() {
				id := put(userCtx, restrictedUser, &model.Playlist{Name: "Created"}, songDayInALife.ID, "pls-otherlib-track")

				Expect(storedIDs(id)).To(Equal([]string{songDayInALife.ID}))
			})

			It("drops them when replacing the tracks of an existing playlist", func() {
				pls := &model.Playlist{Name: "Replaced"}
				put(userCtx, restrictedUser, pls, songDayInALife.ID)

				put(userCtx, restrictedUser, pls, "pls-otherlib-track")

				Expect(storedIDs(pls.ID)).To(BeEmpty())
			})

			It("does not count a dropped id, so it cannot be told apart from an unknown one", func() {
				hidden := put(userCtx, restrictedUser, &model.Playlist{Name: "Hidden"}, songDayInALife.ID, "pls-otherlib-track")
				unknown := put(userCtx, restrictedUser, &model.Playlist{Name: "Unknown"}, songDayInALife.ID, "no-such-track")

				userPls := NewPlaylistRepository(userCtx, GetDBXBuilder())
				h, err := userPls.Get(hidden)
				Expect(err).ToNot(HaveOccurred())
				u, err := userPls.Get(unknown)
				Expect(err).ToNot(HaveOccurred())
				Expect(h.SongCount).To(Equal(u.SongCount))
				Expect(h.Duration).To(Equal(u.Duration))
				Expect(h.Size).To(Equal(u.Size))
			})

			It("keeps order and duplicates of the accessible ids", func() {
				id := put(userCtx, restrictedUser, &model.Playlist{Name: "Ordered"},
					songDayInALife.ID, "pls-otherlib-track", songComeTogether.ID, songDayInALife.ID)

				Expect(storedIDs(id)).To(Equal([]string{songDayInALife.ID, songComeTogether.ID, songDayInALife.ID}))
			})

			It("keeps every id when run as an admin, as the scanner's playlist sync does", func() {
				id := put(adminCtx, adminUser, &model.Playlist{Name: "Synced"}, songDayInALife.ID, "pls-otherlib-track")

				Expect(storedIDs(id)).To(Equal([]string{songDayInALife.ID, "pls-otherlib-track"}))
			})
		})

		It("still shows everything to an admin", func() {
			adminTracks := NewPlaylistRepository(adminCtx, GetDBXBuilder()).Tracks(plsID, false)
			Expect(adminTracks.Count()).To(Equal(int64(2)))
			_, err := adminTracks.Read("2")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
