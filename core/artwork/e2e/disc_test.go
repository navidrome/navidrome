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
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── disc1.jpg            ← matched by disc*.*
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
		// Artist/
		// └── Album/
		//     └── 01 - Track.mp3       (no disc or album art — returns ErrUnavailable)
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
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── cover.jpg            ← album-level fallback (no disc art present)
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
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     ├── disc1.jpg            ← matches request for disc 1
		//     └── disc10.jpg
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
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── disc1.jpg        ← matches request for disc 1
		//     └── CD2/
		//         ├── 01 - Track.mp3
		//         └── disc2.jpg        ← matches request for disc 2
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

	// Doc scenarios from:
	// https://www.navidrome.org/docs/usage/library/artwork/#disc-cover-art
	// Default DiscArtPriority is "disc*.*, cd*.*, cover.*, folder.*, front.*, discsubtitle, embedded".
	When("a disc subfolder has a cd2.png image", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── disc1.jpg
		//     └── CD2/
		//         ├── 01 - Track.mp3
		//         └── cd2.png          ← matched by cd*.* for disc 2
		It("matches via the cd*.* pattern", func() {
			conf.Server.DiscArtPriority = defaultDiscPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/CD1/disc1.jpg":      imageFile("disc-1"),
				"Artist/Album/CD2/cd2.png":        imageFile("cd-2"),
			})
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 2), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(imageBytes("cd-2")))
		})
	})

	When("a disc subfolder has cover.jpg but no disc*.*/cd*.* image", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── cover.jpg        ← matched by cover.* inside disc folder
		//     └── CD2/
		//         ├── 01 - Track.mp3
		//         └── cover.jpg
		It("falls through to cover.* inside the disc folder", func() {
			conf.Server.DiscArtPriority = defaultDiscPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/CD1/cover.jpg":      imageFile("disc1-cover"),
				"Artist/Album/CD2/cover.jpg":      imageFile("disc2-cover"),
			})
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(imageBytes("disc1-cover")))
		})
	})

	When("DiscArtPriority is the empty string", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── disc1.jpg        (ignored — DiscArtPriority is empty)
		//     ├── CD2/
		//     │   ├── 01 - Track.mp3
		//     │   └── cd2.png          (ignored — DiscArtPriority is empty)
		//     └── cover.jpg            ← used for every disc (album-level fallback)
		It("skips every disc-level source and returns the album cover", func() {
			conf.Server.DiscArtPriority = ""
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/CD1/disc1.jpg":      imageFile("disc-1"),
				"Artist/Album/CD2/cd2.png":        imageFile("cd-2"),
				"Artist/Album/cover.jpg":          imageFile("album-cover"),
			})
			scan()

			al := firstAlbum()
			for _, n := range []int{1, 2} {
				discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, n), &al.UpdatedAt)
				Expect(readArtwork(discID)).To(Equal(imageBytes("album-cover")),
					"disc %d should use the album cover when DiscArtPriority is empty", n)
			}
		})
	})

	When("the documented multi-disc layout is used (disc1.jpg + cd2.png + album-root cover.jpg)", func() {
		// Artist/
		// └── Album/
		//     ├── disc1/
		//     │   ├── disc1.jpg        ← matched by disc*.* for disc 1
		//     │   ├── 01 - Track.mp3
		//     │   └── 02 - Track.mp3
		//     ├── disc2/
		//     │   ├── cd2.png          ← matched by cd*.* for disc 2
		//     │   ├── 01 - Track.mp3
		//     │   └── 02 - Track.mp3
		//     └── cover.jpg            (album-level fallback, unused here)
		It("matches the per-disc image for each disc", func() {
			conf.Server.DiscArtPriority = defaultDiscPriority
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/disc1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/disc1/02 - Track.mp3": trackFile(2, "T2", map[string]any{"disc": "1"}),
				"Artist/Album/disc2/01 - Track.mp3": trackFile(1, "T3", map[string]any{"disc": "2"}),
				"Artist/Album/disc2/02 - Track.mp3": trackFile(2, "T4", map[string]any{"disc": "2"}),
				"Artist/Album/disc1/disc1.jpg":      imageFile("disc-1"),
				"Artist/Album/disc2/cd2.png":        imageFile("cd-2"),
				"Artist/Album/cover.jpg":            imageFile("album-root"),
			})
			scan()

			al := firstAlbum()
			disc1ID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			disc2ID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 2), &al.UpdatedAt)
			Expect(readArtwork(disc1ID)).To(Equal(imageBytes("disc-1")))
			Expect(readArtwork(disc2ID)).To(Equal(imageBytes("cd-2")))
		})
	})

	When("discsubtitle keyword matches an image whose stem equals the disc's subtitle", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3       (discsubtitle="Bonus Tracks")
		//     └── Bonus Tracks.jpg     ← matched by "discsubtitle" keyword
		It("selects the subtitle-named image", func() {
			conf.Server.DiscArtPriority = "discsubtitle"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3":   trackFile(1, "T1", map[string]any{"disc": "1", "discsubtitle": "Bonus Tracks"}),
				"Artist/Album/Bonus Tracks.jpg": imageFile("bonus-tracks"),
			})
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(imageBytes("bonus-tracks")))
		})
	})

	When("discsubtitle is set but no image filename matches the subtitle", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3       (discsubtitle="Bonus Tracks")
		//     └── cover.jpg            ← wins (discsubtitle has no match, falls through)
		It("falls through to the next priority entry", func() {
			conf.Server.DiscArtPriority = "discsubtitle, cover.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1", "discsubtitle": "Bonus Tracks"}),
				"Artist/Album/cover.jpg":      imageFile("cover"),
			})
			scan()

			al := firstAlbum()
			discID := model.NewArtworkID(model.KindDiscArtwork, model.DiscArtworkID(al.ID, 1), &al.UpdatedAt)
			Expect(readArtwork(discID)).To(Equal(imageBytes("cover")))
		})
	})
})
