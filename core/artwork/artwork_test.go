package artwork_test

import (
	"context"
	"io"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/resources"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Artwork", func() {
	var aw artwork.Artwork
	var ds model.DataStore
	var ffmpeg *tests.MockFFmpeg

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.ImageCacheSize = "0" // Disable cache
		cache := artwork.GetImageCache()
		ffmpeg = tests.NewMockFFmpeg("content from ffmpeg")
		aw = artwork.NewArtwork(ds, cache, ffmpeg, nil)
	})

	Context("GetOrPlaceholder", func() {
		Context("Empty ID", func() {
			It("returns placeholder if album is not in the DB", func() {
				r, _, err := aw.GetOrPlaceholder(context.Background(), "", 0, false)
				Expect(err).ToNot(HaveOccurred())

				ph, err := resources.FS().Open(consts.PlaceholderAlbumArt)
				Expect(err).ToNot(HaveOccurred())
				phBytes, err := io.ReadAll(ph)
				Expect(err).ToNot(HaveOccurred())

				result, err := io.ReadAll(r)
				Expect(err).ToNot(HaveOccurred())

				Expect(result).To(Equal(phBytes))
			})
		})
	})
	Context("Get", func() {
		Context("Empty ID", func() {
			It("returns an ErrUnavailable error", func() {
				_, _, err := aw.Get(context.Background(), model.ArtworkID{}, 0, false)
				Expect(err).To(MatchError(artwork.ErrUnavailable))
			})
		})
	})
})
