package artworke2e_test

import (
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	defaultCoverPriority = "cover.*, folder.*, front.*, embedded, external"
	defaultDiscPriority  = "disc*.*, cd*.*, cover.*, folder.*, front.*, discsubtitle, embedded"
)

var _ = Describe("Album artwork resolution", func() {
	BeforeEach(func() {
		setupHarness()
	})

	When("an album has a single folder with cover.jpg at the album root", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── cover.jpg            ← matched by cover.*
		It("returns the album-root cover", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":      imageFile("album-root"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("album-root")))
		})
	})

	// Bug 2 variant: cover.* basenames tie across album-root and per-disc folders;
	// compareImageFiles' lexicographic full-path tiebreaker ranks disc-subfolder
	// files first.
	When("a multi-disc album has a cover.jpg at the album root and per-disc covers", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── cover.jpg        ← currently wins (bug)
		//     ├── CD2/
		//     │   ├── 01 - Track.mp3
		//     │   └── cover.jpg
		//     └── cover.jpg            ← should win (album-root fallback)
		It("prefers the album-root cover over per-disc covers", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "Track CD1"),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "Track CD2"),
				"Artist/Album/cover.jpg":          imageFile("album-root"),
				"Artist/Album/CD1/cover.jpg":      imageFile("disc1"),
				"Artist/Album/CD2/cover.jpg":      imageFile("disc2"),
			})
			scan()

			al := firstAlbum()
			Expect(al.FolderIDs).To(HaveLen(2),
				"sanity check: scanner should treat the two disc subfolders as one multi-disc album")
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("album-root")))
		})
	})

	// Bug 2: folder.jpg basenames tie across album-root and per-disc folders;
	// the lexicographic full-path tiebreaker in compareImageFiles ranks
	// "Artist/Album/CD1/folder.jpg" ahead of "Artist/Album/folder.jpg".
	When("a multi-disc album has folder.jpg at the album root AND in each disc subfolder", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── folder.jpg       ← currently wins (bug)
		//     ├── CD2/
		//     │   ├── 01 - Track.mp3
		//     │   └── folder.jpg
		//     └── folder.jpg           ← should win (album-root fallback)
		It("prefers the album-root folder.jpg over per-disc folder.jpg", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "Track CD1"),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "Track CD2"),
				"Artist/Album/folder.jpg":         imageFile("album-root"),
				"Artist/Album/CD1/folder.jpg":     imageFile("disc1"),
				"Artist/Album/CD2/folder.jpg":     imageFile("disc2"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("album-root")))
		})
	})

	// Bug 1: commonParentFolder's `len(folders) < 2` guard skips the parent-folder
	// lookup whenever an album lives entirely under a single subfolder, so an
	// album-root cover is never considered.
	When("an album lives entirely under a single disc subfolder with cover.jpg at the parent", func() {
		// Artist/
		// └── Album/
		//     ├── disc1/
		//     │   └── 01 - Track.mp3
		//     └── cover.jpg            ← should win (parent-folder fallback, currently ignored — bug)
		It("uses the parent-folder cover for single-disc-subfolder albums", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/disc1/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":            imageFile("album-root"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("album-root")))
		})
	})

	When("CoverArtPriority puts embedded first and the album has both embedded and external art", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3      ← has embedded picture (wins via "embedded")
		//     └── cover.jpg
		It("returns the embedded image", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, folder.*, front.*, external"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"has_picture": "true"}),
				"Artist/Album/cover.jpg":      imageFile("external"),
			})
			scan()
			// Swap in real MP3 bytes so libFS.Open returns a taglib-readable stream.
			replaceWithRealMP3("Artist/Album/01 - Track.mp3")

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(embeddedArtBytes))
		})
	})

	When("CoverArtPriority lists external first but no external file is present", func() {
		// Artist/
		// └── Album/
		//     └── 01 - Track.mp3      ← has embedded picture (falls through to "embedded")
		It("falls through to embedded artwork", func() {
			conf.Server.CoverArtPriority = "external, embedded"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track", map[string]any{"has_picture": "true"}),
			})
			scan()
			replaceWithRealMP3("Artist/Album/01 - Track.mp3")

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(embeddedArtBytes))
		})
	})

	When("the only cover file uses uppercase extension and a different case in its name", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── Cover.JPG            ← matched case-insensitively by cover.*
		It("matches case-insensitively against cover.*", func() {
			conf.Server.CoverArtPriority = "cover.*, folder.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/Cover.JPG":      imageFile("case-insensitive"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("case-insensitive")))
		})
	})

	When("two cover files have basenames that tie under the natural-sort tiebreaker", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     ├── cover.jpg            ← wins (no numeric suffix)
		//     └── cover.1.jpg
		It("prefers the file without a numeric suffix", func() {
			conf.Server.CoverArtPriority = "cover.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":      imageFile("primary"),
				"Artist/Album/cover.1.jpg":    imageFile("secondary"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("primary")))
		})
	})

	When("the album has no cover and CoverArtPriority lists only file patterns", func() {
		// Artist/
		// └── Album/
		//     └── 01 - Track.mp3       (no image files — returns ErrUnavailable)
		It("returns ErrUnavailable", func() {
			conf.Server.CoverArtPriority = "cover.*, folder.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
			})
			scan()

			al := firstAlbum()
			_, err := readArtworkOrErr(model.NewArtworkID(model.KindAlbumArtwork, al.ID, &al.UpdatedAt))
			Expect(err).To(HaveOccurred())
		})
	})

	// Doc scenarios from:
	// https://www.navidrome.org/docs/usage/library/artwork/#albums
	// Default CoverArtPriority is "cover.*, folder.*, front.*, embedded, external".
	When("only folder.jpg is present (cover.* and front.* missing)", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── folder.jpg           ← matched by folder.*
		It("falls through to folder.jpg", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/folder.jpg":     imageFile("folder"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("folder")))
		})
	})

	When("only front.jpg is present (cover.* and folder.* missing)", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── front.jpg            ← matched by front.*
		It("falls through to front.jpg", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/front.jpg":      imageFile("front"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("front")))
		})
	})

	When("cover.*, folder.*, and front.* all exist in the same folder", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     ├── cover.jpg            ← wins (cover.* is first in priority)
		//     ├── folder.jpg
		//     └── front.jpg
		It("prefers cover.* (first in CoverArtPriority)", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":      imageFile("cover"),
				"Artist/Album/folder.jpg":     imageFile("folder"),
				"Artist/Album/front.jpg":      imageFile("front"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("cover")))
		})
	})

	When("only folder.* and front.* exist (priority order check)", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     ├── folder.jpg           ← wins (folder.* comes before front.*)
		//     └── front.jpg
		It("prefers folder.* over front.*", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/folder.jpg":     imageFile("folder"),
				"Artist/Album/front.jpg":      imageFile("front"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("folder")))
		})
	})

	When("three cover files tie by basename and differ only by numeric suffix", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     ├── cover.jpg            ← wins (no numeric suffix)
		//     ├── cover.1.jpg
		//     └── cover.2.jpg
		It("selects the unsuffixed file first regardless of numeric-suffix order", func() {
			conf.Server.CoverArtPriority = "cover.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.2.jpg":    imageFile("second"),
				"Artist/Album/cover.jpg":      imageFile("primary"),
				"Artist/Album/cover.1.jpg":    imageFile("first"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("primary")))
		})
	})

	When("CoverArtPriority contains an unknown pattern before a matching one", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── cover.jpg            ← wins (unknown "bogus.*" is skipped)
		It("skips the unknown pattern and falls through to the matching one", func() {
			conf.Server.CoverArtPriority = "bogus.*, cover.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":      imageFile("cover"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("cover")))
		})
	})

	When("embedded is first in CoverArtPriority but the track has no embedded art", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3       (no embedded picture)
		//     └── cover.jpg            ← wins (embedded skipped, falls through)
		It("falls through to the next priority entry", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":      imageFile("cover"),
			})
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(imageBytes("cover")))
		})
	})
})
