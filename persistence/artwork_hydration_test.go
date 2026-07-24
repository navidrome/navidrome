package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

// seedAnnotations gives each id a distinct rating, play count, play date and starred_at for the
// current user, dropping the rows again when the spec ends. It writes them directly, so that
// SetRating's average_rating update does not outlive the cleanup.
func seedAnnotations(itemType string, ids ...string) {
	GinkgoHelper()
	when := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	for i, id := range ids {
		_, err := GetDBXBuilder().NewQuery(`insert or replace into annotation
			(user_id, item_id, item_type, play_count, play_date, rating, rated_at, starred, starred_at)
			values ({:u}, {:id}, {:t}, {:pc}, {:d}, {:r}, {:d}, 1, {:d})`).
			Bind(dbx.Params{"u": adminUser.ID, "id": id, "t": itemType,
				"pc": i + 1, "r": 4 + i, "d": when.Add(time.Duration(i) * time.Hour)}).Execute()
		Expect(err).ToNot(HaveOccurred())
	}
	DeferCleanup(func() {
		for _, id := range ids {
			_, err := GetDBXBuilder().NewQuery("delete from annotation where user_id={:u} and item_type={:t} and item_id={:id}").
				Bind(dbx.Params{"u": adminUser.ID, "t": itemType, "id": id}).Execute()
			Expect(err).ToNot(HaveOccurred())
		}
	})
}

