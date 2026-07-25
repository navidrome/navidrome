package e2e

import (
	"testing/fstest"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Doc reference:
// https://www.navidrome.org/docs/usage/library/artwork/#mediafiles
// Navidrome resolves mediafile artwork in this order:
//  1. Embedded image from the mediafile itself
//  2. For multi-disc albums, disc-level artwork
//  3. Album cover art
//
// Embedded art lands in the content-addressed store (asserted byte-for-byte); disc-level art is a
// serve-time read through the library FS.
var _ = Describe("MediaFile artwork resolution", func() {
	BeforeEach(func() {
		setupResolutionHarness()
	})

	When("a multi-disc album track has no embedded art", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── disc1.jpg
		//     ├── CD2/
		//     │   ├── 01 - Track.mp3   ← track requested
		//     │   └── disc2.jpg        ← wins (disc-level before album-level)
		//     └── cover.jpg
		It("falls back to the disc-level artwork (not the album cover)", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			conf.Server.DiscArtPriority = defaultDiscPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/CD1/disc1.jpg":      smallPNG("disc-1"),
				"Artist/Album/CD2/disc2.jpg":      smallPNG("disc-2"),
				"Artist/Album/cover.jpg":          smallPNG("album-root"),
			})
			scan()

			mf := mediafileOn("Artist/Album/CD2/01 - Track.mp3")
			Expect(serveBytes(mf.CoverArtID())).To(Equal(pngBytes("disc-2")))
		})
	})

	When("a single-disc album track has no embedded art", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3       ← track requested
		//     └── cover.jpg            ← wins (album-level fallback, no disc subfolder)
		It("falls back to the album cover", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			conf.Server.DiscArtPriority = defaultDiscPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":      smallPNG("album-cover"),
			})
			scan()

			mf := mediafileOn("Artist/Album/01 - Track.mp3")
			Expect(serveBytes(mf.CoverArtID())).To(Equal(pngBytes("album-cover")))
		})
	})

	When("a multi-disc album track has no embedded art and the disc has no disc-level image", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   └── 01 - Track.mp3
		//     ├── CD2/
		//     │   └── 01 - Track.mp3   ← track requested
		//     └── cover.jpg            ← wins (no disc image → album-level fallback)
		It("falls through from disc to album cover", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			conf.Server.DiscArtPriority = defaultDiscPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/cover.jpg":          smallPNG("album-root"),
			})
			scan()

			mf := mediafileOn("Artist/Album/CD2/01 - Track.mp3")
			Expect(serveBytes(mf.CoverArtID())).To(Equal(pngBytes("album-root")))
		})
	})

	When("a track has its own embedded art", func() {
		// Artist/
		// └── Album/
		//     └── 01 - Track.mp3       ← has embedded picture (wins over every fallback)
		It("resolves the track's embedded image into the store", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"has_picture": "true"}),
			})
			scan()
			replaceWithRealMP3("Artist/Album/01 - Track.mp3")

			mf := mediafileOn("Artist/Album/01 - Track.mp3")
			ia := acquire(model.KindMediaFileArtwork, mf.ID)
			Expect(ia.Source).To(Equal("embedded"))
			Expect(storedBytes(ia)).To(Equal(embeddedArtBytes))
		})
	})

	When("EnableMediaFileCoverArt is turned off after the track was scanned", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3       ← has embedded picture (must NOT be served)
		//     └── cover.jpg            ← wins (per-track art disabled at serve time)
		It("serves the album cover instead of the track's embedded art", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"has_picture": "true"}),
				"Artist/Album/cover.jpg":      smallPNG("album-cover"),
			})
			scan()
			replaceWithRealMP3("Artist/Album/01 - Track.mp3")

			// The setting is not part of the artwork fingerprint, so a direct mf- request must
			// honor it at serve time rather than serving previously-eligible embedded art.
			conf.Server.EnableMediaFileCoverArt = false
			mf := mediafileOn("Artist/Album/01 - Track.mp3")
			trackArtID := model.NewArtworkID(model.KindMediaFileArtwork, mf.ID, nil)
			Expect(serveBytes(trackArtID)).To(Equal(pngBytes("album-cover")))
		})
	})
})

func mediafileOn(relPath string) model.MediaFile {
	GinkgoHelper()
	mfs, err := rds.MediaFile(rctx).GetAll(model.QueryOptions{
		Filters: squirrel.Like{"media_file.path": relPath},
	})
	Expect(err).ToNot(HaveOccurred())
	if len(mfs) == 0 {
		Fail("mediafile not found: " + relPath)
		return model.MediaFile{}
	}
	return mfs[0]
}
