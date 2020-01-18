package scanner

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metadata", func() {
	FIt("correctly parses mp3 file", func() {
		m, err := ExtractMetadata("../tests/fixtures/test.mp3")
		Expect(err).To(BeNil())
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
	})

	It("correctly parses ogg file with no tags", func() {
		m, err := ExtractMetadata("../tests/fixtures/test.ogg")
		Expect(err).To(BeNil())
		Expect(m.Title()).To(BeEmpty())
		Expect(m.HasPicture()).To(BeFalse())
		Expect(m.Duration()).To(Equal(3))
		Expect(m.BitRate()).To(Equal(9))
		Expect(m.Suffix()).To(Equal("ogg"))
		Expect(m.FilePath()).To(Equal("../tests/fixtures/test.ogg"))
		Expect(m.Size()).To(Equal(4408))
	})

	It("returns error for invalid media file", func() {
		_, err := ExtractMetadata("../tests/fixtures/itunes-library.xml")
		Expect(err).ToNot(BeNil())
	})

	It("returns error for file not found", func() {
		_, err := ExtractMetadata("../tests/fixtures/NOT-FOUND.mp3")
		Expect(err).ToNot(BeNil())
	})
})
