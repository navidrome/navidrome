package artworke2e_test

import (
	"testing/fstest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Doc reference:
// https://www.navidrome.org/docs/usage/library/artwork/#artists
// Default ArtistArtPriority is "artist.*, album/artist.*, external".
var _ = Describe("Artist artwork resolution", func() {
	BeforeEach(func() {
		setupHarness()
	})

	When("the artist folder contains an artist.jpg", func() {
		It("returns the artist.* image from the artist folder", func() {
			conf.Server.ArtistArtPriority = "artist.*, album/artist.*, external"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"albumartist": "Artist"}),
				"Artist/artist.jpg":           imageFile("artist-folder"),
			})
			scan()

			ar := soleArtist()
			artID := model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil)
			Expect(readArtwork(artID)).To(Equal(imageBytes("artist-folder")))
		})
	})

	When("artist.* only exists inside an album folder", func() {
		It("falls through to album/artist.* and returns that image", func() {
			conf.Server.ArtistArtPriority = "artist.*, album/artist.*, external"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"albumartist": "Artist"}),
				"Artist/Album/artist.jpg":     imageFile("album-artist"),
			})
			scan()

			ar := soleArtist()
			artID := model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil)
			Expect(readArtwork(artID)).To(Equal(imageBytes("album-artist")))
		})
	})

	When("both the artist folder and an album folder have an artist.* image", func() {
		It("prefers the artist-folder image (artist.* comes before album/artist.*)", func() {
			conf.Server.ArtistArtPriority = "artist.*, album/artist.*, external"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"albumartist": "Artist"}),
				"Artist/artist.jpg":           imageFile("artist-folder"),
				"Artist/Album/artist.jpg":     imageFile("album-artist"),
			})
			scan()

			ar := soleArtist()
			artID := model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil)
			Expect(readArtwork(artID)).To(Equal(imageBytes("artist-folder")))
		})
	})
})

func soleArtist() model.Artist {
	GinkgoHelper()
	artists, err := ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"artist.name": "Artist"},
	})
	Expect(err).ToNot(HaveOccurred())
	if len(artists) == 0 {
		Fail("sole artist not found")
		return model.Artist{}
	}
	return artists[0]
}
