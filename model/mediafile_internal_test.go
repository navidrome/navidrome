package model

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("fixAlbumArtist", func() {
	var album Album
	BeforeEach(func() {
		album = Album{}
	})
	Context("Non-Compilations", func() {
		BeforeEach(func() {
			album.Compilation = false
			album.Artist = "Sparks"
			album.ArtistID = "ar-123"
		})
		It("returns the track artist if no album artist is specified", func() {
			al := fixAlbumArtist(album, nil)
			Expect(al.AlbumArtistID).To(Equal("ar-123"))
			Expect(al.AlbumArtist).To(Equal("Sparks"))
		})
		It("returns the album artist if it is specified", func() {
			album.AlbumArtist = "Sparks Brothers"
			album.AlbumArtistID = "ar-345"
			al := fixAlbumArtist(album, nil)
			Expect(al.AlbumArtistID).To(Equal("ar-345"))
			Expect(al.AlbumArtist).To(Equal("Sparks Brothers"))
		})
	})
	Context("Compilations", func() {
		BeforeEach(func() {
			album.Compilation = true
			album.Name = "Sgt. Pepper Knew My Father"
			album.AlbumArtistID = "ar-000"
			album.AlbumArtist = "The Beatles"
		})

		It("returns VariousArtists if there's more than one album artist", func() {
			al := fixAlbumArtist(album, []string{"ar-123", "ar-345"})
			Expect(al.AlbumArtistID).To(Equal(consts.VariousArtistsID))
			Expect(al.AlbumArtist).To(Equal(consts.VariousArtists))
		})

		It("returns the sole album artist if they are the same", func() {
			al := fixAlbumArtist(album, []string{"ar-000", "ar-000"})
			Expect(al.AlbumArtistID).To(Equal("ar-000"))
			Expect(al.AlbumArtist).To(Equal("The Beatles"))
		})
	})
})

var _ = Describe("getCoverFromPath", func() {
	var testFolder, testPath, embeddedPath string
	BeforeEach(func() {
		testFolder, _ = os.MkdirTemp("", "album_persistence_tests")
		if err := os.MkdirAll(testFolder, 0777); err != nil {
			panic(err)
		}
		if _, err := os.Create(filepath.Join(testFolder, "Cover.jpeg")); err != nil {
			panic(err)
		}
		if _, err := os.Create(filepath.Join(testFolder, "FRONT.PNG")); err != nil {
			panic(err)
		}
		testPath = filepath.Join(testFolder, "somefile.test")
		embeddedPath = filepath.Join(testFolder, "somefile.mp3")
	})
	AfterEach(func() {
		_ = os.RemoveAll(testFolder)
	})

	It("returns audio file for embedded cover", func() {
		conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
		Expect(getCoverFromPath(testPath, embeddedPath)).To(Equal(""))
	})

	It("returns external file when no embedded cover exists", func() {
		conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
		Expect(getCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "Cover.jpeg")))
	})

	It("returns embedded cover even if not first choice", func() {
		conf.Server.CoverArtPriority = "something.png, embedded, cover.*, front.*"
		Expect(getCoverFromPath(testPath, embeddedPath)).To(Equal(""))
	})

	It("returns first correct match case-insensitively", func() {
		conf.Server.CoverArtPriority = "embedded, cover.jpg, front.svg, front.png"
		Expect(getCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "FRONT.PNG")))
	})

	It("returns match for embedded pattern", func() {
		conf.Server.CoverArtPriority = "embedded, cover.jp?g, front.png"
		Expect(getCoverFromPath(testPath, "")).To(Equal(filepath.Join(testFolder, "Cover.jpeg")))
	})

	It("returns empty string if no match was found", func() {
		conf.Server.CoverArtPriority = "embedded, cover.jpg, front.apng"
		Expect(getCoverFromPath(testPath, "")).To(Equal(""))
	})

	// Reset configuration to default.
	conf.Server.CoverArtPriority = "embedded, cover.*, front.*"
})
