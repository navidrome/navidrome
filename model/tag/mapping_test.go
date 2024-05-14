package tag_test

import (
	"os"
	"time"

	"github.com/navidrome/navidrome/model/tag"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tags", func() {
	var (
		filePath string
		fileInfo os.FileInfo
		props    tag.Properties
		tags     tag.Tags
	)

	BeforeEach(func() {
		filePath = "tests/fixtures/test.mp3"
		fileInfo, _ = os.Stat(filePath)
		props = tag.Properties{
			AudioProperties: tag.AudioProperties{
				Duration: time.Minute * 3,
				BitRate:  320,
			},
			HasPicture: true,
		}
	})

	Describe("New", func() {
		It("should create a new Tags object with the correct properties", func() {
			props.Tags = map[string][]string{
				"Artist":                             {"Artist Name"},
				"Track":                              {"1/10"},
				"----:com.apple.iTunes:originaldate": {"2016-02-23"},
				"tbpm":                               {"120.6"},
				"WM/IsCompilation":                   {"1"},
			}
			tags = tag.New(filePath, fileInfo, props)

			Expect(tags.FilePath()).To(Equal(filePath))
			Expect(tags.ModTime()).To(Equal(fileInfo.ModTime()))
			Expect(tags.Size()).To(Equal(fileInfo.Size()))
			Expect(tags.Suffix()).To(Equal("mp3"))
			Expect(tags.AudioProperties()).To(Equal(props.AudioProperties))
			Expect(tags.HasPicture()).To(Equal(props.HasPicture))
			Expect(tags.Strings(tag.TrackArtist)).To(Equal([]string{"Artist Name"}))
			Expect(tags.String(tag.TrackArtist)).To(Equal("Artist Name"))
			Expect(tags.All()).To(HaveKeyWithValue(string(tag.TrackArtist), []string{"Artist Name"}))
			Expect(tags.Date(tag.OriginalDate)).To(Equal(tag.Date("2016-02-23")))
			Expect(tags.Date(tag.OriginalDate).Year()).To(Equal(2016))
			Expect(tags.Float(tag.BPM)).To(Equal(120.6))
			Expect(tags.Bool(tag.Compilation)).To(BeTrue())
			num, total := tags.NumAndTotal(tag.TrackNumber)
			Expect(num).To(Equal(1))
			Expect(total).To(Equal(10))
		})

		FIt("should clean the tags map correctly", func() {
			const unknownTag = "UNKNOWN_TAG"
			props.Tags = map[string][]string{
				"TPE1":          {"Artist Name", "Artist Name", ""},
				"©ART":          {"Second Artist"},
				"CatalogNumber": {""},
				"Album":         {"Album Name", "", "Album Name"},
				"Year":          {"2022", "2022", ""},
				"Genre":         {"Pop", "", "Pop", "Rock"},
				"Track":         {"1/10", "1/10", ""},
				unknownTag:      {"value"},
			}
			tags = tag.New(filePath, fileInfo, props)

			cleanedTags := tags.All()

			Expect(cleanedTags).To(HaveLen(5))
			Expect(cleanedTags).ToNot(HaveKey(unknownTag))
			Expect(cleanedTags[string(tag.TrackArtist)]).To(Equal([]string{"Artist Name", "Second Artist"}))
			Expect(cleanedTags[string(tag.Album)]).To(ConsistOf("Album Name"))
			Expect(cleanedTags[string(tag.ReleaseDate)]).To(ConsistOf("2022"))
			Expect(cleanedTags[string(tag.Genre)]).To(ConsistOf("Pop", "Rock"))
			Expect(cleanedTags[string(tag.TrackNumber)]).To(ConsistOf("1/10"))
			Expect(cleanedTags[string(tag.TrackNumber)]).To(ConsistOf("1/10"))
		})
	})
})
