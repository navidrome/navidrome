package artwork_test

import (
	"context"
	"io"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/auth"
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
		aw = artwork.NewArtwork(ds, cache, ffmpeg)
	})

	Context("Empty ID", func() {
		It("returns placeholder if album is not in the DB", func() {
			r, _, err := aw.Get(context.Background(), "", 0)
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

	Context("Public ID Encoding", func() {
		BeforeEach(func() {
			auth.TokenAuth = jwtauth.New("HS256", []byte("super secret"), nil)
		})
		It("returns a reversible string representation", func() {
			id := model.NewArtworkID(model.KindArtistArtwork, "1234")
			encoded := artwork.EncodeArtworkID(id)
			decoded, err := artwork.DecodeArtworkID(encoded)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).To(Equal(id))
		})
		It("fails to decode an invalid token", func() {
			_, err := artwork.DecodeArtworkID("xx-123")
			Expect(err).To(MatchError("invalid JWT"))
		})
		It("fails to decode an invalid id", func() {
			encoded := artwork.EncodeArtworkID(model.ArtworkID{})
			_, err := artwork.DecodeArtworkID(encoded)
			Expect(err).To(MatchError("invalid artwork id"))
		})
		It("fails to decode a token without an id", func() {
			token, _ := auth.CreatePublicToken(map[string]any{})
			_, err := artwork.DecodeArtworkID(token)
			Expect(err).To(HaveOccurred())
		})
	})
})