var _ = Describe("Artwork hydration", func() {
	var ctx context.Context
	var aw model.ArtworkRepository

	putInfo := func(kind, id, hash string) {
		Expect(aw.PutItemArtwork(&model.ItemArtwork{
			ItemKind: kind, ItemID: id, ImageType: model.ImageTypePrimary, Hash: hash,
		})).To(Succeed())
	}

	BeforeEach(func() {
		clearArtworkTables()
		DeferCleanup(clearArtworkTables)
		ctx = request.WithUser(log.NewContext(context.Background()), adminUser)
		aw = NewArtworkRepository(ctx, GetDBXBuilder())
	})

	Describe("albums", func() {
		var repo model.AlbumRepository
		BeforeEach(func() { repo = NewAlbumRepository(ctx, GetDBXBuilder()) })

		It("hydrates the found / known-absent / unresolved states", func() {
			putInfo("al", albumSgtPeppers.ID, "althash11111111")
			putInfo("al", albumAbbeyRoad.ID, "")
			// albumRadioactivity: no row -> unresolved

			byID := map[string]model.Album{}
			all, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			for _, a := range all {
				byID[a.ID] = a
			}

			Expect(byID[albumSgtPeppers.ID].ImageHash).To(Equal("althash11111111"))
			Expect(byID[albumSgtPeppers.ID].ImageAbsent).To(BeFalse())
			Expect(byID[albumAbbeyRoad.ID].ImageHash).To(BeEmpty())
			Expect(byID[albumAbbeyRoad.ID].ImageAbsent).To(BeTrue())
			Expect(byID[albumRadioactivity.ID].ImageHash).To(BeEmpty())
			Expect(byID[albumRadioactivity.ID].ImageAbsent).To(BeFalse())
		})

		It("hydrates Get", func() {
			putInfo("al", albumSgtPeppers.ID, "gethash22222222")
			got, err := repo.Get(albumSgtPeppers.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(got.ImageHash).To(Equal("gethash22222222"))
		})

		It("hydrates Search", func() {
			putInfo("al", albumSgtPeppers.ID, "srchash33333333")
			res, err := repo.Search("Peppers", model.QueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeEmpty())
			Expect(res[0].ImageHash).To(Equal("srchash33333333"))
		})

		It("does not persist ImageHash/ImageAbsent on Put", func() {
			al := albumSgtPeppers
			al.ImageHash = "shouldnotpersist"
			al.ImageAbsent = true
			Expect(repo.(*albumRepository).Put(&al)).To(Succeed())

			// No item_artwork rows exist, so a fresh read must observe zero values.
			got, err := repo.Get(al.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(got.ImageHash).To(BeEmpty())
			Expect(got.ImageAbsent).To(BeFalse())
		})
	})

	Describe("artists", func() {
		var repo model.ArtistRepository
		BeforeEach(func() { repo = NewArtistRepository(ctx, GetDBXBuilder()) })

		It("hydrates the found / known-absent / unresolved states", func() {
			putInfo("ar", artistBeatles.ID, "arhash444444444")
			putInfo("ar", artistKraftwerk.ID, "")
			// artistCJK: no row -> unresolved

			byID := map[string]model.Artist{}
			all, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			for _, a := range all {
				byID[a.ID] = a
			}

			Expect(byID[artistBeatles.ID].ImageHash).To(Equal("arhash444444444"))
			Expect(byID[artistBeatles.ID].ImageAbsent).To(BeFalse())
			Expect(byID[artistKraftwerk.ID].ImageHash).To(BeEmpty())
			Expect(byID[artistKraftwerk.ID].ImageAbsent).To(BeTrue())
			Expect(byID[artistCJK.ID].ImageHash).To(BeEmpty())
			Expect(byID[artistCJK.ID].ImageAbsent).To(BeFalse())
		})

		It("hydrates Get", func() {
			putInfo("ar", artistBeatles.ID, "arget5555555555")
			got, err := repo.Get(artistBeatles.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(got.ImageHash).To(Equal("arget5555555555"))
		})

		It("hydrates Search", func() {
			putInfo("ar", artistBeatles.ID, "arsrch666666666")
			res, err := repo.Search("Beatles", model.QueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeEmpty())
			Expect(res[0].ImageHash).To(Equal("arsrch666666666"))
		})
	})

	Describe("playlists", func() {
		var repo model.PlaylistRepository
		BeforeEach(func() { repo = NewPlaylistRepository(ctx, GetDBXBuilder()) })

		It("hydrates the found / known-absent states", func() {
			putInfo("pl", plsBest.ID, "plhash777777777")
			putInfo("pl", plsCool.ID, "")

			byID := map[string]model.Playlist{}
			all, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			for _, p := range all {
				byID[p.ID] = p
			}

			Expect(byID[plsBest.ID].ImageHash).To(Equal("plhash777777777"))
			Expect(byID[plsBest.ID].ImageAbsent).To(BeFalse())
			Expect(byID[plsCool.ID].ImageHash).To(BeEmpty())
			Expect(byID[plsCool.ID].ImageAbsent).To(BeTrue())
		})

		It("hydrates Get", func() {
			putInfo("pl", plsBest.ID, "plget8888888888")
			got, err := repo.Get(plsBest.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(got.ImageHash).To(Equal("plget8888888888"))
		})
	})

	Describe("radios", func() {
		var repo model.RadioRepository
		BeforeEach(func() { repo = NewRadioRepository(ctx, GetDBXBuilder()) })

		It("hydrates the found / known-absent states", func() {
			putInfo("ra", radioWithHomePage.ID, "rahash999999999")
			putInfo("ra", radioWithoutHomePage.ID, "")

			byID := map[string]model.Radio{}
			all, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			for _, rd := range all {
				byID[rd.ID] = rd
			}

			Expect(byID[radioWithHomePage.ID].ImageHash).To(Equal("rahash999999999"))
			Expect(byID[radioWithHomePage.ID].ImageAbsent).To(BeFalse())
			Expect(byID[radioWithoutHomePage.ID].ImageHash).To(BeEmpty())
			Expect(byID[radioWithoutHomePage.ID].ImageAbsent).To(BeTrue())
		})

		It("hydrates Get", func() {
			putInfo("ra", radioWithHomePage.ID, "ragetaaaaaaaaaa")
			got, err := repo.Get(radioWithHomePage.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(got.ImageHash).To(Equal("ragetaaaaaaaaaa"))
		})
	})

	Describe("mediafiles", func() {
		var repo model.MediaFileRepository

		setCover := func(id string, v bool) {
			_, err := GetDBXBuilder().NewQuery("UPDATE media_file SET has_cover_art={:v} WHERE id={:id}").
				Bind(dbx.Params{"v": v, "id": id}).Execute()
			Expect(err).ToNot(HaveOccurred())
		}

		getByID := func() map[string]model.MediaFile {
			byID := map[string]model.MediaFile{}
			all, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			for _, mf := range all {
				byID[mf.ID] = mf
			}
			return byID
		}

		BeforeEach(func() {
			repo = NewMediaFileRepository(ctx, GetDBXBuilder())
			DeferCleanup(configtest.SetupConfig())
			conf.Server.EnableMediaFileCoverArt = true
		})

		It("resolves the embedded-eligible fallback matrix", func() {
			setCover("1001", true) // eligible, own hash
			setCover("1002", true) // eligible, but embedded art absent -> album
			DeferCleanup(func() { setCover("1001", false); setCover("1002", false) })

			putInfo("al", "101", "alh101xxxxxxxxxx") // song 1001's album (found)
			putInfo("al", "102", "alh102xxxxxxxxxx") // song 1002's album (found)
			putInfo("al", "103", "")                 // songs 1003/1004 album known-absent
			putInfo("mf", "1001", "mfh1001xxxxxxxx")
			putInfo("mf", "1002", "") // embedded resolved absent

			byID := getByID()

			// eligible + own hash -> own hash
			Expect(byID["1001"].ImageHash).To(Equal("mfh1001xxxxxxxx"))
			Expect(byID["1001"].ImageAbsent).To(BeFalse())
			// eligible + embedded absent -> falls through to album 102 info
			Expect(byID["1002"].ImageHash).To(Equal("alh102xxxxxxxxxx"))
			Expect(byID["1002"].ImageAbsent).To(BeFalse())
			// not eligible (no embedded cover) -> album 103 info (known-absent)
			Expect(byID["1003"].ImageHash).To(BeEmpty())
			Expect(byID["1003"].ImageAbsent).To(BeTrue())
			// not eligible, album has no row -> zero values (unresolved)
			Expect(byID["2002"].ImageHash).To(BeEmpty())
			Expect(byID["2002"].ImageAbsent).To(BeFalse())
		})

		It("populates AlbumImage from hydrateArtwork regardless of which continue branch a track takes", func() {
			setCover("1001", true) // eligible, resolves its own art -> own-art-wins continue
			DeferCleanup(func() { setCover("1001", false) })

			putInfo("al", "101", "alh101albimgxxxx") // 1001's album: own-art-wins branch
			putInfo("al", "102", "alh102albimgxxxx") // 1002's album: single-disc inheritance branch
			putInfo("al", "104", "")                 // 2002's album: known-absent, multi-disc branch
			putInfo("mf", "1001", "mfh1001albimgxxx")

			byID := getByID()

			// own-art-wins: ImageHash is the track's own, but AlbumImage still carries the album's.
			Expect(byID["1001"].ImageHash).To(Equal("mfh1001albimgxxx"))
			Expect(byID["1001"].AlbumImage.ImageHash).To(Equal("alh101albimgxxxx"))

			// single-disc album-inheritance: not eligible, so ImageHash mirrors the album's hash.
			Expect(byID["1002"].ImageHash).To(Equal("alh102albimgxxxx"))
			Expect(byID["1002"].AlbumImage.ImageHash).To(Equal("alh102albimgxxxx"))

			// multi-disc bailout: ImageHash stays bare, but AlbumImage still reflects the absence.
			Expect(byID["2002"].ImageHash).To(BeEmpty())
			Expect(byID["2002"].ImageAbsent).To(BeFalse())
			Expect(byID["2002"].AlbumImage.ImageAbsent).To(BeTrue())
		})

		It("carries the blurhash alongside the hash in both the own-art and inherited branches", func() {
			setCover("1001", true) // eligible, resolves its own art -> own-art-wins branch
			DeferCleanup(func() { setCover("1001", false) })

			Expect(aw.PutImage(&model.Artwork{Hash: "mfh1001blurxxxxx", Mime: "image/jpeg", BlurHash: "LTRACKblur"})).To(Succeed())
			Expect(aw.PutImage(&model.Artwork{Hash: "alh102blurxxxxxx", Mime: "image/jpeg", BlurHash: "LALBUMblur"})).To(Succeed())
			putInfo("mf", "1001", "mfh1001blurxxxxx")
			putInfo("al", "102", "alh102blurxxxxxx") // 1002's album: single-disc inheritance branch

			byID := getByID()

			Expect(byID["1001"].ImageHash).To(Equal("mfh1001blurxxxxx"))
			Expect(byID["1001"].BlurHash).To(Equal("LTRACKblur"))

			Expect(byID["1002"].ImageHash).To(Equal("alh102blurxxxxxx"))
			Expect(byID["1002"].BlurHash).To(Equal("LALBUMblur"))
		})

		It("keeps an eligible file optimistic when its own art is unresolved, even if the album is absent", func() {
			// 1004 is eligible (has embedded cover) with no mf state row yet; its album (103) is absent.
			setCover("1004", true)
			DeferCleanup(func() { setCover("1004", false) })
			putInfo("al", "103", "") // album known-absent

			byID := getByID()

			// The track's own embedded art is still unresolved, so it must NOT inherit the album's
			// absence: coverArt stays requestable so serving can extract the embedded art.
			Expect(byID["1004"].ImageAbsent).To(BeFalse())
			Expect(byID["1004"].ImageHash).To(BeEmpty())
			// A non-eligible sibling on the same absent album still inherits the absence.
			Expect(byID["1003"].ImageAbsent).To(BeTrue())
		})

		It("does not stamp a found album hash onto a multi-disc track (its dc- id is disc-served)", func() {
			putInfo("al", "104", "alh104foundxxxxx") // songs 2002/2004 album is found

			byID := getByID()

			// 2002 is multi-disc (DiscNumber>0); CoverArtID emits a dc- id served from disc art of
			// unknown identity, so it must not advertise the album's hash as its content-version.
			Expect(byID["2002"].ImageHash).To(BeEmpty())
			Expect(byID["2002"].ImageAbsent).To(BeFalse())
		})

		It("keeps a multi-disc track requestable when its album is absent (disc art may resolve)", func() {
			putInfo("al", "104", "") // songs 2002/2004 album known-absent

			byID := getByID()

			// 2002 is a multi-disc track (DiscNumber>0); CoverArtID points at disc art, which
			// resolves provisionally, so it must never inherit the album's absence.
			Expect(byID["2002"].ImageAbsent).To(BeFalse())
			Expect(byID["2002"].ImageHash).To(BeEmpty())
		})

		It("uses the album hash for an eligible file whose own art is unresolved but album is found", func() {
			setCover("1004", true)
			DeferCleanup(func() { setCover("1004", false) })
			putInfo("al", "103", "alh103found11111")

			byID := getByID()
			Expect(byID["1004"].ImageAbsent).To(BeFalse())
			Expect(byID["1004"].ImageHash).To(Equal("alh103found11111"))
		})

		It("uses album info for an eligible file when EnableMediaFileCoverArt is off", func() {
			conf.Server.EnableMediaFileCoverArt = false
			setCover("1001", true)
			DeferCleanup(func() { setCover("1001", false) })

			putInfo("al", "101", "alh101offxxxxxxx")
			putInfo("mf", "1001", "mfh1001offxxxxx")

			byID := getByID()
			Expect(byID["1001"].ImageHash).To(Equal("alh101offxxxxxxx"))
			Expect(byID["1001"].ImageAbsent).To(BeFalse())
		})

		It("hydrates Search", func() {
			putInfo("al", "101", "alsrchhhhhhhhhhh")
			res, err := repo.Search("A Day In A Life", model.QueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(res).ToNot(BeEmpty())
			Expect(res[0].ImageHash).To(Equal("alsrchhhhhhhhhhh"))
		})
	})

	Describe("GetCursor", func() {
		var albumRepo model.AlbumRepository
		var artistRepo model.ArtistRepository
		var playlistRepo model.PlaylistRepository
		var onlyAlbums, onlyArtists, onlyPlaylists squirrel.Eq

		scoped := func(opts model.QueryOptions, only squirrel.Eq) model.QueryOptions {
			if opts.Filters == nil {
				opts.Filters = only
			} else {
				opts.Filters = squirrel.And{only, opts.Filters}
			}
			return opts
		}

		BeforeEach(func() {
			albumRepo = NewAlbumRepository(ctx, GetDBXBuilder())
			artistRepo = NewArtistRepository(ctx, GetDBXBuilder())
			playlistRepo = NewPlaylistRepository(ctx, GetDBXBuilder())
			// Other specs leave rows behind, so scope every cursor spec to the fixtures.
			onlyAlbums = squirrel.Eq{"album.id": []string{albumSgtPeppers.ID, albumAbbeyRoad.ID,
				albumRadioactivity.ID, albumMultiDisc.ID, albumCJK.ID, albumPunctuation.ID}}
			onlyArtists = squirrel.Eq{"artist.id": []string{artistKraftwerk.ID, artistBeatles.ID,
				artistCJK.ID, artistPunctuation.ID}}
			// Both fixture playlists belong to the same owner, so the owner_name sort would have a
			// single value to order by. This one is also private and owned by a third user, which
			// is what the non-admin visibility spec needs to see hidden.
			foreign := model.Playlist{Name: "Foreign", OwnerID: thirdUser.ID, OwnerName: thirdUser.UserName}
			Expect(playlistRepo.Put(&foreign)).To(Succeed())
			DeferCleanup(func() { Expect(playlistRepo.Delete(foreign.ID)).To(Succeed()) })
			onlyPlaylists = squirrel.Eq{"playlist.id": []string{plsBest.ID, plsCool.ID, foreign.ID}}

			// The suite annotates a single album and artist, so the annotation-backed sorts would
			// have nothing to order. Seed distinct values, and drop them again after each spec.
			seedAnnotations("album", albumSgtPeppers.ID, albumAbbeyRoad.ID)
			seedAnnotations("artist", artistKraftwerk.ID, artistCJK.ID)

			Expect(aw.PutImage(&model.Artwork{
				Hash: "curhash11111111", Mime: "image/jpeg", BlurHash: "LEHV6nWB2yk8",
			})).To(Succeed())
			putInfo("al", albumSgtPeppers.ID, "curhash11111111")
			putInfo("al", albumAbbeyRoad.ID, "")
			putInfo("ar", artistBeatles.ID, "curhash11111111")
			putInfo("pl", plsBest.ID, "curhash11111111")
		})

		It("hydrates every streamed album, like GetAll", func() {
			opts := model.QueryOptions{Sort: "name", Filters: onlyAlbums}
			want, err := albumRepo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())

			got := collectCursor(albumRepo.GetCursor(opts))

			Expect(got).To(ConsistOf(want))
			byID := map[string]model.Album{}
			for _, al := range got {
				byID[al.ID] = al
			}
			Expect(byID[albumSgtPeppers.ID].ImageHash).To(Equal("curhash11111111"))
			Expect(byID[albumSgtPeppers.ID].BlurHash).To(Equal("LEHV6nWB2yk8"))
			Expect(byID[albumAbbeyRoad.ID].ImageAbsent).To(BeTrue())
			Expect(byID[albumRadioactivity.ID].ImageHash).To(BeEmpty())
			Expect(byID[albumRadioactivity.ID].ImageAbsent).To(BeFalse())
		})

		It("hydrates every streamed artist, like GetAll", func() {
			opts := model.QueryOptions{Sort: "name", Filters: onlyArtists}
			want, err := artistRepo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())

			got := collectCursor(artistRepo.GetCursor(opts))

			Expect(got).To(ConsistOf(want))
			Expect(slice.Map(got, func(a model.Artist) string { return a.ImageHash })).
				To(ContainElement("curhash11111111"))
		})

		It("hydrates every streamed playlist, like GetAll", func() {
			opts := model.QueryOptions{Sort: "name", Filters: onlyPlaylists}
			want, err := playlistRepo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())

			got := collectCursor(playlistRepo.GetCursor(opts))

			Expect(got).To(ConsistOf(want))
			Expect(slice.Map(got, func(p model.Playlist) string { return p.ImageHash })).
				To(ContainElement("curhash11111111"))
		})

		It("honors Max and Offset exactly once", func() {
			opts := model.QueryOptions{Sort: "name", Filters: onlyAlbums, Max: 2, Offset: 1}
			want, err := albumRepo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(want).To(HaveLen(2))

			got := collectCursor(albumRepo.GetCursor(opts))

			Expect(slice.Map(got, func(a model.Album) string { return a.ID })).
				To(Equal(slice.Map(want, func(a model.Album) string { return a.ID })))
		})

		// The sorts and filters the Jellyfin list endpoints issue: the id pre-pass must resolve each
		// of them exactly like the full query. Comparing sort keys, not ids, keeps ties out of it.
		DescribeTable("orders albums like GetAll",
			func(opts model.QueryOptions, key func(model.Album) string) {
				opts = scoped(opts, onlyAlbums)
				want, err := albumRepo.GetAll(opts)
				Expect(err).ToNot(HaveOccurred())
				Expect(want).ToNot(BeEmpty())

				got := collectCursor(albumRepo.GetCursor(opts))

				Expect(slice.Map(got, key)).To(Equal(slice.Map(want, key)))
				Expect(slice.Map(got, func(a model.Album) string { return a.ID })).
					To(ConsistOf(slice.Map(want, func(a model.Album) string { return a.ID })))
			},
			Entry("by name", model.QueryOptions{Sort: "name"},
				func(a model.Album) string { return a.OrderAlbumName }),
			Entry("by artist", model.QueryOptions{Sort: "artist"},
				func(a model.Album) string { return a.OrderAlbumArtistName }),
			Entry("by recently added", model.QueryOptions{Sort: "recently_added", Order: "desc"},
				func(a model.Album) string { return fmt.Sprint(a.CreatedAt) }),
			Entry("by year", model.QueryOptions{Sort: "max_year"},
				func(a model.Album) string { return fmt.Sprint(a.MaxYear) }),
			Entry("by play count", model.QueryOptions{Sort: "play_count", Order: "desc"},
				func(a model.Album) string { return fmt.Sprint(a.PlayCount) }),
			Entry("by last played", model.QueryOptions{Sort: "play_date", Order: "desc"},
				func(a model.Album) string { return fmt.Sprint(a.PlayDate) }),
			Entry("by rating", model.QueryOptions{Sort: "rating", Order: "desc"},
				func(a model.Album) string { return fmt.Sprint(a.Rating) }),
			Entry("starred only", model.QueryOptions{Sort: "starred_at", Order: "desc",
				Filters: squirrel.Eq{"starred": true}},
				func(a model.Album) string { return fmt.Sprint(a.StarredAt) }),
		)

		DescribeTable("orders artists like GetAll",
			func(opts model.QueryOptions, key func(model.Artist) string) {
				opts = scoped(opts, onlyArtists)
				want, err := artistRepo.GetAll(opts)
				Expect(err).ToNot(HaveOccurred())
				Expect(want).ToNot(BeEmpty())

				got := collectCursor(artistRepo.GetCursor(opts))

				Expect(slice.Map(got, key)).To(Equal(slice.Map(want, key)))
				Expect(slice.Map(got, func(a model.Artist) string { return a.ID })).
					To(ConsistOf(slice.Map(want, func(a model.Artist) string { return a.ID })))
			},
			Entry("by name", model.QueryOptions{Sort: "name"},
				func(a model.Artist) string { return a.OrderArtistName }),
			Entry("by album count", model.QueryOptions{Sort: "album_count", Order: "desc"},
				func(a model.Artist) string { return fmt.Sprint(a.AlbumCount) }),
			Entry("by song count", model.QueryOptions{Sort: "song_count", Order: "desc"},
				func(a model.Artist) string { return fmt.Sprint(a.SongCount) }),
			Entry("by play count", model.QueryOptions{Sort: "play_count", Order: "desc"},
				func(a model.Artist) string { return fmt.Sprint(a.PlayCount) }),
			Entry("starred only", model.QueryOptions{Sort: "starred_at", Order: "desc",
				Filters: squirrel.Eq{"starred": true}},
				func(a model.Artist) string { return fmt.Sprint(a.StarredAt) }),
		)

		DescribeTable("orders playlists like GetAll",
			func(opts model.QueryOptions, key func(model.Playlist) string) {
				opts = scoped(opts, onlyPlaylists)
				want, err := playlistRepo.GetAll(opts)
				Expect(err).ToNot(HaveOccurred())
				Expect(want).ToNot(BeEmpty())

				got := collectCursor(playlistRepo.GetCursor(opts))

				Expect(slice.Map(got, key)).To(Equal(slice.Map(want, key)))
				Expect(slice.Map(got, func(p model.Playlist) string { return p.ID })).
					To(ConsistOf(slice.Map(want, func(p model.Playlist) string { return p.ID })))
			},
			Entry("by name", model.QueryOptions{Sort: "name"},
				func(p model.Playlist) string { return p.Name }),
			Entry("by creation date", model.QueryOptions{Sort: "created_at", Order: "desc"},
				func(p model.Playlist) string { return fmt.Sprint(p.CreatedAt) }),
			Entry("by owner", model.QueryOptions{Sort: "owner_name"},
				func(p model.Playlist) string { return p.OwnerName }),
		)

		It("streams every album exactly once when sorted randomly", func() {
			opts := model.QueryOptions{Sort: "random", Filters: onlyAlbums}
			want, err := albumRepo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())

			got := collectCursor(albumRepo.GetCursor(opts))

			Expect(slice.Map(got, func(a model.Album) string { return a.ID })).
				To(ConsistOf(slice.Map(want, func(a model.Album) string { return a.ID })))
		})

		It("keeps a non-admin from streaming another user's private playlists", func() {
			otherCtx := request.WithUser(log.NewContext(context.Background()), regularUser)
			repo := NewPlaylistRepository(otherCtx, GetDBXBuilder())
			opts := model.QueryOptions{Sort: "name", Filters: onlyPlaylists}

			// Both phases must filter on their own: the id pre-pass, and the chunk fetch that would
			// otherwise re-read an id whose visibility changed in between.
			Expect(repo.GetAllIDs(opts)).To(ConsistOf(plsBest.ID))
			all, err := repo.GetAll(model.QueryOptions{Filters: onlyPlaylists})
			Expect(err).ToNot(HaveOccurred())
			Expect(slice.Map(all, func(p model.Playlist) string { return p.ID })).To(ConsistOf(plsBest.ID))

			got := collectCursor(repo.GetCursor(opts))

			Expect(slice.Map(got, func(p model.Playlist) string { return p.Name })).
				To(ConsistOf(plsBest.Name))
		})
	})

	Describe("GetCursorWithArtwork", func() {
		var mfRepo model.MediaFileRepository
		var onlySongs squirrel.Eq

		BeforeEach(func() {
			mfRepo = NewMediaFileRepository(ctx, GetDBXBuilder())
			putInfo("al", albumSgtPeppers.ID, "curhash11111111")
			// Distinct titles keep the ordering/paging specs tie-free; other fixture songs share
			// titles (e.g. "Antenna") or albums, which would make row order ambiguous.
			onlySongs = squirrel.Eq{"media_file.id": []string{songDayInALife.ID, songComeTogether.ID,
				songRadioactivity.ID, songAntenna.ID, songDisc1Track01.ID, songCJK.ID, songPunctuation.ID}}
		})

		It("hydrates artwork onto every streamed track, unlike GetCursor", func() {
			// Scoped to onlySongs (distinct titles): the full fixture has title ties (e.g. "Antenna"
			// x3), so positional comparison against GetAll would only pass by tie-order coincidence.
			opts := model.QueryOptions{Sort: "title", Filters: onlySongs}
			want, err := mfRepo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(want).ToNot(BeEmpty())

			cursor, err := mfRepo.GetCursorWithArtwork(opts)
			Expect(err).ToNot(HaveOccurred())
			var got model.MediaFiles
			cursor(func(mf model.MediaFile, err error) bool {
				Expect(err).ToNot(HaveOccurred())
				got = append(got, mf)
				return true
			})

			Expect(got).To(HaveLen(len(want)))
			for i := range want {
				Expect(got[i].ID).To(Equal(want[i].ID))
				Expect(got[i].ImageHash).To(Equal(want[i].ImageHash))
				Expect(got[i].ImageAbsent).To(Equal(want[i].ImageAbsent))
				Expect(got[i].AlbumImage.ImageHash).To(Equal(want[i].AlbumImage.ImageHash))
				Expect(got[i].AlbumImage.BlurHash).To(Equal(want[i].AlbumImage.BlurHash))
			}
			Expect(want).To(ContainElement(HaveField("AlbumImage.ImageHash", Not(BeEmpty()))),
				"fixture must include at least one track with album artwork, or this proves nothing")
		})

		It("leaves the scanner's GetCursor unhydrated", func() {
			cursor, err := mfRepo.GetCursor(model.QueryOptions{Sort: "title"})
			Expect(err).ToNot(HaveOccurred())
			var seen int
			cursor(func(mf model.MediaFile, err error) bool {
				Expect(err).ToNot(HaveOccurred())
				Expect(mf.AlbumImage.ImageHash).To(BeEmpty(), "GetCursor must stay unhydrated for the scanner")
				seen++
				return true
			})
			Expect(seen).To(BeNumerically(">", 0))
		})

		It("streams the same ids in the same order as GetAll", func() {
			opts := model.QueryOptions{Sort: "title", Filters: onlySongs}
			want, err := mfRepo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(want).ToNot(BeEmpty())

			got := collectCursor(mfRepo.GetCursorWithArtwork(opts))

			Expect(slice.Map(got, func(mf model.MediaFile) string { return mf.ID })).
				To(Equal(slice.Map(want, func(mf model.MediaFile) string { return mf.ID })))
		})

		It("honors Max and Offset exactly once", func() {
			opts := model.QueryOptions{Sort: "title", Filters: onlySongs, Max: 2, Offset: 1}
			want, err := mfRepo.GetAll(opts)
			Expect(err).ToNot(HaveOccurred())
			Expect(want).To(HaveLen(2))

			got := collectCursor(mfRepo.GetCursorWithArtwork(opts))

			Expect(slice.Map(got, func(mf model.MediaFile) string { return mf.ID })).
				To(Equal(slice.Map(want, func(mf model.MediaFile) string { return mf.ID })))
		})
	})

	Describe("chunkOptions", func() {
		It("carries Sort, Order and the caller's filters, but never Max/Offset", func() {
			base := model.QueryOptions{Sort: "name", Order: "desc", Max: 10, Offset: 20,
				Filters: squirrel.Eq{"missing": false}}

			got := chunkOptions([]model.QueryOptions{base}, "album.id")([]string{"al-1", "al-2"})

			Expect(got.Max).To(BeZero())
			Expect(got.Offset).To(BeZero())
			Expect(got.Sort).To(Equal("name"))
			Expect(got.Order).To(Equal("desc"))
			Expect(got.Filters).To(Equal(squirrel.And{
				base.Filters, squirrel.Eq{"album.id": []string{"al-1", "al-2"}},
			}))
		})

		It("filters by ids alone when the caller passed no filters", func() {
			got := chunkOptions(nil, "artist.id")([]string{"ar-1"})
			Expect(got.Filters).To(Equal(squirrel.Eq{"artist.id": []string{"ar-1"}}))
		})
	})

	Describe("streamByIDs", func() {
		It("fetches in chunks and yields every row in order", func() {
			ids := make([]string, artworkChunkSize+3)
			for i := range ids {
				ids[i] = fmt.Sprintf("id-%d", i)
			}
			var chunks [][]string
			got := collectCursor(streamByIDs(ids, func(chunk []string) ([]string, error) {
				chunks = append(chunks, chunk)
				return chunk, nil
			}), nil)

			Expect(chunks).To(HaveLen(2))
			Expect(chunks[0]).To(HaveLen(artworkChunkSize))
			Expect(chunks[1]).To(HaveLen(3))
			Expect(got).To(Equal(ids))
		})

		It("yields the fetch error and stops", func() {
			ids := make([]string, artworkChunkSize+1)
			calls := 0
			var errs []error
			for _, err := range streamByIDs(ids, func(chunk []string) ([]string, error) {
				calls++
				return nil, errors.New("boom")
			}) {
				errs = append(errs, err)
			}

			Expect(calls).To(Equal(1))
			Expect(errs).To(HaveLen(1))
			Expect(errs[0]).To(MatchError("boom"))
		})

		It("stops fetching when the consumer breaks", func() {
			ids := make([]string, artworkChunkSize+1)
			calls := 0
			for range streamByIDs(ids, func(chunk []string) ([]string, error) {
				calls++
				return chunk, nil
			}) {
				break
			}

			Expect(calls).To(Equal(1))
		})
	})

	Describe("applyItemImage", func() {
		It("copies hash, absence and blurhash onto the item", func() {
			infos := map[string]model.ItemArtworkInfo{
				"al-1": {ItemID: "al-1", Hash: "0123456789abcdef", BlurHash: "LEHV6nWB2yk8"},
			}
			var img model.ItemImage
			applyItemImage(infos, "al-1", &img)
			Expect(img.ImageHash).To(Equal("0123456789abcdef"))
			Expect(img.ImageAbsent).To(BeFalse())
			Expect(img.BlurHash).To(Equal("LEHV6nWB2yk8"))
		})

		It("marks a hashless entry absent and carries no blurhash", func() {
			infos := map[string]model.ItemArtworkInfo{"al-2": {ItemID: "al-2"}}
			var img model.ItemImage
			applyItemImage(infos, "al-2", &img)
			Expect(img.ImageAbsent).To(BeTrue())
			Expect(img.BlurHash).To(BeEmpty())
		})

		It("leaves an unresolved item zero-valued", func() {
			var img model.ItemImage
			applyItemImage(map[string]model.ItemArtworkInfo{}, "al-3", &img)
			Expect(img.ImageHash).To(BeEmpty())
			Expect(img.ImageAbsent).To(BeFalse())
			Expect(img.BlurHash).To(BeEmpty())
		})
	})
})
