package e2e

import (
	"os"
	"path/filepath"
	"testing/fstest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Doc reference:
// https://www.navidrome.org/docs/usage/library/artwork/#artists
// Default ArtistArtPriority is "artist.*, album/artist.*, external".
// Library-folder images are file-backed (asserted on the worker state row); uploaded and
// image-folder images are real files on disk (asserted byte-for-byte).
var _ = Describe("Artist artwork resolution", func() {
	BeforeEach(func() {
		setupResolutionHarness()
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
				"Artist/artist.jpg":           smallPNG("artist-folder"),
			})
			scan()
			expectArtistFolder(soleArtist(), "Artist/artist.jpg")
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
				"Artist/Album/artist.jpg":     smallPNG("album-artist"),
			})
			scan()
			expectArtistFolder(soleArtist(), "Artist/Album/artist.jpg")
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
				"Artist/artist.jpg":           smallPNG("artist-folder"),
				"Artist/Album/artist.jpg":     smallPNG("album-artist"),
			})
			scan()
			expectArtistFolder(soleArtist(), "Artist/artist.jpg")
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
				"Artist/artist.jpg":           smallPNG("artist-folder"),
			})
			scan()
			ar := soleArtist()

			uploaded := ar.ID + "_upload.jpg"
			writeUploadedImage(consts.EntityArtist, uploaded, pngBytes("artist-uploaded"))
			ar.UploadedImage = uploaded
			Expect(rds.Artist(rctx).Put(&ar)).To(Succeed())

			ia := acquire(model.KindArtistArtwork, ar.ID)
			Expect(ia.Source).To(Equal("upload"))
			Expect(serveBytes(model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil))).To(Equal(pngBytes("artist-uploaded")))
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
				"Artist/Album/artist.jpg":     smallPNG("album-artist"),
			})
			scan()
			expectArtistFolder(soleArtist(), "Artist/Album/artist.jpg")
		})
	})

	// resolveArtist only samples albums where this artist is the SOLE album artist, so a
	// collaboration or compilation never donates its images as the artist's own.
	When("the artist's only album is credited to two album artists", func() {
		// Artist/
		// └── Collab Album/            (album artists: "Artist" + a collaborator)
		//     ├── 01 - Track.mp3
		//     └── artist.jpg           ← must NOT become the artist image
		It("ignores the album's images and settles absent", func() {
			conf.Server.ArtistArtPriority = "album/artist.*"
			// " / " is a default artists split separator, so this single tag yields two album artists.
			setLayout(fstest.MapFS{
				"Artist/Collab Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"albumartist": "Artist / Collaborator"}),
				"Artist/Collab Album/artist.jpg":     smallPNG("collab-artist"),
			})
			scan()
			Expect(firstAlbum().Participants[model.RoleAlbumArtist]).To(HaveLen(2),
				"sanity check: the album must be credited to two album artists")

			ar := soleArtist()
			ia := acquire(model.KindArtistArtwork, ar.ID)
			Expect(ia.Hash).To(BeEmpty())
			Expect(serveErr(model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil))).
				To(MatchError(artwork.ErrUnavailable))
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
			Expect(os.WriteFile(filepath.Join(imgFolder, "Artist.jpg"), pngBytes("image-folder"), 0o600)).To(Succeed())
			conf.Server.ArtistImageFolder = imgFolder
			conf.Server.ArtistArtPriority = "image-folder, artist.*, album/artist.*"

			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"albumartist": "Artist"}),
			})
			scan()

			ar := soleArtist()
			ia := acquire(model.KindArtistArtwork, ar.ID)
			Expect(ia.Source).To(Equal("folder"))
			Expect(serveBytes(model.NewArtworkID(model.KindArtistArtwork, ar.ID, nil))).To(Equal(pngBytes("image-folder")))
		})
	})
})

func soleArtist() model.Artist {
	GinkgoHelper()
	artists, err := rds.Artist(rctx).GetAll(model.QueryOptions{
		Filters: squirrel.Eq{"artist.name": "Artist"},
	})
	Expect(err).ToNot(HaveOccurred())
	if len(artists) == 0 {
		Fail("sole artist not found")
		return model.Artist{}
	}
	return artists[0]
}
