package metadata

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tags", func() {
	Describe("getYear", func() {
		It("parses the year correctly", func() {
			var examples = map[string]int{
				"1985":         1985,
				"2002-01":      2002,
				"1969.06":      1969,
				"1980.07.25":   1980,
				"2004-00-00":   2004,
				"2013-May-12":  2013,
				"May 12, 2016": 2016,
				"01/10/1990":   1990,
			}
			for tag, expected := range examples {
				md := &Tags{}
				md.tags = map[string][]string{"date": {tag}}
				Expect(md.Year()).To(Equal(expected))
			}
		})

		It("returns 0 if year is invalid", func() {
			md := &Tags{}
			md.tags = map[string][]string{"date": {"invalid"}}
			Expect(md.Year()).To(Equal(0))
		})
	})

	Describe("getMbzID", func() {
		It("return a valid MBID", func() {
			md := &Tags{}
			md.tags = map[string][]string{
				"musicbrainz_trackid":       {"8f84da07-09a0-477b-b216-cc982dabcde1"},
				"musicbrainz_albumid":       {"f68c985d-f18b-4f4a-b7f0-87837cf3fbf9"},
				"musicbrainz_artistid":      {"89ad4ac3-39f7-470e-963a-56509c546377"},
				"musicbrainz_albumartistid": {"ada7a83c-e3e1-40f1-93f9-3e73dbc9298a"},
			}
			Expect(md.MbzTrackID()).To(Equal("8f84da07-09a0-477b-b216-cc982dabcde1"))
			Expect(md.MbzAlbumID()).To(Equal("f68c985d-f18b-4f4a-b7f0-87837cf3fbf9"))
			Expect(md.MbzArtistID()).To(Equal("89ad4ac3-39f7-470e-963a-56509c546377"))
			Expect(md.MbzAlbumArtistID()).To(Equal("ada7a83c-e3e1-40f1-93f9-3e73dbc9298a"))
		})
		It("return empty string for invalid MBID", func() {
			md := &Tags{}
			md.tags = map[string][]string{
				"musicbrainz_trackid":       {"11406732-6"},
				"musicbrainz_albumid":       {"11406732"},
				"musicbrainz_artistid":      {"200455"},
				"musicbrainz_albumartistid": {"194"},
			}
			Expect(md.MbzTrackID()).To(Equal(""))
			Expect(md.MbzAlbumID()).To(Equal(""))
			Expect(md.MbzArtistID()).To(Equal(""))
			Expect(md.MbzAlbumArtistID()).To(Equal(""))
		})
	})
})
