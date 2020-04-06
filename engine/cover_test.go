package engine

import (
	"bytes"
	"image"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cover", func() {
	var cover Cover
	var ds model.DataStore
	ctx := log.NewContext(nil)

	BeforeEach(func() {
		ds = &persistence.MockDataStore{MockedTranscoding: &mockTranscodingRepository{}}
		ds.Album(ctx).(*persistence.MockAlbum).SetData(`[{"id": "222", "CoverArtId": "222"}, {"id": "333", "CoverArtId": ""}]`, 1)
		ds.MediaFile(ctx).(*persistence.MockMediaFile).SetData(`[{"id": "123", "path": "tests/fixtures/test.mp3", "hasCoverArt": true, "updatedAt":"2020-04-02T21:29:31.6377Z"}]`, 1)
		cover = NewCover(ds, testCache)
	})

	It("retrieves the original cover art from an album", func() {
		buf := new(bytes.Buffer)

		Expect(cover.Get(ctx, "222", 0, buf)).To(BeNil())

		_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
		Expect(err).To(BeNil())
		Expect(format).To(Equal("png"))
	})

	It("accepts albumIds with 'al-' prefix", func() {
		buf := new(bytes.Buffer)

		Expect(cover.Get(ctx, "al-222", 0, buf)).To(BeNil())

		_, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		Expect(err).To(BeNil())
	})

	It("returns the default cover if album does not have cover", func() {
		buf := new(bytes.Buffer)

		Expect(cover.Get(ctx, "333", 0, buf)).To(BeNil())

		_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
		Expect(err).To(BeNil())
		Expect(format).To(Equal("png"))
	})

	It("returns the default cover if album is not found", func() {
		buf := new(bytes.Buffer)

		Expect(cover.Get(ctx, "444", 0, buf)).To(BeNil())

		_, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
		Expect(err).To(BeNil())
		Expect(format).To(Equal("png"))
	})

	It("retrieves the original cover art from a media_file", func() {
		buf := new(bytes.Buffer)

		Expect(cover.Get(ctx, "123", 0, buf)).To(BeNil())

		img, format, err := image.Decode(bytes.NewReader(buf.Bytes()))
		Expect(err).To(BeNil())
		Expect(format).To(Equal("jpeg"))
		Expect(img.Bounds().Size().X).To(Equal(600))
		Expect(img.Bounds().Size().Y).To(Equal(600))
	})

	It("resized cover art as requested", func() {
		buf := new(bytes.Buffer)

		Expect(cover.Get(ctx, "123", 200, buf)).To(BeNil())

		img, _, err := image.Decode(bytes.NewReader(buf.Bytes()))
		Expect(err).To(BeNil())
		Expect(img.Bounds().Size().X).To(Equal(200))
		Expect(img.Bounds().Size().Y).To(Equal(200))
	})

})
