package artwork

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("radioArtworkReader", func() {
	Describe("fromRadioUploadedImage", func() {
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
})
