package model_test

import (
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtworkID", func() {
	Describe("NewArtworkID()", func() {
		It("creates a valid parseable ArtworkID", func() {
			now := time.Now()
			id := model.NewArtworkID(model.KindAlbumArtwork, "1234", &now)
			parsedId, err := model.ParseArtworkID(id.String())
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedId.Kind).To(Equal(id.Kind))
			Expect(parsedId.ID).To(Equal(id.ID))
			Expect(parsedId.LastUpdate.Unix()).To(Equal(id.LastUpdate.Unix()))
		})
		It("creates a valid ArtworkID without lastUpdate info", func() {
			id := model.NewArtworkID(model.KindPlaylistArtwork, "1234", nil)
			parsedId, err := model.ParseArtworkID(id.String())
			Expect(err).ToNot(HaveOccurred())
			Expect(parsedId.Kind).To(Equal(id.Kind))
			Expect(parsedId.ID).To(Equal(id.ID))
			Expect(parsedId.LastUpdate.Unix()).To(Equal(id.LastUpdate.Unix()))
		})
	})
	Describe("ParseArtworkID()", func() {
		It("parses album artwork ids", func() {
			id, err := model.ParseArtworkID("al-1234")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.Kind).To(Equal(model.KindAlbumArtwork))
			Expect(id.ID).To(Equal("1234"))
		})
		It("parses media file artwork ids", func() {
			id, err := model.ParseArtworkID("mf-a6f8d2b1")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.Kind).To(Equal(model.KindMediaFileArtwork))
			Expect(id.ID).To(Equal("a6f8d2b1"))
		})
		It("parses playlists artwork ids", func() {
			id, err := model.ParseArtworkID("pl-18690de0-151b-4d86-81cb-f418a907315a")
			Expect(err).ToNot(HaveOccurred())
			Expect(id.Kind).To(Equal(model.KindPlaylistArtwork))
			Expect(id.ID).To(Equal("18690de0-151b-4d86-81cb-f418a907315a"))
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
