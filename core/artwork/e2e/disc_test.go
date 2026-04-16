package artworke2e_test

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Disc artwork resolution requires tracks with non-zero disc numbers, but the
// MP3 fixtures we copy from `tests/fixtures/` have empty TPOS (disc) tags so
// every track lands on disc 0. Tests below drive the disc reader directly with
// discNumber=1, which still exercises the file-glob branch even though the
// underlying tracks report disc 0.
var _ = Describe("Disc artwork resolution", func() {
	BeforeEach(func() {
		setupHarness()
	})

	When("the album is single-disc with a disc1.jpg in the only folder", func() {
		It("returns the disc1.jpg image (matched as disc*.*)", func() {
			conf.Server.DiscArtPriority = "disc*.*, cd*.*, cover.*, folder.*, front.*, embedded"
			writeLayout(
				mp3Naked("Artist/Album/01 - Track.mp3"),
				imgJPEG("Artist/Album/disc1.jpg", "disc1-image"),
			)
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(jpegLabel("disc1-image")))
		})
	})

	When("the album has no per-disc image and no album cover", func() {
		It("returns ErrUnavailable for the disc lookup", func() {
			conf.Server.DiscArtPriority = "disc*.*, cd*.*"
			conf.Server.CoverArtPriority = "cover.*, folder.*"
			writeLayout(
				mp3Naked("Artist/Album/01 - Track.mp3"),
			)
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			_, err := readArtworkOrErr(discID)
			Expect(err).To(HaveOccurred())
		})
	})

	When("the album has no per-disc image but has an album cover", func() {
		It("falls back to the album cover", func() {
			conf.Server.DiscArtPriority = "disc*.*, cd*.*"
			conf.Server.CoverArtPriority = defaultCoverPriority
			writeLayout(
				mp3Naked("Artist/Album/01 - Track.mp3"),
				imgJPEG("Artist/Album/cover.jpg", "album-cover"),
			)
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(jpegLabel("album-cover")))
		})
	})

	When("multiple disc images exist in the same folder (disc1 vs disc10)", func() {
		It("matches the requested disc number, not a higher-numbered one", func() {
			conf.Server.DiscArtPriority = "disc*.*"
			writeLayout(
				mp3Naked("Artist/Album/01 - Track.mp3"),
				imgJPEG("Artist/Album/disc1.jpg", "disc-one"),
				imgJPEG("Artist/Album/disc10.jpg", "disc-ten"),
			)
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(jpegLabel("disc-one")))
		})
	})
})
