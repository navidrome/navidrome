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
})
