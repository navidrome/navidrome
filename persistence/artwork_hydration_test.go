package persistence

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

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
})
