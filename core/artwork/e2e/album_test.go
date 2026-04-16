package artworke2e_test

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const defaultCoverPriority = "cover.*, folder.*, front.*, embedded, external"

var _ = Describe("Album artwork resolution", func() {
	BeforeEach(func() {
		setupHarness()
	})

	When("an album has a single folder with cover.jpg at the album root", func() {
		It("returns the album-root cover", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			writeLayout(
				mp3Naked("Artist/Album/01 - Track.mp3"),
				imgJPEG("Artist/Album/cover.jpg", "album-root"),
			)
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(jpegLabel("album-root")))
		})
	})

	// Bug 2 variant: cover.* basenames tie across album-root and per-disc folders;
	// compareImageFiles' lexicographic full-path tiebreaker ranks disc-subfolder
	// files first. Flip from PIt to It once it prefers shorter/parent paths.
	When("a multi-disc album has a cover.jpg at the album root and per-disc covers", func() {
		PIt("uses the album-root cover (currently picks a disc subfolder image — bug)", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			writeLayout(
				mp3Naked("Artist/Album/CD1/01 - Track.mp3"),
				mp3Naked("Artist/Album/CD2/01 - Track.mp3"),
				imgJPEG("Artist/Album/cover.jpg", "album-root"),
				imgJPEG("Artist/Album/CD1/cover.jpg", "disc1"),
				imgJPEG("Artist/Album/CD2/cover.jpg", "disc2"),
			)
			scan()

			al := firstAlbum()
			Expect(al.FolderIDs).To(HaveLen(2),
				"sanity check: scanner should treat the two disc subfolders as one multi-disc album")
			Expect(readArtwork(al.CoverArtID())).To(Equal(jpegLabel("album-root")))
		})
	})

	// Bug 2: folder.jpg basenames tie across album-root and per-disc folders;
	// the lexicographic full-path tiebreaker in compareImageFiles ranks
	// "Artist/Album/CD1/folder.jpg" ahead of "Artist/Album/folder.jpg".
	// Flip from PIt to It once compareImageFiles prefers shorter/parent paths.
	When("a multi-disc album has folder.jpg at the album root AND in each disc subfolder", func() {
		PIt("uses the album-root folder.jpg (currently picks a disc subfolder image — bug)", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			writeLayout(
				mp3Naked("Artist/Album/CD1/01 - Track.mp3"),
				mp3Naked("Artist/Album/CD2/01 - Track.mp3"),
				imgJPEG("Artist/Album/folder.jpg", "album-root"),
				imgJPEG("Artist/Album/CD1/folder.jpg", "disc1"),
				imgJPEG("Artist/Album/CD2/folder.jpg", "disc2"),
			)
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(jpegLabel("album-root")))
		})
	})

	// Bug 1: commonParentFolder's `len(folders) < 2` guard skips the parent-folder
	// lookup whenever an album lives entirely under a single subfolder, so an
	// album-root cover is never considered. Flip from PIt to It once the guard
	// accepts single-folder albums whose parent isn't already in the folder set.
	When("an album lives entirely under a single disc subfolder with cover.jpg at the parent", func() {
		PIt("uses the parent-folder cover (currently ignored — bug)", func() {
			conf.Server.CoverArtPriority = defaultCoverPriority
			writeLayout(
				mp3Naked("Artist/Album/disc1/01 - Track.mp3"),
				imgJPEG("Artist/Album/cover.jpg", "album-root"),
			)
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(jpegLabel("album-root")))
		})
	})

	When("CoverArtPriority puts embedded first and the album has both embedded and external art", func() {
		It("returns the embedded image", func() {
			conf.Server.CoverArtPriority = "embedded, cover.*, folder.*, front.*, external"
			writeLayout(
				mp3Embedded("Artist/Album/01 - Track.mp3"),
				imgJPEG("Artist/Album/cover.jpg", "external"),
			)
			scan()

			al := firstAlbum()
			data := readArtwork(al.CoverArtID())
			Expect(data).ToNot(Equal(jpegLabel("external")))
			Expect(data).ToNot(BeEmpty())
		})
	})

	When("CoverArtPriority lists external first but no external file is present", func() {
		It("falls through to embedded artwork", func() {
			conf.Server.CoverArtPriority = "external, embedded"
			writeLayout(
				mp3Embedded("Artist/Album/01 - Track.mp3"),
			)
			scan()

			al := firstAlbum()
			data := readArtwork(al.CoverArtID())
			Expect(data).ToNot(BeEmpty())
		})
	})

	When("the only cover file uses uppercase extension and a different case in its name", func() {
		It("matches case-insensitively against cover.*", func() {
			conf.Server.CoverArtPriority = "cover.*, folder.*"
			writeLayout(
				mp3Naked("Artist/Album/01 - Track.mp3"),
				imgJPEG("Artist/Album/Cover.JPG", "case-insensitive"),
			)
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(jpegLabel("case-insensitive")))
		})
	})

	When("two cover files have basenames that tie under the natural-sort tiebreaker", func() {
		It("prefers the file without a numeric suffix", func() {
			conf.Server.CoverArtPriority = "cover.*"
			writeLayout(
				mp3Naked("Artist/Album/01 - Track.mp3"),
				imgJPEG("Artist/Album/cover.jpg", "primary"),
				imgJPEG("Artist/Album/cover.1.jpg", "secondary"),
			)
			scan()

			al := firstAlbum()
			Expect(readArtwork(al.CoverArtID())).To(Equal(jpegLabel("primary")))
		})
	})

	When("the album has no cover and CoverArtPriority lists only file patterns", func() {
		It("returns ErrUnavailable", func() {
			conf.Server.CoverArtPriority = "cover.*, folder.*"
			writeLayout(
				mp3Naked("Artist/Album/01 - Track.mp3"),
			)
			scan()

			al := firstAlbum()
			_, err := readArtworkOrErr(model.NewArtworkID(model.KindAlbumArtwork, al.ID, &al.UpdatedAt))
			Expect(err).To(HaveOccurred())
		})
	})
})
