package model_test

import (
	"time"

	. "github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Share", func() {
	Describe("CoverArtID", func() {
		It("uses the loaded album, so the public URL busts on cover edits", func() {
			s := Share{ResourceType: "album", ResourceIDs: "al-1"}
			plain := s.CoverArtID().String()
			Expect(s.CoverArtID().ID).To(Equal("al-1"))

			stamp := time.Now()
			s.Albums = Albums{{ID: "al-1", CoverArtUpdatedAt: &stamp}}
			Expect(s.CoverArtID().ID).To(Equal("al-1"))
			Expect(s.CoverArtID().String()).ToNot(Equal(plain))
		})
		It("finds the shared album even when it is not first in the loaded list", func() {
			stamp := time.Now()
			s := Share{ResourceType: "album", ResourceIDs: "al-2,al-9"}
			s.Albums = Albums{{ID: "al-9"}, {ID: "al-2", CoverArtUpdatedAt: &stamp}}
			id := s.CoverArtID()
			Expect(id.ID).To(Equal("al-2"))
			Expect(id.String()).ToNot(HaveSuffix("_0"))
		})
	})
})
