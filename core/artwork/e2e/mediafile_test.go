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
// https://www.navidrome.org/docs/usage/library/artwork/#mediafiles
// Navidrome resolves mediafile artwork in this order:
//  1. Embedded image from the mediafile itself
//  2. For multi-disc albums, disc-level artwork
//  3. Album cover art
//
// FakeFS cannot synthesize taglib-readable embedded JPEGs, so scenario (1)
// is covered by the existing embedded-art album tests (which currently
// Skip under FakeFS). The tests below cover (2) and (3): the fallback
// chain for tracks without embedded art.
var _ = Describe("MediaFile artwork fallback", func() {
	BeforeEach(func() {
		setupHarness()
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
				"Artist/Album/CD1/disc1.jpg":      imageFile("disc-1"),
				"Artist/Album/CD2/disc2.jpg":      imageFile("disc-2"),
				"Artist/Album/cover.jpg":          imageFile("album-root"),
			})
			scan()

			mf := mediafileOn("Artist/Album/CD2/01 - Track.mp3")
			Expect(readArtwork(mf.CoverArtID())).To(Equal(imageBytes("disc-2")))
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
				"Artist/Album/cover.jpg":      imageFile("album-cover"),
			})
			scan()

			mf := mediafileOn("Artist/Album/01 - Track.mp3")
			Expect(readArtwork(mf.CoverArtID())).To(Equal(imageBytes("album-cover")))
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
				"Artist/Album/cover.jpg":          imageFile("album-root"),
			})
			scan()

			mf := mediafileOn("Artist/Album/CD2/01 - Track.mp3")
			Expect(readArtwork(mf.CoverArtID())).To(Equal(imageBytes("album-root")))
		})
	})
})

func mediafileOn(relPath string) model.MediaFile {
	GinkgoHelper()
	mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: squirrel.Like{"media_file.path": relPath},
	})
	Expect(err).ToNot(HaveOccurred())
	if len(mfs) == 0 {
		Fail("mediafile not found: " + relPath)
		return model.MediaFile{}
	}
	return mfs[0]
}
