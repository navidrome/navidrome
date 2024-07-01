package metadata_test

import (
	"os"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	var (
		filePath string
		fileInfo os.FileInfo
		props    metadata.Info
		md       metadata.Metadata
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
		props = metadata.Info{
			AudioProperties: metadata.AudioProperties{
				Duration: time.Minute * 3,
				BitRate:  320,
			},
			HasPicture: true,
			FileInfo:   testFileInfo{fileInfo},
		}
	})

	Describe("Metadata", func() {
		Describe("New", func() {
			It("should create a new Metadata object with the correct properties", func() {
				props.Tags = map[string][]string{
					"©ART":                                {"First Artist", "Second Artist"},
					"----:com.apple.iTunes:CATALOGNUMBER": {"1234"},
					"tbpm":                                {"120.6"},
					"WM/IsCompilation":                    {"1"},
				}
				md = metadata.New(filePath, props)

				Expect(md.FilePath()).To(Equal(filePath))
				Expect(md.ModTime()).To(Equal(fileInfo.ModTime()))
				Expect(md.BirthTime()).To(BeTemporally("~", md.ModTime(), time.Second))
				Expect(md.Size()).To(Equal(fileInfo.Size()))
				Expect(md.Suffix()).To(Equal("mp3"))
				Expect(md.AudioProperties()).To(Equal(props.AudioProperties))
				Expect(md.Length()).To(Equal(float32(3 * 60)))
				Expect(md.HasPicture()).To(Equal(props.HasPicture))
				Expect(md.Strings(model.TagTrackArtist)).To(Equal([]string{"First Artist", "Second Artist"}))
				Expect(md.String(model.TagTrackArtist)).To(Equal("First Artist"))
				Expect(md.Int(model.TagCatalogNumber)).To(Equal(int64(1234)))
				Expect(md.Float(model.TagBPM)).To(Equal(120.6))
				Expect(md.Bool(model.TagCompilation)).To(BeTrue())
				Expect(md.All()).To(SatisfyAll(
					HaveLen(4),
					HaveKeyWithValue(model.TagTrackArtist, []string{"First Artist", "Second Artist"}),
					HaveKeyWithValue(model.TagBPM, []string{"120.6"}),
					HaveKeyWithValue(model.TagCompilation, []string{"1"}),
					HaveKeyWithValue(model.TagCatalogNumber, []string{"1234"}),
				))

			})

			It("should clean the tags map correctly", func() {
				const unknownTag = "UNKNOWN_TAG"
				props.Tags = map[string][]string{
					"TPE1":          {"Artist Name", "Artist Name", ""},
					"©ART":          {"Second Artist"},
					"CatalogNumber": {""},
					"Album":         {"Album Name", "", "Album Name"},
					"Date":          {"2022-10-02 12:15:01"},
					"Year":          {"2022", "2022", ""},
					"Genre":         {"Pop", "", "Pop", "Rock"},
					"Track":         {"1/10", "1/10", ""},
					unknownTag:      {"value"},
				}
				md = metadata.New(filePath, props)

				Expect(md.All()).To(SatisfyAll(
					HaveLen(5),
					Not(HaveKey(unknownTag)),
					HaveKeyWithValue(model.TagTrackArtist, []string{"Artist Name", "Second Artist"}),
					HaveKeyWithValue(model.TagAlbum, []string{"Album Name"}),
					HaveKeyWithValue(model.TagRecordingDate, []string{"2022-10-02", "2022"}),
					HaveKeyWithValue(model.TagGenre, []string{"Pop", "Rock"}),
					HaveKeyWithValue(model.TagTrackNumber, []string{"1/10"}),
				))
			})

			It("should truncate long strings", func() {
				props.Tags = map[string][]string{
					"Title": {strings.Repeat("a", 2048)},
				}
				md = metadata.New(filePath, props)

				Expect(md.String(model.TagTitle)).To(HaveLen(1024))
			})
		})

		DescribeTable("Date",
			func(value string, expectedYear int, expectedDate string) {
				props.Tags = map[string][]string{
					"date": {value},
				}
				md = metadata.New(filePath, props)

				testDate := md.Date(model.TagRecordingDate)
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
				md = metadata.New(filePath, props)

				testNum, testTotal := md.NumAndTotal(model.TagTrackNumber)
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
