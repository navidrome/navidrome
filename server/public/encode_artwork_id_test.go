package public_test

import (
	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/public"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EncodeArtworkID", func() {
	Context("Public ID Encoding", func() {
		BeforeEach(func() {
			auth.TokenAuth = jwtauth.New("HS256", []byte("super secret"), nil)
		})
		It("returns a reversible string representation", func() {
			id := model.NewArtworkID(model.KindArtistArtwork, "1234")
			encoded := public.EncodeArtworkID(id)
			decoded, err := public.DecodeArtworkID(encoded)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).To(Equal(id))
		})
		It("fails to decode an invalid token", func() {
			_, err := public.DecodeArtworkID("xx-123")
			Expect(err).To(MatchError("invalid JWT"))
		})
		It("fails to decode an invalid id", func() {
			encoded := public.EncodeArtworkID(model.ArtworkID{})
			_, err := public.DecodeArtworkID(encoded)
			Expect(err).To(MatchError("invalid artwork id"))
		})
		It("fails to decode a token without an id", func() {
			token, _ := auth.CreatePublicToken(map[string]any{})
			_, err := public.DecodeArtworkID(token)
			Expect(err).To(HaveOccurred())
		})
	})
})
