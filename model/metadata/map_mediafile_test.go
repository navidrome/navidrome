package metadata_test

import (
	"encoding/json"
	"os"
	"sort"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/tests"
	. "github.com/navidrome/navidrome/utils/gg"
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

	var toMediaFile = func(tags model.RawTags) model.MediaFile {
		props.Tags = tags
		md = metadata.New("filepath", props)
		return md.ToMediaFile(1, "folderID")
	}

	Describe("Dates", func() {
		It("should parse properly tagged dates ", func() {
			mf = toMediaFile(model.RawTags{
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

		It("should parse dates with only year", func() {
			mf = toMediaFile(model.RawTags{
				"ORIGINALYEAR": {"1978"},
				"DATE":         {"1977"},
				"RELEASEDATE":  {"2002"},
			})

			Expect(mf.Year).To(Equal(1977))
			Expect(mf.Date).To(Equal("1977"))
			Expect(mf.OriginalYear).To(Equal(1978))
			Expect(mf.OriginalDate).To(Equal("1978"))
			Expect(mf.ReleaseYear).To(Equal(2002))
			Expect(mf.ReleaseDate).To(Equal("2002"))
		})

		It("should parse dates tagged the legacy way (no release date)", func() {
			mf = toMediaFile(model.RawTags{
				"DATE":         {"2014"},
				"ORIGINALDATE": {"1966"},
			})

			Expect(mf.Year).To(Equal(1966))
			Expect(mf.OriginalYear).To(Equal(1966))
			Expect(mf.ReleaseYear).To(Equal(2014))
		})
	})

	Describe("Lyrics", func() {
		It("should parse the lyrics", func() {
			mf = toMediaFile(model.RawTags{
				"LYRICS:XXX": {"Lyrics"},
				"LYRICS:ENG": {
					"[00:00.00]This is\n[00:02.50]English SYLT\n",
				},
			})
			var actual model.LyricList
			err := json.Unmarshal([]byte(mf.Lyrics), &actual)
			Expect(err).ToNot(HaveOccurred())

			expected := model.LyricList{
				{Lang: "eng", Line: []model.Line{
					{Value: "This is", Start: P(int64(0))},
					{Value: "English SYLT", Start: P(int64(2500))},
				}, Synced: true},
				{Lang: "xxx", Line: []model.Line{{Value: "Lyrics"}}, Synced: false},
			}
			sort.Slice(actual, func(i, j int) bool { return actual[i].Lang < actual[j].Lang })
			sort.Slice(expected, func(i, j int) bool { return expected[i].Lang < expected[j].Lang })
			Expect(actual).To(Equal(expected))
		})
	})
})
