package artwork

import (
	"time"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaFile Artwork Reader", func() {
	Describe("Key", func() {
		It("changes when the album's cover stamp changes", func() {
			r := &mediafileArtworkReader{}
			r.album = model.Album{ID: "al-1"}
			before := r.Key()
			stamp := time.Now()
			r.album.CoverArtUpdatedAt = &stamp
			Expect(r.Key()).ToNot(Equal(before))
		})
	})
})
