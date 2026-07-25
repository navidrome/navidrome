package e2e

import (
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Folder-backed album art is served via os.Open(SourcePath), which the in-memory library FS
// cannot satisfy; the worker's persisted state row (Source + SourcePath) is the resolver's
// selection, so folder scenarios assert on it. Embedded art lands in the content-addressed store
// and is asserted byte-for-byte; a no-art album settles absent.
var _ = Describe("Album artwork resolution", func() {
	BeforeEach(func() {
		setupResolutionHarness()
	})

	expectFolderCover := expectAlbumFolderCover
	expectAbsent := expectAlbumAbsent

	When("an album has a single folder with cover.jpg at the album root", func() {
		// Artist/
		// └── Album/
		//     ├── 01 - Track.mp3
		//     └── cover.jpg            ← matched by cover.*
		It("returns the album-root cover", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":      smallPNG("album-root"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/cover.jpg")
		})
	})

	// https://github.com/navidrome/navidrome/issues/5376
	// cover.* basenames tie across album-root and per-disc folders;
	// compareImageFiles must prefer shallower paths.
	When("a multi-disc album has a cover.jpg at the album root and per-disc covers", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── cover.jpg        ← should not win
		//     ├── CD2/
		//     │   ├── 01 - Track.mp3
		//     │   └── cover.jpg
		//     └── cover.jpg            ← should win (album-root fallback)
		It("prefers the album-root cover over per-disc covers", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "Track CD1"),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "Track CD2"),
				"Artist/Album/cover.jpg":          smallPNG("album-root"),
				"Artist/Album/CD1/cover.jpg":      smallPNG("disc1"),
				"Artist/Album/CD2/cover.jpg":      smallPNG("disc2"),
			})
			scan()

			al := firstAlbum()
			Expect(al.FolderIDs).To(HaveLen(2),
				"sanity check: the two disc subfolders should form one multi-disc album")
			expectFolderCover(al, "Artist/Album/cover.jpg")
		})
	})

	// https://github.com/navidrome/navidrome/issues/5376
	// folder.jpg basenames tie across album-root and per-disc folders;
	// compareImageFiles must prefer shallower paths.
	When("a multi-disc album has folder.jpg at the album root AND in each disc subfolder", func() {
		// Artist/
		// └── Album/
		//     ├── CD1/
		//     │   ├── 01 - Track.mp3
		//     │   └── folder.jpg       ← should not win
		//     ├── CD2/
		//     │   ├── 01 - Track.mp3
		//     │   └── folder.jpg
		//     └── folder.jpg           ← should win (album-root fallback)
		It("prefers the album-root folder.jpg over per-disc folder.jpg", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/CD1/01 - Track.mp3": trackFile(1, "Track CD1"),
				"Artist/Album/CD2/01 - Track.mp3": trackFile(1, "Track CD2"),
				"Artist/Album/folder.jpg":         smallPNG("album-root"),
				"Artist/Album/CD1/folder.jpg":     smallPNG("disc1"),
				"Artist/Album/CD2/folder.jpg":     smallPNG("disc2"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/folder.jpg")
		})
	})

	// https://github.com/navidrome/navidrome/issues/5376
	// Single-subfolder albums must still consider the parent folder's images.
	When("an album lives entirely under a single disc subfolder with cover.jpg at the parent", func() {
		// Artist/
		// └── Album/
		//     ├── disc1/
		//     │   └── 01 - Track.mp3
		//     └── cover.jpg            ← should win (parent-folder fallback)
		It("uses the parent-folder cover for single-disc-subfolder albums", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/Album/disc1/01 - Track.mp3": trackFile(1, "Track"),
				"Artist/Album/cover.jpg":            smallPNG("album-root"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/cover.jpg")
		})
	})

	// https://github.com/navidrome/navidrome/issues/5456
	When("a top-level multi-disc album has cover.jpg at the album root and per-disc folder.jpg", func() {
		// Album/                        (top-level folder, Path=".")
		// ├── CD1/
		// │   ├── 01 - Track.mp3
		// │   └── folder.jpg
		// ├── CD2/
		// │   ├── 01 - Track.mp3
		// │   └── folder.jpg
		// └── cover.jpg                ← should win (album-root)
		It("prefers the album-root cover.jpg", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Album/CD1/01 - Track.mp3": trackFile(1, "Track CD1"),
				"Album/CD2/01 - Track.mp3": trackFile(1, "Track CD2"),
				"Album/cover.jpg":          smallPNG("album-root"),
				"Album/CD1/folder.jpg":     smallPNG("disc1"),
				"Album/CD2/folder.jpg":     smallPNG("disc2"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Album/cover.jpg")
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
				"Artist/Album/cover.jpg":      smallPNG("external"),
			})
			scan()
			replaceWithRealMP3("Artist/Album/01 - Track.mp3")

			ia := acquire(model.KindAlbumArtwork, firstAlbum().ID)
			Expect(ia.Source).To(Equal("embedded"))
			Expect(storedBytes(ia)).To(Equal(embeddedArtBytes))
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

			ia := acquire(model.KindAlbumArtwork, firstAlbum().ID)
			Expect(ia.Source).To(Equal("embedded"))
			Expect(storedBytes(ia)).To(Equal(embeddedArtBytes))
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
				"Artist/Album/Cover.JPG":      smallPNG("case-insensitive"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/Cover.JPG")
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
				"Artist/Album/cover.jpg":      smallPNG("primary"),
				"Artist/Album/cover.1.jpg":    smallPNG("secondary"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/cover.jpg")
		})
	})

	When("the album has no cover and CoverArtPriority lists only file patterns", func() {
		// Artist/
		// └── Album/
		//     └── 01 - Track.mp3       (no image files — settles absent)
		It("settles absent", func() {
			conf.Server.CoverArtPriority = "cover.*, folder.*"
			setLayout(fstest.MapFS{
				"Artist/Album/01 - Track.mp3": trackFile(1, "Track"),
			})
			scan()
			expectAbsent(firstAlbum())
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
				"Artist/Album/folder.jpg":     smallPNG("folder"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/folder.jpg")
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
				"Artist/Album/front.jpg":      smallPNG("front"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/front.jpg")
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
				"Artist/Album/cover.jpg":      smallPNG("cover"),
				"Artist/Album/folder.jpg":     smallPNG("folder"),
				"Artist/Album/front.jpg":      smallPNG("front"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/cover.jpg")
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
				"Artist/Album/folder.jpg":     smallPNG("folder"),
				"Artist/Album/front.jpg":      smallPNG("front"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/folder.jpg")
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
				"Artist/Album/cover.2.jpg":    smallPNG("second"),
				"Artist/Album/cover.jpg":      smallPNG("primary"),
				"Artist/Album/cover.1.jpg":    smallPNG("first"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/cover.jpg")
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
				"Artist/Album/cover.jpg":      smallPNG("cover"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/cover.jpg")
		})
	})

	// Regression introduced in v0.62.0 (#5451 + #5457): the parent-folder
	// fallback can pick up images from the ARTIST folder, serving the artist
	// thumbnail as album art for any album without its own image files.
	When("an album has no images and the artist folder has folder.jpg", func() {
		// Artist/
		// ├── folder.jpg               ← artist thumbnail, must NOT become album art
		// ├── Album A/
		// │   └── 01 - Track.mp3       (no images)
		// └── Album B/
		//     ├── 01 - Track.mp3
		//     └── cover.jpg
		It("does not use the artist image as album art", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/folder.jpg":             smallPNG("artist-thumbnail"),
				"Artist/Album A/01 - Track.mp3": trackFile(1, "Track A", map[string]any{"album": "Album A", "albumartist": "Artist"}),
				"Artist/Album B/01 - Track.mp3": trackFile(1, "Track B", map[string]any{"album": "Album B", "albumartist": "Artist"}),
				"Artist/Album B/cover.jpg":      smallPNG("album-b"),
			})
			scan()

			expectAbsent(albumByName("Album A"))
			expectFolderCover(albumByName("Album B"), "Artist/Album B/cover.jpg")
		})
	})

	When("a single-disc album is spread across sibling folders under the artist folder", func() {
		// Artist/
		// ├── folder.jpg               ← artist thumbnail, must NOT become album art
		// ├── Album A/
		// │   └── 01 - Track.mp3       (album: "Album A")
		// ├── Album A bonus/
		// │   └── 02 - Track.mp3       (album: "Album A" — same album, second folder)
		// └── Album B/
		//     ├── 01 - Track.mp3
		//     └── cover.jpg
		It("does not use the artist image as album art for the spread album", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/folder.jpg":                   smallPNG("artist-thumbnail"),
				"Artist/Album A/01 - Track.mp3":       trackFile(1, "Track A1", map[string]any{"album": "Album A", "albumartist": "Artist"}),
				"Artist/Album A bonus/02 - Track.mp3": trackFile(2, "Track A2", map[string]any{"album": "Album A", "albumartist": "Artist"}),
				"Artist/Album B/01 - Track.mp3":       trackFile(1, "Track B", map[string]any{"album": "Album B", "albumartist": "Artist"}),
				"Artist/Album B/cover.jpg":            smallPNG("album-b"),
			})
			scan()

			alA := albumByName("Album A")
			Expect(alA.FolderIDs).To(HaveLen(2),
				"sanity check: the two sibling folders should form one spread album")
			expectAbsent(alA)
		})
	})

	When("a spread album has its own front.jpg but the artist folder has cover.jpg", func() {
		// Artist/
		// ├── cover.jpg                ← artist image; matches cover.* (first pattern),
		// │                              must NOT shadow the album's own front.jpg
		// ├── Album A/
		// │   ├── 01 - Track.mp3       (album: "Album A")
		// │   └── front.jpg            ← should win
		// ├── Album A bonus/
		// │   └── 02 - Track.mp3       (album: "Album A")
		// └── Album B/
		//     └── 01 - Track.mp3       other-album audio, so the artist folder is
		//                              correctly rejected as Album A's root
		It("prefers the album's own art over the artist image", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			setLayout(fstest.MapFS{
				"Artist/cover.jpg":                    smallPNG("artist-image"),
				"Artist/Album A/01 - Track.mp3":       trackFile(1, "Track A1", map[string]any{"album": "Album A", "albumartist": "Artist"}),
				"Artist/Album A/front.jpg":            smallPNG("album-a-front"),
				"Artist/Album A bonus/02 - Track.mp3": trackFile(2, "Track A2", map[string]any{"album": "Album A", "albumartist": "Artist"}),
				"Artist/Album B/01 - Track.mp3":       trackFile(1, "Track B", map[string]any{"album": "Album B", "albumartist": "Artist"}),
			})
			scan()

			alA := albumByName("Album A")
			Expect(alA.FolderIDs).To(HaveLen(2),
				"sanity check: the two sibling folders should form one spread album")
			expectFolderCover(alA, "Artist/Album A/front.jpg")
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
				"Artist/Album/cover.jpg":      smallPNG("cover"),
			})
			scan()
			expectFolderCover(firstAlbum(), "Artist/Album/cover.jpg")
		})
	})
})
