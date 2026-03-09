package public

import (
	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("decodeArtworkID", func() {
	BeforeEach(func() {
		auth.TokenAuth = jwtauth.New("HS256", []byte("super secret"), nil)
	})

	It("fails to decode an invalid token", func() {
		_, err := decodeArtworkID("xx-123")
		Expect(err).To(HaveOccurred())
	})

	It("fails to decode a token without an id", func() {
		token, _ := auth.CreatePublicToken(auth.Claims{})
		_, err := decodeArtworkID(token)
		Expect(err).To(HaveOccurred())
	})
})
