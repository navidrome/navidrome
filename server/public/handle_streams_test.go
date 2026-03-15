package public

import (
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sanitizeName", func() {
	It("leaves a plain string unchanged", func() {
		Expect(sanitizeName("Artist Name")).To(Equal("Artist Name"))
	})

	It("replaces a single slash with an underscore", func() {
		Expect(sanitizeName("AC/DC")).To(Equal("AC_DC"))
	})

	It("replaces multiple slashes with underscores", func() {
		Expect(sanitizeName("a/b/c")).To(Equal("a_b_c"))
	})
})

var _ = Describe("downloadFilename", func() {
	var mf *model.MediaFile

	BeforeEach(func() {
		mf = &model.MediaFile{
			Artist:    "The Beatles",
			Title:     "Hey Jude",
			Suffix:    "flac",
			UpdatedAt: time.Now(),
		}
	})

	It("uses the media file suffix when format is empty", func() {
		Expect(downloadFilename(mf, "")).To(Equal("The Beatles - Hey Jude.flac"))
	})

	It("uses the media file suffix when format is 'raw'", func() {
		Expect(downloadFilename(mf, "raw")).To(Equal("The Beatles - Hey Jude.flac"))
	})

	It("uses the format as the extension when a real format is given", func() {
		Expect(downloadFilename(mf, "mp3")).To(Equal("The Beatles - Hey Jude.mp3"))
	})

	It("sanitizes slashes in the artist name", func() {
		mf.Artist = "AC/DC"
		Expect(downloadFilename(mf, "")).To(Equal("AC_DC - Hey Jude.flac"))
	})

	It("sanitizes slashes in the title", func() {
		mf.Title = "Love/Hate"
		Expect(downloadFilename(mf, "")).To(Equal("The Beatles - Love_Hate.flac"))
	})
})
