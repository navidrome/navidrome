package e2e

import (
	"fmt"
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Disc art is a serve-time read through the library FS (no worker state row), so per-disc images
// are asserted byte-for-byte. Only multi-disc albums use the disc chain; a single-disc album serves
// its album art directly (a deliberate change from the legacy reader), so those cases are covered by
// the album suite. Album-root covers here are folder-backed and asserted on the album state row.
var _ = Describe("Disc artwork resolution", func() {
	BeforeEach(func() {
		setupResolutionHarness()
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
				"Artist/Album/CD1/disc1.jpg":      smallPNG("disc-1"),
				"Artist/Album/CD2/disc2.jpg":      smallPNG("disc-2"),
			})
			scan()
			expectDiscImage(firstAlbum(), 2, "disc-2")
		})
	})

	When("multiple disc images exist in the same folder (disc1 vs disc10)", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3       (disc 1)
		//     ├── 02 - Track.mp3       (disc 10)
		//     ├── disc1.jpg            ← matches request for disc 1
		//     └── disc10.jpg
		It("matches the requested disc number, not a higher-numbered one", func() {
			conf.Server.DiscArtPriority = "disc*.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/02 - Track.mp3": trackFile(2, "T10", map[string]any{"disc": "10"}),
				"Artist/Album/disc1.jpg":      smallPNG("disc-one"),
				"Artist/Album/disc10.jpg":     smallPNG("disc-ten"),
			})
			scan()
			expectDiscImage(firstAlbum(), 1, "disc-one")
		})
	})

	When("a multi-disc album has no per-disc image but has an album cover", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   └── 01 - Track.mp3
		//     ├── CD2/
		//     │   └── 01 - Track.mp3
		//     └── cover.jpg            ← album-level fallback (no disc art present)
		It("falls back to the album cover", func() {
			conf.Server.DiscArtPriority = "disc*.*, cd*.*"
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/cover.jpg":          smallPNG("album-cover"),
			})
			scan()
			expectDiscImage(firstAlbum(), 1, "album-cover")
		})
	})

	When("a multi-disc album has no per-disc image and no album cover", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   └── 01 - Track.mp3
		//     └── CD2/
		//         └── 01 - Track.mp3   (no images anywhere — nothing to serve)
		It("reports the disc lookup as unavailable", func() {
			conf.Server.DiscArtPriority = "disc*.*, cd*.*"
			conf.Server.CoverArtPriority = "cover.*, folder.*"
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
			})
			scan()
			Expect(serveErr(discArtID(firstAlbum(), 1))).To(MatchError(artwork.ErrUnavailable))
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
				"Artist/Album/CD1/disc1.jpg":      smallPNG("disc-1"),
				"Artist/Album/CD2/cd2.png":        smallPNG("cd-2"),
				"Artist/Album/cover.jpg":          smallPNG("album-cover"),
			})
			scan()

			al := firstAlbum()
			for _, n := range []int{1, 2} {
				expectDiscImage(al, n, "album-cover")
			}
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
				"Artist/Album/CD1/disc1.jpg":      smallPNG("disc-1"),
				"Artist/Album/CD2/cd2.png":        smallPNG("cd-2"),
			})
			scan()
			expectDiscImage(firstAlbum(), 2, "cd-2")
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
				"Artist/Album/CD1/cover.jpg":      smallPNG("disc1-cover"),
				"Artist/Album/CD2/cover.jpg":      smallPNG("disc2-cover"),
			})
			scan()
			expectDiscImage(firstAlbum(), 1, "disc1-cover")
		})
	})

	When("the documented multi-disc layout is used (disc1.jpg + cd2.png + album-root cover.jpg)", func() {
		// Artist/
		// └── Album/
		//     ├── disc1/
		//     │   ├── disc1.jpg        ← matched by disc*.* for disc 1
		//     │   └── 01 - Track.mp3
		//     ├── disc2/
		//     │   ├── cd2.png          ← matched by cd*.* for disc 2
		//     │   └── 01 - Track.mp3
		//     └── cover.jpg            ← album-level cover
		It("matches the per-disc image for each disc and the album-root cover for the album", func() {
			conf.Server.DiscArtPriority = defaultDiscPriority
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/disc1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1"}),
				"Artist/Album/disc2/01 - Track.mp3": trackFile(1, "T3", map[string]any{"disc": "2"}),
				"Artist/Album/disc1/disc1.jpg":      smallPNG("disc-1"),
				"Artist/Album/disc2/cd2.png":        smallPNG("cd-2"),
				"Artist/Album/cover.jpg":            smallPNG("album-root"),
			})
			scan()

			al := firstAlbum()
			expectDiscImage(al, 1, "disc-1")
			expectDiscImage(al, 2, "cd-2")
			expectAlbumFolderCover(al, "Artist/Album/cover.jpg")
		})
	})

	When("discsubtitle keyword matches an image whose stem equals the disc's subtitle", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   └── 01 - Track.mp3   (discsubtitle="Bonus Tracks")
		//     ├── CD2/
		//     │   └── 01 - Track.mp3
		//     └── Bonus Tracks.jpg     ← matched by "discsubtitle" keyword for disc 1
		It("selects the subtitle-named image", func() {
			conf.Server.DiscArtPriority = "discsubtitle"
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1", "discsubtitle": "Bonus Tracks"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/Bonus Tracks.jpg":   smallPNG("bonus-tracks"),
			})
			scan()
			expectDiscImage(firstAlbum(), 1, "bonus-tracks")
		})
	})

	When("discsubtitle is set but no image filename matches the subtitle", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3   (discsubtitle="Bonus Tracks")
		//     │   └── cover.jpg        ← wins (discsubtitle has no match, falls through)
		//     └── CD2/
		//         └── 01 - Track.mp3
		It("falls through to the next priority entry", func() {
			conf.Server.DiscArtPriority = "discsubtitle, cover.*"
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "T1", map[string]any{"disc": "1", "discsubtitle": "Bonus Tracks"}),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "T2", map[string]any{"disc": "2"}),
				"Artist/Album/CD1/cover.jpg":      smallPNG("disc1-cover"),
			})
			scan()
			expectDiscImage(firstAlbum(), 1, "disc1-cover")
		})
	})

	// https://github.com/navidrome/navidrome/issues/5456
	// Top-level album variant — album folder at library root (Path=".").
	When("a top-level multi-disc album has cover.jpg and per-disc folder.jpg", func() {
		// Album/                       (top-level, Path=".")
		// ├── cover.jpg                ← album-level cover
		// ├── Disc 01/
		// │   ├── 01 - Track.mp3
		// │   └── folder.jpg           ← disc 1 art
		// ├── Disc 02/
		// │   ├── 01 - Track.mp3
		// │   └── folder.jpg
		// └── Disc 03/
		//     ├── 01 - Track.mp3
		//     └── folder.jpg
		It("uses album-root cover.jpg for album art and per-disc folder.jpg for each disc", func() {
			conf.Server.DiscArtPriority = defaultDiscPriority
			conf.Server.CoverArtPriority = defaultCoverPriority
			layout := fstest.MapFS{
				"Album/cover.jpg": smallPNG("album-root-cover"),
			}
			for i := 1; i <= 3; i++ {
				prefix := fmt.Sprintf("Album/Disc %02d/", i)
				layout[prefix+"01 - Track.mp3"] = trackFile(1, fmt.Sprintf("T%d", i), map[string]any{"disc": fmt.Sprintf("%d", i)})
				layout[prefix+"folder.jpg"] = smallPNG(fmt.Sprintf("disc-%02d-folder", i))
			}
			setLayout(layout)
			scan()

			al := firstAlbum()
			expectAlbumFolderCover(al, "Album/cover.jpg")
			for i := 1; i <= 3; i++ {
				expectDiscImage(al, i, fmt.Sprintf("disc-%02d-folder", i))
			}
		})
	})

	// Reproduces https://github.com/navidrome/navidrome/issues/5456
	// Deeply nested layout matching the reporter's actual structure.
	When("a deeply nested multi-disc album has cover.jpg and per-disc folder.jpg", func() {
		// Pop; Rock/Grateful Dead/(2001) The Golden Road/   ← album root with cover.jpg
		// ├── cover.jpg                                     ← album-level cover
		// ├── Disc 01 (Subtitle)/
		// │   ├── 01 - Track.mp3
		// │   └── folder.jpg                                ← disc 1 art
		// ├── Disc 02 (Subtitle)/
		// │   ├── 01 - Track.mp3
		// │   └── folder.jpg
		// └── ... (6 discs)
		It("uses album-root cover.jpg for album art and per-disc folder.jpg for each disc", func() {
			conf.Server.DiscArtPriority = defaultDiscPriority
			conf.Server.CoverArtPriority = defaultCoverPriority
			discNames := []string{
				"Disc 01 (Birth of the Dead - The Studio Sides)",
				"Disc 02 (Birth of the Dead - The Live Sides)",
				"Disc 03 (The Grateful Dead)",
				"Disc 04 (Anthem of the Sun)",
				"Disc 05 (Aoxomoxoa)",
				"Disc 06 (Live; Dead)",
			}
			layout := fstest.MapFS{
				"Pop; Rock/Grateful Dead/(2001) The Golden Road/cover.jpg": smallPNG("album-root-cover"),
			}
			for i, name := range discNames {
				discNum := i + 1
				prefix := fmt.Sprintf("Pop; Rock/Grateful Dead/(2001) The Golden Road/%s/", name)
				layout[prefix+"01 - Track.mp3"] = trackFile(1, fmt.Sprintf("T%d", discNum), map[string]any{"disc": fmt.Sprintf("%d", discNum)})
				layout[prefix+"folder.jpg"] = smallPNG(fmt.Sprintf("disc-%02d-folder", discNum))
			}
			setLayout(layout)
			scan()

			al := firstAlbum()
			expectAlbumFolderCover(al, "(2001) The Golden Road/cover.jpg")
			for i := range discNames {
				discNum := i + 1
				expectDiscImage(al, discNum, fmt.Sprintf("disc-%02d-folder", discNum))
			}
		})
	})
})
