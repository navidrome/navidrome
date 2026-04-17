package artworke2e_test

import (
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Disc artwork resolution", func() {
	BeforeEach(func() {
		setupHarness()
	})

	When("the album is single-disc with a disc1.jpg in the only folder", func() {
		It("returns the disc1.jpg image (matched as disc*.*)", func() {
			conf.Server.DiscArtPriority = "disc*.*, cd*.*, cover.*, folder.*, front.*, embedded"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/disc1.jpg":      imageFile("disc1-image"),
			})
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(imageBytes("disc1-image")))
		})
	})

	When("the album has no per-disc image and no album cover", func() {
		It("returns ErrUnavailable for the disc lookup", func() {
			conf.Server.DiscArtPriority = "disc*.*, cd*.*"
			conf.Server.CoverArtPriority = "cover.*, folder.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
			})
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
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":      imageFile("album-cover"),
			})
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(imageBytes("album-cover")))
		})
	})

	When("multiple disc images exist in the same folder (disc1 vs disc10)", func() {
		It("matches the requested disc number, not a higher-numbered one", func() {
			conf.Server.DiscArtPriority = "disc*.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/disc1.jpg":      imageFile("disc-one"),
				"Artist/Album/disc10.jpg":     imageFile("disc-ten"),
			})
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(imageBytes("disc-one")))
		})
	})

	When("a multi-disc album has per-disc covers", func() {
		It("returns the requested disc's image", func() {
			conf.Server.DiscArtPriority = "disc*.*"
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/CD1/disc1.jpg":      imageFile("disc-1"),
				"Artist/Album/CD2/disc2.jpg":      imageFile("disc-2"),
			})
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 2), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(imageBytes("disc-2")))
		})
	})
})
