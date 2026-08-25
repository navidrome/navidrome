package e2e

import (
	"image/color"
	"os"
	"path/filepath"
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Playlist artwork resolves in this priority order:
//  1. Uploaded image (<DataFolder>/artwork/playlist/<file>)
//  2. Sidecar image next to the .m3u file (same basename, any image ext)
//  3. ExternalImageURL (http/https requires EnableM3UExternalAlbumArt; local path always allowed)
//  4. Generated 2x2 tiled cover from the playlist's albums
//  5. Absent
//
// The library is an in-memory FS, but uploaded/sidecar/local-external images are real files on
// disk — the resolver reads them via os.Open, so those tests place them in a real tempdir.
var _ = Describe("Playlist artwork resolution", func() {
	BeforeEach(func() {
		setupResolutionHarness()
	})

	When("a playlist has an uploaded image", func() {
		// <DataFolder>/
		// └── artwork/
		//     └── playlist/
		//         └── pl-1_upload.jpg   ← matched by UploadedImagePath() (highest priority)
		It("returns the uploaded image bytes", func() {
			writeUploadedImage(consts.EntityPlaylist, "pl-1_upload.jpg", pngBytes("playlist-upload"))
			pl := putPlaylist(model.Playlist{ID: "pl-1", Name: "Test", UploadedImage: "pl-1_upload.jpg"})

			ia := acquire(model.KindPlaylistArtwork, pl.ID)
			Expect(ia.Source).To(Equal("upload"))
			Expect(serveBytes(pl.CoverArtID())).To(Equal(pngBytes("playlist-upload")))
		})
	})

	When("a playlist has no uploaded image but a sidecar image beside its .m3u file", func() {
		// <tempdir>/
		// ├── MyList.m3u
		// └── MyList.jpg               ← matched by sidecar (same basename, case-insensitive)
		It("returns the sidecar image", func() {
			dir := GinkgoT().TempDir()
			m3uPath := filepath.Join(dir, "MyList.m3u")
			Expect(os.WriteFile(m3uPath, []byte("#EXTM3U\n"), 0o600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "MyList.jpg"), pngBytes("sidecar"), 0o600)).To(Succeed())

			pl := putPlaylist(model.Playlist{ID: "pl-2", Name: "MyList", Path: m3uPath})

			ia := acquire(model.KindPlaylistArtwork, pl.ID)
			Expect(ia.Source).To(Equal("folder"))
			Expect(serveBytes(pl.CoverArtID())).To(Equal(pngBytes("sidecar")))
		})
	})

	When("a playlist's sidecar uses a different extension case", func() {
		// <tempdir>/
		// ├── MyList.m3u
		// └── MyList.PNG               ← matched case-insensitively
		It("matches case-insensitively", func() {
			dir := GinkgoT().TempDir()
			m3uPath := filepath.Join(dir, "MyList.m3u")
			Expect(os.WriteFile(m3uPath, []byte("#EXTM3U\n"), 0o600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(dir, "MyList.PNG"), pngBytes("sidecar-png"), 0o600)).To(Succeed())

			pl := putPlaylist(model.Playlist{ID: "pl-3", Name: "MyList", Path: m3uPath})

			Expect(serveBytes(pl.CoverArtID())).To(Equal(pngBytes("sidecar-png")))
		})
	})

	When("a playlist has an ExternalImageURL pointing to a local file", func() {
		// <tempdir>/
		// └── cover.jpg                ← absolute path stored in ExternalImageURL
		It("returns the local file regardless of EnableM3UExternalAlbumArt", func() {
			conf.Server.EnableM3UExternalAlbumArt = false // local paths bypass the toggle
			dir := GinkgoT().TempDir()
			imgPath := filepath.Join(dir, "cover.jpg")
			Expect(os.WriteFile(imgPath, pngBytes("external-local"), 0o600)).To(Succeed())

			pl := putPlaylist(model.Playlist{ID: "pl-4", Name: "WithExt", ExternalImageURL: imgPath})

			Expect(serveBytes(pl.CoverArtID())).To(Equal(pngBytes("external-local")))
		})
	})

	When("a playlist has an http(s) ExternalImageURL and EnableM3UExternalAlbumArt is false", func() {
		// (no local files — the http source is gated off, so resolution settles absent)
		It("skips the URL and settles absent", func() {
			conf.Server.EnableM3UExternalAlbumArt = false
			pl := putPlaylist(model.Playlist{ID: "pl-5", Name: "HttpGated", ExternalImageURL: "https://example.com/cover.jpg"})

			ia := acquire(model.KindPlaylistArtwork, pl.ID)
			Expect(ia.Hash).To(BeEmpty())
			Expect(serveErr(pl.CoverArtID())).To(MatchError(artwork.ErrUnavailable))

			img, err := rsvc.GetOrPlaceholder(rctx, pl.CoverArtID().String(), 0, false)
			Expect(err).ToNot(HaveOccurred())
			defer img.Close()
			Expect(img.Placeholder).To(BeTrue())
		})
	})

	When("a playlist has no images and no tracks", func() {
		// (no uploaded/sidecar/external image and no album art to sample)
		It("settles absent", func() {
			pl := putPlaylist(model.Playlist{ID: "pl-6", Name: "Empty"})

			ia := acquire(model.KindPlaylistArtwork, pl.ID)
			Expect(ia.Hash).To(BeEmpty())
			Expect(serveErr(pl.CoverArtID())).To(MatchError(artwork.ErrUnavailable))
		})
	})

	When("a playlist has no uploaded/sidecar/external image but has tracks with album covers", func() {
		// Library:
		// Artist/
		// ├── AlbumA/
		// │   ├── 01 - Track.mp3
		// │   └── cover.png          ← tile 1 source
		// └── AlbumB/
		//     ├── 01 - Track.mp3
		//     └── cover.png          ← tile 2 source
		// Playlist "pl-7" references tracks from both albums, so the worker generates a tiled
		// cover from 2 distinct album art tiles (mirrored to fill the 2x2 grid).
		It("generates a tiled cover from album art", func() {
			conf.Server.CoverArtPriority = "cover.*"
			setLayout(fstest.MapFS{
				"Artist/AlbumA/01 - Track.mp3": trackFile(1, "TA", map[string]any{"album": "AlbumA"}),
				"Artist/AlbumA/cover.png":      smallPNG("albumA"),
				"Artist/AlbumB/01 - Track.mp3": trackFile(1, "TB", map[string]any{"album": "AlbumB"}),
				"Artist/AlbumB/cover.png":      smallPNG("albumB"),
			})
			scan()

			mfs, err := rds.MediaFile(rctx).GetAll(model.QueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).To(HaveLen(2))

			pl := model.Playlist{ID: "pl-7", Name: "Mix", OwnerID: "admin-1"}
			pl.AddMediaFilesByID([]string{mfs[0].ID, mfs[1].ID})
			Expect(rds.Playlist(rctx).Put(&pl)).To(Succeed())

			ia := acquire(model.KindPlaylistArtwork, pl.ID)
			Expect(ia.Source).To(Equal("generated"))
			data := storedBytes(ia)
			// The tiled cover is a PNG-encoded image; exact bytes vary (random album order).
			Expect(data[:8]).To(Equal([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}))
			// Two tiles are mirrored into the 2x2 grid as [A B B A], so opposite corners match.
			q := gridQuadrants(data)
			Expect(q[0]).To(Equal(q[3]))
			Expect(q[1]).To(Equal(q[2]))
			Expect(q[0]).ToNot(Equal(q[1]))
		})
	})

	When("a playlist has tracks from four albums, each with its own cover", func() {
		// Library:
		// Artist/
		// ├── AlbumA/{01 - Track.mp3, cover.png}   ← tile 1
		// ├── AlbumB/{01 - Track.mp3, cover.png}   ← tile 2
		// ├── AlbumC/{01 - Track.mp3, cover.png}   ← tile 3
		// └── AlbumD/{01 - Track.mp3, cover.png}   ← tile 4
		It("fills all four grid quadrants with distinct album art", func() {
			conf.Server.CoverArtPriority = "cover.*"
			layout := fstest.MapFS{}
			for _, name := range []string{"AlbumA", "AlbumB", "AlbumC", "AlbumD"} {
				layout["Artist/"+name+"/01 - Track.mp3"] = trackFile(1, "T"+name, map[string]any{"album": name})
				layout["Artist/"+name+"/cover.png"] = smallPNG(name)
			}
			setLayout(layout)
			scan()

			mfs, err := rds.MediaFile(rctx).GetAll(model.QueryOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(mfs).To(HaveLen(4))
			ids := slice.Map(mfs, func(mf model.MediaFile) string { return mf.ID })

			pl := model.Playlist{ID: "pl-8", Name: "Four", OwnerID: "admin-1"}
			pl.AddMediaFilesByID(ids)
			Expect(rds.Playlist(rctx).Put(&pl)).To(Succeed())

			ia := acquire(model.KindPlaylistArtwork, pl.ID)
			Expect(ia.Source).To(Equal("generated"))
			q := gridQuadrants(storedBytes(ia))
			Expect([]color.RGBA{q[0], q[1], q[2], q[3]}).To(HaveLen(4))
			Expect(q[0]).ToNot(Equal(q[1]))
			Expect(q[0]).ToNot(Equal(q[2]))
			Expect(q[0]).ToNot(Equal(q[3]))
			Expect(q[1]).ToNot(Equal(q[2]))
			Expect(q[1]).ToNot(Equal(q[3]))
			Expect(q[2]).ToNot(Equal(q[3]))
		})
	})
})

func putPlaylist(pl model.Playlist) model.Playlist {
	GinkgoHelper()
	if pl.OwnerID == "" {
		pl.OwnerID = "admin-1"
	}
	Expect(rds.Playlist(rctx).Put(&pl)).To(Succeed())
	return pl
}
