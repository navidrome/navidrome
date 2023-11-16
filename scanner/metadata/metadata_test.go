package metadata_test

import (
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/scanner/metadata"
	_ "github.com/navidrome/navidrome/scanner/metadata/ffmpeg"
	_ "github.com/navidrome/navidrome/scanner/metadata/taglib"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tags", func() {
	Context("Extract", func() {
		BeforeEach(func() {
			conf.Server.Scanner.Extractor = "taglib"
		})

		It("correctly parses metadata from all files in folder", func() {
			mds, err := metadata.Extract("tests/fixtures/test.mp3", "tests/fixtures/test.ogg")
			Expect(err).NotTo(HaveOccurred())
			Expect(mds).To(HaveLen(2))

			m := mds["tests/fixtures/test.mp3"]
			Expect(m.Title()).To(Equal("Song"))
			Expect(m.Album()).To(Equal("Album"))
			Expect(m.Artist()).To(Equal("Artist"))
			Expect(m.AlbumArtist()).To(Equal("Album Artist"))
			Expect(m.Compilation()).To(BeTrue())
			Expect(m.Genres()).To(Equal([]string{"Rock"}))
			y, d := m.Date()
			Expect(y).To(Equal(2014))
			Expect(d).To(Equal("2014-05-21"))
			y, d = m.OriginalDate()
			Expect(y).To(Equal(1996))
			Expect(d).To(Equal("1996-11-21"))
			y, d = m.ReleaseDate()
			Expect(y).To(Equal(2020))
			Expect(d).To(Equal("2020-12-31"))
			n, t := m.TrackNumber()
			Expect(n).To(Equal(2))
			Expect(t).To(Equal(10))
			n, t = m.DiscNumber()
			Expect(n).To(Equal(1))
			Expect(t).To(Equal(2))
			Expect(m.HasPicture()).To(BeTrue())
			Expect(m.Duration()).To(BeNumerically("~", 1.02, 0.01))
			Expect(m.BitRate()).To(Equal(192))
			Expect(m.Channels()).To(Equal(2))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.mp3"))
			Expect(m.Suffix()).To(Equal("mp3"))
			Expect(m.Size()).To(Equal(int64(51876)))
			Expect(m.RGAlbumGain()).To(Equal(3.21518))
			Expect(m.RGAlbumPeak()).To(Equal(0.9125))
			Expect(m.RGTrackGain()).To(Equal(-1.48))
			Expect(m.RGTrackPeak()).To(Equal(0.4512))

			m = mds["tests/fixtures/test.ogg"]
			Expect(err).To(BeNil())
			Expect(m.Title()).To(Equal("Title"))
			Expect(m.HasPicture()).To(BeFalse())
			Expect(m.Duration()).To(BeNumerically("~", 1.04, 0.01))
			Expect(m.Suffix()).To(Equal("ogg"))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.ogg"))
			Expect(m.Size()).To(Equal(int64(6333)))
			// TabLib 1.12 returns 18, previous versions return 39.
			// See https://github.com/taglib/taglib/commit/2f238921824741b2cfe6fbfbfc9701d9827ab06b
			Expect(m.BitRate()).To(BeElementOf(18, 39, 40, 49))
		})
	})
})
