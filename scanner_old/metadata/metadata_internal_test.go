package metadata

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tags", func() {
	DescribeTable("getDate",
		func(tag string, expectedYear int, expectedDate string) {
			md := &Tags{}
			md.Tags = map[string][]string{"date": {tag}}
			testYear, testDate := md.Date()
			Expect(testYear).To(Equal(expectedYear))
			Expect(testDate).To(Equal(expectedDate))
		},
		Entry(nil, "1985", 1985, "1985"),
		Entry(nil, "2002-01", 2002, "2002-01"),
		Entry(nil, "1969.06", 1969, "1969"),
		Entry(nil, "1980.07.25", 1980, "1980"),
		Entry(nil, "2004-00-00", 2004, "2004"),
		Entry(nil, "2016-12-31", 2016, "2016-12-31"),
		Entry(nil, "2013-May-12", 2013, "2013"),
		Entry(nil, "May 12, 2016", 2016, "2016"),
		Entry(nil, "01/10/1990", 1990, "1990"),
		Entry(nil, "invalid", 0, ""),
	)

	Describe("getMbzID", func() {
		It("return a valid MBID", func() {
			md := &Tags{}
			md.Tags = map[string][]string{
				"musicbrainz_trackid":        {"8f84da07-09a0-477b-b216-cc982dabcde1"},
				"musicbrainz_releasetrackid": {"6caf16d3-0b20-3fe6-8020-52e31831bc11"},
				"musicbrainz_albumid":        {"f68c985d-f18b-4f4a-b7f0-87837cf3fbf9"},
				"musicbrainz_artistid":       {"89ad4ac3-39f7-470e-963a-56509c546377"},
				"musicbrainz_albumartistid":  {"ada7a83c-e3e1-40f1-93f9-3e73dbc9298a"},
			}
			Expect(md.MbzRecordingID()).To(Equal("8f84da07-09a0-477b-b216-cc982dabcde1"))
			Expect(md.MbzReleaseTrackID()).To(Equal("6caf16d3-0b20-3fe6-8020-52e31831bc11"))
			Expect(md.MbzAlbumID()).To(Equal("f68c985d-f18b-4f4a-b7f0-87837cf3fbf9"))
			Expect(md.MbzArtistID()).To(Equal("89ad4ac3-39f7-470e-963a-56509c546377"))
			Expect(md.MbzAlbumArtistID()).To(Equal("ada7a83c-e3e1-40f1-93f9-3e73dbc9298a"))
		})
		It("return empty string for invalid MBID", func() {
			md := &Tags{}
			md.Tags = map[string][]string{
				"musicbrainz_trackid":       {"11406732-6"},
				"musicbrainz_albumid":       {"11406732"},
				"musicbrainz_artistid":      {"200455"},
				"musicbrainz_albumartistid": {"194"},
			}
			Expect(md.MbzRecordingID()).To(Equal(""))
			Expect(md.MbzAlbumID()).To(Equal(""))
			Expect(md.MbzArtistID()).To(Equal(""))
			Expect(md.MbzAlbumArtistID()).To(Equal(""))
		})
	})

	Describe("getAllTagValues", func() {
		It("returns values from all tag names", func() {
			md := &Tags{}
			md.Tags = map[string][]string{
				"genre": {"Rock", "Pop", "New Wave"},
			}

			Expect(md.Genres()).To(ConsistOf("Rock", "Pop", "New Wave"))
		})
	})

	Describe("removeDuplicatesAndEmpty", func() {
		It("removes duplicates", func() {
			md := NewTag("/music/artist/album01/Song.mp3", nil, ParsedTags{
				"genre": []string{"pop", "rock", "pop"},
				"date":  []string{"2023-03-01", "2023-03-01"},
				"mood":  []string{"happy", "sad"},
			})
			Expect(md.Tags).To(HaveKeyWithValue("genre", []string{"pop", "rock"}))
			Expect(md.Tags).To(HaveKeyWithValue("date", []string{"2023-03-01"}))
			Expect(md.Tags).To(HaveKeyWithValue("mood", []string{"happy", "sad"}))
		})
		It("removes empty tags", func() {
			md := NewTag("/music/artist/album01/Song.mp3", nil, ParsedTags{
				"genre": []string{"pop", "rock", "pop"},
				"mood":  []string{"", ""},
			})
			Expect(md.Tags).To(HaveKeyWithValue("genre", []string{"pop", "rock"}))
			Expect(md.Tags).ToNot(HaveKey("mood"))
		})
	})

	Describe("Bpm", func() {
		var t *Tags
		BeforeEach(func() {
			t = &Tags{Tags: map[string][]string{
				"fbpm": []string{"141.7"},
			}}
		})

		It("rounds a floating point fBPM tag", func() {
			Expect(t.Bpm()).To(Equal(142))
		})
	})

	Describe("ReplayGain", func() {
		DescribeTable("getGainValue",
			func(tag string, expected float64) {
				md := &Tags{}
				md.Tags = map[string][]string{"replaygain_track_gain": {tag}}
				Expect(md.RGTrackGain()).To(Equal(expected))

			},
			Entry("0", "0", 0.0),
			Entry("1.2dB", "1.2dB", 1.2),
			Entry("Infinity", "Infinity", 0.0),
			Entry("Invalid value", "INVALID VALUE", 0.0),
		)
		DescribeTable("getPeakValue",
			func(tag string, expected float64) {
				md := &Tags{}
				md.Tags = map[string][]string{"replaygain_track_peak": {tag}}
				Expect(md.RGTrackPeak()).To(Equal(expected))

			},
			Entry("0", "0", 0.0),
			Entry("0.5", "0.5", 0.5),
			Entry("Invalid dB suffix", "0.7dB", 1.0),
			Entry("Infinity", "Infinity", 1.0),
			Entry("Invalid value", "INVALID VALUE", 1.0),
		)
		DescribeTable("getR128GainValue",
			func(tag string, expected float64) {
				md := &Tags{}
				md.Tags = map[string][]string{"r128_track_gain": {tag}}
				Expect(md.RGTrackGain()).To(Equal(expected))

			},
			Entry("0", "0", 5.0),
			Entry("-3776", "-3776", -9.75),
			Entry("Infinity", "Infinity", 0.0),
			Entry("Invalid value", "INVALID VALUE", 0.0),
		)
	})
})
