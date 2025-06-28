package metadata_test

import (
	"os"
	"strings"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/gg"
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
				props.Tags = model.RawTags{
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
				props.Tags = model.RawTags{
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
					Not(HaveKey(unknownTag)),
					HaveKeyWithValue(model.TagTrackArtist, []string{"Artist Name", "Second Artist"}),
					HaveKeyWithValue(model.TagAlbum, []string{"Album Name"}),
					HaveKeyWithValue(model.TagRecordingDate, []string{"2022-10-02"}),
					HaveKeyWithValue(model.TagReleaseDate, []string{"2022"}),
					HaveKeyWithValue(model.TagGenre, []string{"Pop", "Rock"}),
					HaveKeyWithValue(model.TagTrackNumber, []string{"1/10"}),
					HaveLen(6),
				))
			})

			It("should truncate long strings", func() {
				props.Tags = model.RawTags{
					"Title":      {strings.Repeat("a", 2048)},
					"Comment":    {strings.Repeat("a", 8192)},
					"lyrics:xxx": {strings.Repeat("a", 60000)},
				}
				md = metadata.New(filePath, props)

				Expect(md.String(model.TagTitle)).To(HaveLen(1024))
				Expect(md.String(model.TagComment)).To(HaveLen(4096))
				pair := md.Pairs(model.TagLyrics)

				Expect(pair).To(HaveLen(1))
				Expect(pair[0].Key()).To(Equal("xxx"))

				// Note: a total of 6 characters are lost from maxLength from
				// the key portion and separator
				Expect(pair[0].Value()).To(HaveLen(32762))
			})

			It("should split multiple values", func() {
				props.Tags = model.RawTags{
					"Genre": {"Rock/Pop;;Punk"},
				}
				md = metadata.New(filePath, props)

				Expect(md.Strings(model.TagGenre)).To(Equal([]string{"Rock", "Pop", "Punk"}))
			})
		})

		DescribeTable("Date",
			func(value string, expectedYear int, expectedDate string) {
				props.Tags = model.RawTags{
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
				props.Tags = model.RawTags{
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

		Describe("Performers", func() {
			Describe("ID3", func() {
				BeforeEach(func() {
					props.Tags = model.RawTags{
						"PERFORMER:GUITAR":            {"Guitarist 1", "Guitarist 2"},
						"PERFORMER:BACKGROUND VOCALS": {"Backing Singer"},
						"PERFORMER:PERFORMER":         {"Wonderlove", "Lovewonder"},
					}
				})

				It("should return the performers", func() {
					md = metadata.New(filePath, props)

					Expect(md.All()).To(HaveKey(model.TagPerformer))
					Expect(md.Strings(model.TagPerformer)).To(ConsistOf(
						metadata.NewPair("guitar", "Guitarist 1"),
						metadata.NewPair("guitar", "Guitarist 2"),
						metadata.NewPair("background vocals", "Backing Singer"),
						metadata.NewPair("", "Wonderlove"),
						metadata.NewPair("", "Lovewonder"),
					))
				})
			})

			Describe("Vorbis", func() {
				BeforeEach(func() {
					props.Tags = model.RawTags{
						"PERFORMER": {
							"John Adams (Rhodes piano)",
							"Vincent Henry (alto saxophone, baritone saxophone and tenor saxophone)",
							"Salaam Remi (drums (drum set) and organ)",
							"Amy Winehouse (guitar)",
							"Amy Winehouse (vocals)",
							"Wonderlove",
						},
					}
				})

				It("should return the performers", func() {
					md = metadata.New(filePath, props)

					Expect(md.All()).To(HaveKey(model.TagPerformer))
					Expect(md.Strings(model.TagPerformer)).To(ConsistOf(
						metadata.NewPair("rhodes piano", "John Adams"),
						metadata.NewPair("alto saxophone, baritone saxophone and tenor saxophone", "Vincent Henry"),
						metadata.NewPair("drums (drum set) and organ", "Salaam Remi"),
						metadata.NewPair("guitar", "Amy Winehouse"),
						metadata.NewPair("vocals", "Amy Winehouse"),
						metadata.NewPair("", "Wonderlove"),
					))
				})
			})
		})

		Describe("Lyrics", func() {
			BeforeEach(func() {
				props.Tags = model.RawTags{
					"LYRICS:POR": {"Letras"},
					"LYRICS:ENG": {"Lyrics"},
				}
			})

			It("should return the lyrics", func() {
				md = metadata.New(filePath, props)

				Expect(md.All()).To(HaveKey(model.TagLyrics))
				Expect(md.Strings(model.TagLyrics)).To(ContainElements(
					metadata.NewPair("por", "Letras"),
					metadata.NewPair("eng", "Lyrics"),
				))
			})
		})

		Describe("ReplayGain", func() {
			createMF := func(tag, tagValue string) model.MediaFile {
				props.Tags = model.RawTags{
					tag: {tagValue},
				}
				md = metadata.New(filePath, props)
				return md.ToMediaFile(0, "0")
			}

			DescribeTable("Gain",
				func(tagValue string, expected *float64) {
					mf := createMF("replaygain_track_gain", tagValue)
					Expect(mf.RGTrackGain).To(Equal(expected))
				},
				Entry("0", "0", gg.P(0.0)),
				Entry("1.2dB", "1.2dB", gg.P(1.2)),
				Entry("Infinity", "Infinity", nil),
				Entry("Invalid value", "INVALID VALUE", nil),
				Entry("NaN", "NaN", nil),
			)
			DescribeTable("Peak",
				func(tagValue string, expected *float64) {
					mf := createMF("replaygain_track_peak", tagValue)
					Expect(mf.RGTrackPeak).To(Equal(expected))
				},
				Entry("0", "0", gg.P(0.0)),
				Entry("1.0", "1.0", gg.P(1.0)),
				Entry("0.5", "0.5", gg.P(0.5)),
				Entry("Invalid dB suffix", "0.7dB", nil),
				Entry("Infinity", "Infinity", nil),
				Entry("Invalid value", "INVALID VALUE", nil),
				Entry("NaN", "NaN", nil),
			)
			DescribeTable("getR128GainValue",
				func(tagValue string, expected *float64) {
					mf := createMF("r128_track_gain", tagValue)
					Expect(mf.RGTrackGain).To(Equal(expected))

				},
				Entry("0", "0", gg.P(5.0)),
				Entry("-3776", "-3776", gg.P(-9.75)),
				Entry("Infinity", "Infinity", nil),
				Entry("Invalid value", "INVALID VALUE", nil),
			)
		})

	})
})
