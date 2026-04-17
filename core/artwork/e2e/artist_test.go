package artworke2e_test

import (
	"os"
	"path/filepath"
	"testing/fstest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
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
		// Artist/
		// ├── artist.jpg               ← matched by artist.*
		// └── Album/
		//     └── 01 - Track.mp3
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
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── artist.jpg           ← matched by album/artist.*
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
		// Artist/
		// ├── artist.jpg               ← wins (artist.* before album/artist.*)
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── artist.jpg
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

	When("an artist has an uploaded image and a matching artist.* file", func() {
		// <DataFolder>/
		// └── artwork/
		//     └── artist/
		//         └── <id>_upload.jpg  ← wins (uploaded image beats the priority chain)
		// Library:
		// Artist/
		// ├── artist.jpg               (ignored — uploaded image comes first)
		// └── Album/
		//     └── 01 - Track.mp3
		It("prefers the uploaded image over any priority-chain match", func() {
			conf.Server.ArtistArtPriority = "artist.*, album/artist.*, external"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"albumartist": "Artist"}),
				"Artist/artist.jpg":           imageFile("artist-folder"),
			})
			scan()
			ar := soleArtist()

			uploaded := ar.ID + "_upload.jpg"
			writeUploadedImage(consts.EntityArtist, uploaded, imageBytes("artist-uploaded"))
			ar.UploadedImage = uploaded
			Expect(ds.Artist(ctx).Put(&ar)).To(Succeed())

			artID := model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil)
			Expect(readArtwork(artID)).To(Equal(imageBytes("artist-uploaded")))
		})
	})

	When("ArtistArtPriority uses album/<arbitrary pattern> (not just album/artist.*)", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── artist.jpg            ← matched by album/artist.*
		It("resolves the pattern against the artist's album image files", func() {
			conf.Server.ArtistArtPriority = "album/artist.*, external"
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

	When("ArtistArtPriority starts with image-folder and ArtistImageFolder has a name-matching image", func() {
		// <ArtistImageFolder>/
		// └── Artist.jpg               ← matched by artist name (image-folder source)
		// Library:
		// Artist/
		// └── Album/
		//     └── 01 - Track.mp3       (no artist.* present in library)
		It("returns the image from the configured artist image folder", func() {
			imgFolder := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(imgFolder, "Artist.jpg"), imageBytes("image-folder"), 0600)).To(Succeed())
			conf.Server.ArtistImageFolder = imgFolder
			conf.Server.ArtistArtPriority = "image-folder, artist.*, album/artist.*"

			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"albumartist": "Artist"}),
			})
			scan()

			ar := soleArtist()
			artID := model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil)
			Expect(readArtwork(artID)).To(Equal(imageBytes("image-folder")))
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
