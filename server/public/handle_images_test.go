package public

import (
	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("decodeArtworkID", func() {
	BeforeEach(func() {
		auth.TokenAuth = jwtauth.New("HS256", []byte("super secret"), nil)
	})

	It("fails to decode an invalid token", func() {
		_, err := decodeArtworkID("xx-123")
		Expect(err).To(MatchError("invalid JWT"))
	})

	It("defaults to kind mediafile for empty artwork ID", func() {
		token, _ := auth.CreatePublicToken(map[string]any{"id": ""})
		id, err := decodeArtworkID(token)
		Expect(err).ToNot(HaveOccurred())
		Expect(id.Kind).To(Equal(model.KindMediaFileArtwork))
	})

	It("fails to decode a token without an id", func() {
		token, _ := auth.CreatePublicToken(map[string]any{})
		_, err := decodeArtworkID(token)
		Expect(err).To(HaveOccurred())
	})
})
