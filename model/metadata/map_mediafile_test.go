package metadata_test

import (
	"os"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ToMediaFile", func() {
	var (
		props metadata.Info
		md    metadata.Metadata
		mf    model.MediaFile
	)

	BeforeEach(func() {
		_, filePath, _ := tests.TempFile(GinkgoT(), "test", ".mp3")
		fileInfo, _ := os.Stat(filePath)
		props = metadata.Info{
			FileInfo: testFileInfo{fileInfo},
		}
	})

	var toMediaFile = func(tags map[string][]string) model.MediaFile {
		props.Tags = tags
		md = metadata.New("filepath", props)
		return md.ToMediaFile()
	}

	Describe("Dates", func() {
		It("should parse the dates like Picard", func() {
			mf = toMediaFile(map[string][]string{
				"ORIGINALDATE": {"1978-09-10"},
				"DATE":         {"1977-03-04"},
				"RELEASEDATE":  {"2002-01-02"},
			})

			Expect(mf.Year).To(Equal(1977))
			Expect(mf.Date).To(Equal("1977-03-04"))
			Expect(mf.OriginalYear).To(Equal(1978))
			Expect(mf.OriginalDate).To(Equal("1978-09-10"))
			Expect(mf.ReleaseYear).To(Equal(2002))
			Expect(mf.ReleaseDate).To(Equal("2002-01-02"))
		})
	})
})
