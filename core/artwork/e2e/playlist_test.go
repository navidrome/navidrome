package artworke2e_test

import (
	"os"
	"path/filepath"
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Playlist artwork resolves in this priority order:
//  1. Uploaded image (<DataFolder>/artwork/playlist/<file>)
//  2. Sidecar image next to the .m3u file (same basename, any image ext)
//  3. ExternalImageURL (http/https requires EnableM3UExternalAlbumArt; local path always allowed)
//  4. Generated 2x2 tiled cover from the playlist's albums
//  5. Album placeholder image
//
// The library FS is FakeFS, but uploaded/sidecar/local-external images are
// real files on disk — the reader reads them via os.Open, so the tests
// place them in a real tempdir under DataFolder.
var _ = Describe("Playlist artwork resolution", func() {
	BeforeEach(func() {
		setupHarness()
	})

	When("a playlist has an uploaded image", func() {
		// <DataFolder>/
		// └── artwork/
		//     └── playlist/
		//         └── pl-1_upload.jpg   ← matched by UploadedImagePath() (highest priority)
		It("returns the uploaded image bytes", func() {
			writeUploadedImage(consts.EntityPlaylist, "pl-1_upload.jpg", imageBytes("playlist-upload"))

			pl := putPlaylist(model.Playlist{ID: "pl-1", Name: "Test", UploadedImage: "pl-1_upload.jpg"})

			Expect(readArtwork(pl.CoverArtID())).To(Equal(imageBytes("playlist-upload")))
		})
	})

	When("a playlist has no uploaded image but a sidecar image beside its .m3u file", func() {
		// <tempdir>/
		// ├── MyList.m3u
		// └── MyList.jpg               ← matched by sidecar (same basename, case-insensitive)
		It("returns the sidecar image", func() {
			dir := GinkgoT().TempDir()
			m3uPath := filepath.Join(dir, "MyList.m3u")
			Expect(os.WriteFile(m3uPath, []byte("#EXTM3U\n"), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "MyList.jpg"), imageBytes("sidecar"), 0600)).To(Succeed())

			pl := putPlaylist(model.Playlist{ID: "pl-2", Name: "MyList", Path: m3uPath})

			Expect(readArtwork(pl.CoverArtID())).To(Equal(imageBytes("sidecar")))
		})
	})

	When("a playlist's sidecar uses a different extension case", func() {
		// <tempdir>/
		// ├── MyList.m3u
		// └── MyList.PNG               ← matched case-insensitively
		It("matches case-insensitively", func() {
			dir := GinkgoT().TempDir()
			m3uPath := filepath.Join(dir, "MyList.m3u")
			Expect(os.WriteFile(m3uPath, []byte("#EXTM3U\n"), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "MyList.PNG"), imageBytes("sidecar-png"), 0600)).To(Succeed())

			pl := putPlaylist(model.Playlist{ID: "pl-3", Name: "MyList", Path: m3uPath})

			Expect(readArtwork(pl.CoverArtID())).To(Equal(imageBytes("sidecar-png")))
		})
	})

	When("a playlist has an ExternalImageURL pointing to a local file", func() {
		// <tempdir>/
		// └── cover.jpg                ← absolute path stored in ExternalImageURL
		It("returns the local file regardless of EnableM3UExternalAlbumArt", func() {
			conf.Server.EnableM3UExternalAlbumArt = false // local paths bypass the toggle
			dir := GinkgoT().TempDir()
			imgPath := filepath.Join(dir, "cover.jpg")
			Expect(os.WriteFile(imgPath, imageBytes("external-local"), 0600)).To(Succeed())

			pl := putPlaylist(model.Playlist{ID: "pl-4", Name: "WithExt", ExternalImageURL: imgPath})

			Expect(readArtwork(pl.CoverArtID())).To(Equal(imageBytes("external-local")))
		})
	})

	When("a playlist has an http(s) ExternalImageURL and EnableM3UExternalAlbumArt is false", func() {
		// (no local files — http source is gated off, reader falls through to placeholder)
		It("skips the URL and falls through to the bundled placeholder", func() {
			conf.Server.EnableM3UExternalAlbumArt = false

			pl := putPlaylist(model.Playlist{ID: "pl-5", Name: "HttpGated", ExternalImageURL: "https://example.com/cover.jpg"})

			Expect(readArtwork(pl.CoverArtID())).To(Equal(placeholderBytes()))
		})
	})

	When("a playlist has no images and no tracks", func() {
		// (reader falls all the way through to the bundled album placeholder)
		It("returns the album placeholder", func() {
			pl := putPlaylist(model.Playlist{ID: "pl-6", Name: "Empty"})

			Expect(readArtwork(pl.CoverArtID())).To(Equal(placeholderBytes()))
		})
	})

	When("a playlist has no uploaded/sidecar/external image but has tracks with album covers", func() {
		// Library:
		// Artist/
		// ├── AlbumA/
		// │   ├── 01 - Track.mp3
		// │   └── cover.png          (real PNG — wins as tile 1 source)
		// └── AlbumB/
		//     ├── 01 - Track.mp3
		//     └── cover.png          (real PNG — wins as tile 2 source)
		// Playlist "pl-7" references tracks from both albums, so the reader
		// generates a 2x2 tiled cover from 2 distinct album art tiles (the
		// tiled generator mirrors when it has fewer than 4 unique tiles).
		It("generates a tiled cover from album art", func() {
			conf.Server.CoverArtPriority = "cover.*"
			setLayout(fstest.MapFS{
				"Artist/AlbumA/01 - Track.mp3": trackFile(1, "TA", map[string]any{"album": "AlbumA"}),
				"Artist/AlbumA/cover.png":      realPNG("albumA"),
				"Artist/AlbumB/01 - Track.mp3": trackFile(1, "TB", map[string]any{"album": "AlbumB"}),
				"Artist/AlbumB/cover.png":      realPNG("albumB"),
			})
			scan()

			// Pull the scanned mediafile IDs so we can attach them to the playlist.
			mfs, err := ds.MediaFile(ctx).GetAll(model.QueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).To(HaveLen(2))

			pl := model.Playlist{ID: "pl-7", Name: "Mix", OwnerID: "admin-1"}
			pl.AddMediaFilesByID([]string{mfs[0].ID, mfs[1].ID})
			Expect(ds.Playlist(ctx).Put(&pl)).To(Succeed())

			data := readArtwork(pl.CoverArtID())
			// The tiled cover is a PNG-encoded 600x600 image (tileSize const).
			// Exact bytes vary (random album order), so assert format + non-trivial size.
			Expect(data[:8]).To(Equal([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}))
			Expect(len(data)).To(BeNumerically(">", 1000))
		})
	})
})

func putPlaylist(pl model.Playlist) model.Playlist {
	GinkgoHelper()
	if pl.OwnerID == "" {
		pl.OwnerID = "admin-1"
	}
	Expect(ds.Playlist(ctx).Put(&pl)).To(Succeed())
	return pl
}
