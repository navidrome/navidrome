package metadata_test

import (
	"encoding/json"
	"os"
	"sort"
	"strings"

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
		DescribeTable("legacyReleaseDate (TaggedLikePicard old behavior)",
			func(recordingDate, originalDate, releaseDate, expected string) {
				mf := toMediaFile(model.RawTags{
					"DATE":         {recordingDate},
					"ORIGINALDATE": {originalDate},
					"RELEASEDATE":  {releaseDate},
				})

				Expect(mf.ReleaseDate).To(Equal(expected))
			},
			Entry("regular mapping", "2020-05-15", "2019-02-10", "2021-01-01", "2021-01-01"),
			Entry("legacy mapping", "2020-05-15", "2019-02-10", "", "2020-05-15"),
			Entry("legacy mapping, originalYear < year", "2018-05-15", "2019-02-10", "2021-01-01", "2021-01-01"),
			Entry("legacy mapping, originalYear empty", "2020-05-15", "", "2021-01-01", "2021-01-01"),
			Entry("legacy mapping, releaseYear", "2020-05-15", "2019-02-10", "2021-01-01", "2021-01-01"),
			Entry("legacy mapping, same dates", "2020-05-15", "2020-05-15", "", "2020-05-15"),
		)
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
					{Value: "This is", Start: new(int64(0))},
					{Value: "English SYLT", Start: new(int64(2500))},
				}, Synced: true},
				{Lang: "xxx", Line: []model.Line{{Value: "Lyrics"}}, Synced: false},
			}
			sort.Slice(actual, func(i, j int) bool { return actual[i].Lang < actual[j].Lang })
			sort.Slice(expected, func(i, j int) bool { return expected[i].Lang < expected[j].Lang })
			Expect(actual).To(Equal(expected))
		})

		It("should parse embedded TTML lyrics before sanitizing XML tags", func() {
			mf = toMediaFile(model.RawTags{
				"LYRICS:ENG": {`<tt xmlns="http://www.w3.org/ns/ttml">
  <body>
    <div>
      <p begin="00:00:01.000" end="00:00:02.500">Embedded TTML line</p>
    </div>
  </body>
</tt>`},
			})
			var actual model.LyricList
			err := json.Unmarshal([]byte(mf.Lyrics), &actual)
			Expect(err).ToNot(HaveOccurred())

			Expect(actual).To(Equal(model.LyricList{
				{
					Kind:   "main",
					Lang:   "eng",
					Line:   []model.Line{{Start: new(int64(1000)), End: new(int64(2500)), Value: "Embedded TTML line"}},
					Synced: true,
				},
			}))
		})

		It("should parse embedded TTML lyrics longer than the metadata tag max length", func() {
			padding := strings.Repeat(`<text for="unused">padding</text>`, 1400)
			content := `<tt xmlns="http://www.w3.org/ns/ttml" xmlns:itunes="http://music.apple.com/lyric-ttml-internal" xml:lang="en">
  <head>
    <metadata>
      <iTunesMetadata xmlns="http://music.apple.com/lyric-ttml-internal">
        <translations>
          <translation xml:lang="en-US">` + padding + `</translation>
        </translations>
      </iTunesMetadata>
    </metadata>
  </head>
  <body>
    <div>
      <p begin="00:00:01.000" end="00:00:02.500" itunes:key="L1">Long embedded TTML line</p>
    </div>
  </body>
</tt>`

			Expect(len(content)).To(BeNumerically(">", 32768))

			mf = toMediaFile(model.RawTags{
				"LYRICS:ENG": {content},
			})
			var actual model.LyricList
			err := json.Unmarshal([]byte(mf.Lyrics), &actual)
			Expect(err).ToNot(HaveOccurred())

			Expect(actual).To(HaveLen(1))
			Expect(actual[0].Kind).To(Equal("main"))
			Expect(actual[0].Lang).To(Equal("en"))
			Expect(actual[0].Line).To(Equal([]model.Line{
				{Start: new(int64(1000)), End: new(int64(2500)), Value: "Long embedded TTML line"},
			}))
		})

		It("should parse embedded SRT lyrics with the tag language", func() {
			mf = toMediaFile(model.RawTags{
				"LYRICS:POR": {`1
00:00:18,800 --> 00:00:22,800
Estamos nas legendas`},
			})
			var actual model.LyricList
			err := json.Unmarshal([]byte(mf.Lyrics), &actual)
			Expect(err).ToNot(HaveOccurred())

			Expect(actual).To(Equal(model.LyricList{
				{
					Lang: "por",
					Line: []model.Line{
						{Start: new(int64(18800)), End: new(int64(22800)), Value: "Estamos nas legendas"},
					},
					Synced: true,
				},
			}))
		})
	})

	Describe("BPM", func() {
		It("maps the BPM tag rounded to the nearest integer", func() {
			mf = toMediaFile(model.RawTags{"BPM": {"120.6"}})
			Expect(mf.BPM).To(Equal(new(121)))
		})
		It("leaves BPM nil when the tag is absent", func() {
			mf = toMediaFile(model.RawTags{})
			Expect(mf.BPM).To(BeNil())
		})
		It("leaves BPM nil when the tag is zero or unparseable", func() {
			Expect(toMediaFile(model.RawTags{"BPM": {"0"}}).BPM).To(BeNil())
			Expect(toMediaFile(model.RawTags{"BPM": {"fast"}}).BPM).To(BeNil())
		})
	})

	Describe("BitDepth", func() {
		It("maps the bit depth when present", func() {
			props.AudioProperties = metadata.AudioProperties{BitDepth: 24}
			mf = toMediaFile(model.RawTags{})
			Expect(mf.BitDepth).To(Equal(new(24)))
		})
		It("leaves BitDepth nil when zero (lossy codecs have no bit depth)", func() {
			props.AudioProperties = metadata.AudioProperties{BitDepth: 0}
			mf = toMediaFile(model.RawTags{})
			Expect(mf.BitDepth).To(BeNil())
		})
	})
})
