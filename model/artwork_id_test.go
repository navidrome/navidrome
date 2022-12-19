package model_test

import (
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseArtworkID()", func() {
	It("parses album artwork ids", func() {
		id, err := model.ParseArtworkID("al-1234-ff")
		Expect(err).ToNot(HaveOccurred())
		Expect(id.Kind).To(Equal(model.KindAlbumArtwork))
		Expect(id.ID).To(Equal("1234"))
		Expect(id.LastAccess).To(Equal(time.Unix(255, 0)))
	})
	It("parses media file artwork ids", func() {
		id, err := model.ParseArtworkID("mf-a6f8d2b1-ffff")
		Expect(err).ToNot(HaveOccurred())
		Expect(id.Kind).To(Equal(model.KindMediaFileArtwork))
		Expect(id.ID).To(Equal("a6f8d2b1"))
		Expect(id.LastAccess).To(Equal(time.Unix(65535, 0)))
	})
	It("fails to parse malformed ids", func() {
		_, err := model.ParseArtworkID("a6f8d2b1")
		Expect(err).To(MatchError("invalid artwork id"))
	})
	It("fails to parse ids with invalid kind", func() {
		_, err := model.ParseArtworkID("xx-a6f8d2b1-ff")
		Expect(err).To(MatchError("invalid artwork kind"))
	})
})
