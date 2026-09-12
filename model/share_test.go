package model_test

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Share.CoverArtID", func() {
	It("returns an empty artwork ID for a media file share with no visible tracks", func() {
		s := model.Share{ResourceType: "media_file", ResourceIDs: "mf-1"}
		Expect(s.CoverArtID()).To(Equal(model.ArtworkID{}))
	})

	It("picks a track's cover for a media file share", func() {
		s := model.Share{ResourceType: "media_file", ResourceIDs: "mf-1", Tracks: model.MediaFiles{{ID: "mf-1"}}}
		Expect(s.CoverArtID()).To(Equal(model.MediaFile{ID: "mf-1"}.CoverArtID()))
	})
})
