package tag_test

import (
	"os"
	"time"

	"github.com/navidrome/navidrome/model/tag"
	"github.com/navidrome/navidrome/utils"
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
		// It is easier to have a real file to test the mod and birth times
		filePath = utils.TempFileName("test", ".mp3")
		f, _ := os.Create(filePath)
		DeferCleanup(func() {
			_ = f.Close()
			_ = os.Remove(filePath)
		})

		fileInfo, _ = os.Stat(filePath)
		props = tag.Properties{
			AudioProperties: tag.AudioProperties{
				Duration: time.Minute * 3,
				BitRate:  320,
			},
			HasPicture: true,
		}
	})

	Describe("Tags", func() {
		Describe("New", func() {
			It("should create a new Tags object with the correct properties", func() {
				props.Tags = map[string][]string{
					"©ART":                                {"First Artist", "Second Artist"},
					"----:com.apple.iTunes:CATALOGNUMBER": {"1234"},
					"tbpm":                                {"120.6"},
					"WM/IsCompilation":                    {"1"},
				}
				tags = tag.New(filePath, fileInfo, props)

				Expect(tags.FilePath()).To(Equal(filePath))
				Expect(tags.ModTime()).To(Equal(fileInfo.ModTime()))
				Expect(tags.BirthTime()).To(BeTemporally("~", tags.ModTime(), time.Second))
				Expect(tags.Size()).To(Equal(fileInfo.Size()))
				Expect(tags.Suffix()).To(Equal("mp3"))
				Expect(tags.AudioProperties()).To(Equal(props.AudioProperties))
				Expect(tags.HasPicture()).To(Equal(props.HasPicture))
				Expect(tags.Strings(tag.TrackArtist)).To(Equal([]string{"First Artist", "Second Artist"}))
				Expect(tags.String(tag.TrackArtist)).To(Equal("First Artist"))
				Expect(tags.Int(tag.CatalogNumber)).To(Equal(int64(1234)))
				Expect(tags.Float(tag.BPM)).To(Equal(120.6))
				Expect(tags.Bool(tag.Compilation)).To(BeTrue())
				Expect(tags.All()).To(SatisfyAll(
					HaveLen(4),
					HaveKeyWithValue(string(tag.TrackArtist), []string{"First Artist", "Second Artist"}),
					HaveKeyWithValue(string(tag.BPM), []string{"120.6"}),
					HaveKeyWithValue(string(tag.Compilation), []string{"1"}),
					HaveKeyWithValue(string(tag.CatalogNumber), []string{"1234"}),
				))

			})

			It("should clean the tags map correctly", func() {
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

				Expect(tags.All()).To(SatisfyAll(
					HaveLen(5),
					Not(HaveKey(unknownTag)),
					HaveKeyWithValue(string(tag.TrackArtist), []string{"Artist Name", "Second Artist"}),
					HaveKeyWithValue(string(tag.Album), []string{"Album Name"}),
					HaveKeyWithValue(string(tag.ReleaseDate), []string{"2022"}),
					HaveKeyWithValue(string(tag.Genre), []string{"Pop", "Rock"}),
					HaveKeyWithValue(string(tag.TrackNumber), []string{"1/10"}),
				))
			})
		})

		DescribeTable("Date",
			func(value string, expectedYear int, expectedDate string) {
				props.Tags = map[string][]string{
					"date": {value},
				}
				tags = tag.New(filePath, fileInfo, props)

				testDate := tags.Date(tag.ReleaseDate)
				Expect(string(testDate)).To(Equal(expectedDate))
				Expect(testDate.Year()).To(Equal(expectedYear))
			},
			Entry(nil, "1985", 1985, "1985"),
			Entry(nil, "2002-01", 2002, "2002-01"),
			Entry(nil, "1969.06", 1969, "1969"),
			Entry(nil, "1980.07.25", 1980, "1980"),
			Entry(nil, "2004-00-00", 2004, "2004"),
			Entry(nil, "2016-12-31", 2016, "2016-12-31"),
			Entry(nil, "2016-12-31 12:15", 2016, "2016-12-31"),
			Entry(nil, "2013-May-12", 2013, "2013"),
			Entry(nil, "May 12, 2016", 2016, "2016"),
			Entry(nil, "01/10/1990", 1990, "1990"),
			Entry(nil, "invalid", 0, ""),
		)

		DescribeTable("NumAndTotal",
			func(num, total string, expectedNum int, expectedTotal int) {
				props.Tags = map[string][]string{
					"Track":      {num},
					"TrackTotal": {total},
				}
				tags = tag.New(filePath, fileInfo, props)

				testNum, testTotal := tags.NumAndTotal(tag.TrackNumber)
				Expect(testNum).To(Equal(expectedNum))
				Expect(testTotal).To(Equal(expectedTotal))
			},
			Entry(nil, "2", "", 2, 0),
			Entry(nil, "2", "10", 2, 10),
			Entry(nil, "2/10", "", 2, 10),
			Entry(nil, "", "", 0, 0),
			Entry(nil, "A", "", 0, 0),
		)
	})
})
