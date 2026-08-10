package model_test

import (
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtworkID", func() {
	Describe("String()", func() {
		It("returns a bare id when there is no hash", func() {
			id := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: "1234"}
			Expect(id.String()).To(Equal("al-1234"))
		})
		It("appends the hash suffix when set", func() {
			id := model.ArtworkID{Kind: model.KindAlbumArtwork, ID: "1234", Hash: "abcdef0123456789"}
			Expect(id.String()).To(Equal("al-1234_abcdef0123456789"))
		})
		It("never emits a legacy timestamp/_0 suffix", func() {
			id := model.NewArtworkID(model.KindAlbumArtwork, "1234", new(time.Now()))
			Expect(id.String()).To(Equal("al-1234"))
		})
		It("returns empty string for an empty id", func() {
			Expect(model.ArtworkID{Kind: model.KindAlbumArtwork}.String()).To(BeEmpty())
		})
	})

	Describe("NewArtworkID()", func() {
		It("round-trips Kind and ID through String()", func() {
			id := model.NewArtworkID(model.KindAlbumArtwork, "1234", new(time.Now()))
			parsedId, err := model.ParseArtworkID(id.String())
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedId.Kind).To(Equal(id.Kind))
			Expect(parsedId.ID).To(Equal(id.ID))
		})
		It("creates a valid ArtworkID without lastUpdate info", func() {
			id := model.NewArtworkID(model.KindPlaylistArtwork, "1234", nil)
			parsedId, err := model.ParseArtworkID(id.String())
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedId.Kind).To(Equal(id.Kind))
			Expect(parsedId.ID).To(Equal(id.ID))
		})
	})

	Describe("ParseArtworkID - disc kind", func() {
		It("parses a disc artwork ID with dc prefix", func() {
			id := model.NewArtworkID(model.KindDiscArtwork, "albumid123:2", nil)
			parsedId, err := model.ParseArtworkID(id.String())
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedId.Kind).To(Equal(model.KindDiscArtwork))
			Expect(parsedId.ID).To(Equal("albumid123:2"))
		})
	})

	Describe("ParseDiscArtworkID", func() {
		DescribeTable("parses composite disc artwork IDs",
			func(id string, expectedAlbum string, expectedDisc int, expectErr bool) {
				albumID, discNumber, err := model.ParseDiscArtworkID(id)
				if expectErr {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).ToNot(HaveOccurred())
					Expect(albumID).To(Equal(expectedAlbum))
					Expect(discNumber).To(Equal(expectedDisc))
				}
			},
			Entry("valid id", "albumid123:2", "albumid123", 2, false),
			Entry("disc number 1", "abc:1", "abc", 1, false),
			Entry("large disc number", "abc:10", "abc", 10, false),
			Entry("missing colon", "albumid123", "", 0, true),
			Entry("missing disc number", "albumid123:", "", 0, true),
			Entry("non-numeric disc", "albumid123:abc", "", 0, true),
			Entry("empty string", "", "", 0, true),
		)
	})

	Describe("ParseArtworkID()", func() {
		It("parses album artwork ids", func() {
			id, err := model.ParseArtworkID("al-1234")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.Kind).To(Equal(model.KindAlbumArtwork))
			Expect(id.ID).To(Equal("1234"))
			Expect(id.Hash).To(BeEmpty())
		})
		It("parses media file artwork ids", func() {
			id, err := model.ParseArtworkID("mf-a6f8d2b1")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.Kind).To(Equal(model.KindMediaFileArtwork))
			Expect(id.ID).To(Equal("a6f8d2b1"))
		})
		It("parses playlist artwork ids with dashed UUID", func() {
			id, err := model.ParseArtworkID("pl-18690de0-151b-4d86-81cb-f418a907315a")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.Kind).To(Equal(model.KindPlaylistArtwork))
			Expect(id.ID).To(Equal("18690de0-151b-4d86-81cb-f418a907315a"))
		})
		It("captures a 16-hex suffix as Hash", func() {
			id, err := model.ParseArtworkID("al-1234_abcdef0123456789")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.ID).To(Equal("1234"))
			Expect(id.Hash).To(Equal("abcdef0123456789"))
			Expect(id.LastUpdate.IsZero()).To(BeTrue())
		})
		It("captures a high-bit 16-hex suffix as Hash without error", func() {
			id, err := model.ParseArtworkID("al-1234_ffffffffffffffff")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.ID).To(Equal("1234"))
			Expect(id.Hash).To(Equal("ffffffffffffffff"))
		})
		It("parses a legacy hex-timestamp suffix as LastUpdate", func() {
			id, err := model.ParseArtworkID("al-123_688a1b2c")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.ID).To(Equal("123"))
			Expect(id.Hash).To(BeEmpty())
			Expect(id.LastUpdate.Unix()).To(Equal(int64(0x688a1b2c)))
		})
		It("parses a legacy _0 suffix", func() {
			id, err := model.ParseArtworkID("al-123_0")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.ID).To(Equal("123"))
			Expect(id.Hash).To(BeEmpty())
			Expect(id.LastUpdate.IsZero()).To(BeTrue())
		})
		It("silently drops a garbage suffix", func() {
			id, err := model.ParseArtworkID("al-123_zz")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.ID).To(Equal("123"))
			Expect(id.Hash).To(BeEmpty())
		})
		It("fails to parse malformed ids", func() {
			_, err := model.ParseArtworkID("a6f8d2b1")
			Expect(err).To(MatchError("invalid artwork id"))
		})
		It("fails to parse ids with invalid kind", func() {
			_, err := model.ParseArtworkID("xx-a6f8d2b1")
			Expect(err).To(MatchError("invalid artwork kind"))
		})
	})
})
