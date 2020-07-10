package auth_test

import (
	"testing"
	"time"

	"github.com/deluan/navidrome/core/auth"
	"github.com/deluan/navidrome/log"
	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Test Suite")
}

const testJWTSecret = "not so secret"

var _ = Describe("Auth", func() {
	BeforeEach(func() {
		auth.JwtSecret = []byte(testJWTSecret)
	})
	Context("Validate", func() {
		It("returns error with an invalid JWT token", func() {
			_, err := auth.Validate("invalid.token")
			Expect(err).To(Not(BeNil()))
		})

		It("returns the claims from a valid JWT token", func() {
			token := jwt.New(jwt.SigningMethodHS256)
			claims := token.Claims.(jwt.MapClaims)
			claims["iss"] = "issuer"
			claims["exp"] = time.Now().Add(1 * time.Minute).Unix()
			tokenStr, _ := token.SignedString(auth.JwtSecret)

			decodedClaims, err := auth.Validate(tokenStr)
			Expect(err).To(BeNil())
			Expect(decodedClaims["iss"]).To(Equal("issuer"))
		})

		It("returns ErrExpired if the `exp` field is in the past", func() {
			token := jwt.New(jwt.SigningMethodHS256)
			claims := token.Claims.(jwt.MapClaims)
			claims["iss"] = "issuer"
			claims["exp"] = time.Now().Add(-1 * time.Minute).Unix()
			tokenStr, _ := token.SignedString(auth.JwtSecret)

			_, err := auth.Validate(tokenStr)
			Expect(err).To(MatchError("Token is expired"))
		})
	})
})
