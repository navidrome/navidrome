package metadata

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("taglibExtractor", func() {
	Context("Extract", func() {
		It("correctly parses metadata from all files in folder", func() {
			e := &taglibExtractor{}
			mds, err := e.Extract("tests/fixtures/test.mp3", "tests/fixtures/test.ogg")
			Expect(err).NotTo(HaveOccurred())
			Expect(mds).To(HaveLen(2))

			m := mds["tests/fixtures/test.mp3"]
			Expect(m.Title()).To(Equal("Song"))
			Expect(m.Album()).To(Equal("Album"))
			Expect(m.Artist()).To(Equal("Artist"))
			Expect(m.AlbumArtist()).To(Equal("Album Artist"))
			Expect(m.Composer()).To(Equal("Composer"))
			// TODO This is not working. See https://github.com/taglib/taglib/issues/971
			//Expect(m.Compilation()).To(BeTrue())
			Expect(m.Genre()).To(Equal("Rock"))
			Expect(m.Year()).To(Equal(2014))
			n, t := m.TrackNumber()
			Expect(n).To(Equal(2))
			Expect(t).To(Equal(10))
			n, t = m.DiscNumber()
			Expect(n).To(Equal(1))
			Expect(t).To(Equal(2))
			Expect(m.HasPicture()).To(BeTrue())
			Expect(m.Duration()).To(Equal(float32(1)))
			Expect(m.BitRate()).To(Equal(192))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.mp3"))
			Expect(m.Suffix()).To(Equal("mp3"))
			Expect(m.Size()).To(Equal(int64(60845)))

			m = mds["tests/fixtures/test.ogg"]
			Expect(err).To(BeNil())
			Expect(m.Title()).To(BeEmpty())
			Expect(m.HasPicture()).To(BeFalse())
			Expect(m.Duration()).To(Equal(float32(3)))
			Expect(m.BitRate()).To(Equal(10))
			Expect(m.Suffix()).To(Equal("ogg"))
			Expect(m.FilePath()).To(Equal("tests/fixtures/test.ogg"))
			Expect(m.Size()).To(Equal(int64(4408)))
		})
	})
})
