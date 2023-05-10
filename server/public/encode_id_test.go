package public

import (
	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("encodeArtworkID", func() {
	Context("Public ID Encoding", func() {
		BeforeEach(func() {
			auth.TokenAuth = jwtauth.New("HS256", []byte("super secret"), nil)
		})
		It("returns a reversible string representation", func() {
			id := model.NewArtworkID(model.KindArtistArtwork, "1234", nil)
			encoded := encodeArtworkID(id)
			decoded, err := decodeArtworkID(encoded)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).To(Equal(id))
		})
		It("fails to decode an invalid token", func() {
			_, err := decodeArtworkID("xx-123")
			Expect(err).To(MatchError("invalid JWT"))
		})
		It("defaults to kind mediafile", func() {
			encoded := encodeArtworkID(model.ArtworkID{})
			id, err := decodeArtworkID(encoded)
			Expect(err).ToNot(HaveOccurred())
			Expect(id.Kind).To(Equal(model.KindMediaFileArtwork))
		})
		It("fails to decode a token without an id", func() {
			token, _ := auth.CreatePublicToken(map[string]any{})
			_, err := decodeArtworkID(token)
			Expect(err).To(HaveOccurred())
		})
	})
})
