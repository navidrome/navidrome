package artwork

import (
	"context"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("radioArtworkReader", func() {
	var (
		tempDir string
		reader  *radioArtworkReader
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		tempDir = GinkgoT().TempDir()
		conf.Server.DataFolder = tempDir

		Expect(os.MkdirAll(filepath.Join(tempDir, "artwork", "radio"), 0755)).To(Succeed())

		reader = &radioArtworkReader{}
	})

	Describe("fromRadioUploadedImage", func() {
		When("radio has an uploaded image", func() {
			It("returns the uploaded image", func() {
				imgPath := filepath.Join(tempDir, "artwork", "radio", "rd-1_test.jpg")
				Expect(os.WriteFile(imgPath, []byte("uploaded radio image"), 0600)).To(Succeed())

				reader.radio = model.Radio{ID: "rd-1", UploadedImage: "rd-1_test.jpg"}
				sf := reader.fromRadioUploadedImage()
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				Expect(path).To(Equal(imgPath))
				r.Close()
			})
		})

		When("radio has no uploaded image", func() {
			It("returns nil reader (falls through)", func() {
				reader.radio = model.Radio{ID: "rd-1"}
				sf := reader.fromRadioUploadedImage()
				r, path, err := sf()
				Expect(err).ToNot(HaveOccurred())
				Expect(r).To(BeNil())
				Expect(path).To(BeEmpty())
			})
		})
	})

	Describe("Reader", func() {
		When("radio has an uploaded image", func() {
			It("returns the image reader", func() {
				imgPath := filepath.Join(tempDir, "artwork", "radio", "rd-1_test.jpg")
				Expect(os.WriteFile(imgPath, []byte("uploaded radio image"), 0600)).To(Succeed())

				reader.radio = model.Radio{ID: "rd-1", UploadedImage: "rd-1_test.jpg"}
				reader.cacheKey.artID = model.ArtworkID{Kind: model.KindRadioArtwork, ID: "rd-1"}
				r, _, err := reader.Reader(context.Background())
				Expect(err).ToNot(HaveOccurred())
				Expect(r).ToNot(BeNil())
				r.Close()
			})
		})

		When("radio has no uploaded image", func() {
			It("returns ErrUnavailable", func() {
				reader.radio = model.Radio{ID: "rd-1"}
				reader.cacheKey.artID = model.ArtworkID{Kind: model.KindRadioArtwork, ID: "rd-1"}
				r, _, err := reader.Reader(context.Background())
				Expect(err).To(MatchError(ErrUnavailable))
				Expect(r).To(BeNil())
			})
		})
	})
})
