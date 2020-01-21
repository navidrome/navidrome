package scanner

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	It("correctly parses metadata from all files in folder", func() {
		mds, err := ExtractAllMetadata("../tests/fixtures")
		Expect(err).NotTo(HaveOccurred())
		Expect(mds).To(HaveLen(3))

		m := mds["../tests/fixtures/test.mp3"]
		Expect(m.Title()).To(Equal("Song"))
		Expect(m.Album()).To(Equal("Album"))
		Expect(m.Artist()).To(Equal("Artist"))
		Expect(m.AlbumArtist()).To(Equal("Album Artist"))
		Expect(m.Composer()).To(Equal("Composer"))
		Expect(m.Compilation()).To(BeTrue())
		Expect(m.Genre()).To(Equal("Rock"))
		Expect(m.Year()).To(Equal(2014))
		n, t := m.TrackNumber()
		Expect(n).To(Equal(2))
		Expect(t).To(Equal(10))
		n, t = m.DiscNumber()
		Expect(n).To(Equal(1))
		Expect(t).To(Equal(2))
		Expect(m.HasPicture()).To(BeTrue())
		Expect(m.Duration()).To(Equal(1))
		Expect(m.BitRate()).To(Equal(476))
		Expect(m.FilePath()).To(Equal("../tests/fixtures/test.mp3"))
		Expect(m.Suffix()).To(Equal("mp3"))
		Expect(m.Size()).To(Equal(60845))

		m = mds["../tests/fixtures/test.ogg"]
		Expect(err).To(BeNil())
		Expect(m.Title()).To(BeEmpty())
		Expect(m.HasPicture()).To(BeFalse())
		Expect(m.Duration()).To(Equal(3))
		Expect(m.BitRate()).To(Equal(9))
		Expect(m.Suffix()).To(Equal("ogg"))
		Expect(m.FilePath()).To(Equal("../tests/fixtures/test.ogg"))
		Expect(m.Size()).To(Equal(4408))
	})

	It("returns error if path does not exist", func() {
		_, err := ExtractAllMetadata("./INVALID/PATH")
		Expect(err).To(HaveOccurred())
	})

	It("returns empty map if there are no audio files in path", func() {
		Expect(ExtractAllMetadata(".")).To(BeEmpty())
	})

	Context("extractMetadata", func() {
		const outputWithOverlappingTitleTag = `
Input #0, mp3, from 'groovin.mp3':
  Metadata:
    title           : Groovin' (feat. Daniel Sneijers, Susanne Alt)
    artist          : Bone 40
    track           : 1
    album           : Groovin'
    album_artist    : Bone 40
    comment         : Visit http://bone40.bandcamp.com
    date            : 2016
  Duration: 00:03:34.28, start: 0.025056, bitrate: 323 kb/s
    Stream #0:0: Audio: mp3, 44100 Hz, stereo, fltp, 320 kb/s
    Metadata:
      encoder         : LAME3.99r
    Side data:
      replaygain: track gain - -6.000000, track peak - unknown, album gain - unknown, album peak - unknown,
    Stream #0:1: Video: mjpeg, yuvj444p(pc, bt470bg/unknown/unknown), 700x700 [SAR 72:72 DAR 1:1], 90k tbr, 90k tbn, 90k tbc
    Metadata:
      title           : cover
      comment         : Cover (front)
At least one output file must be specified`

		It("parses correct the title without overlapping with the stream tag", func() {
			md, _ := extractMetadata("groovin.mp3", outputWithOverlappingTitleTag)
			Expect(md.Title()).To(Equal("Groovin' (feat. Daniel Sneijers, Susanne Alt)"))
		})
	})
})
