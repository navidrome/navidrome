package auth_test

import (
	"testing"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Test Suite")
}

const (
	testJWTSecret = "not so secret"
	oneDay        = 24 * time.Hour
)

var _ = BeforeSuite(func() {
	conf.Server.SessionTimeout = 2 * oneDay
})

var _ = Describe("Auth", func() {

	BeforeEach(func() {
		auth.Secret = []byte(testJWTSecret)
		auth.TokenAuth = jwtauth.New("HS256", auth.Secret, nil)
	})

	Describe("Validate", func() {
		It("returns error with an invalid JWT token", func() {
			_, err := auth.Validate("invalid.token")
			Expect(err).To(HaveOccurred())
		})

		It("returns the claims from a valid JWT token", func() {
			claims := map[string]interface{}{}
			claims["iss"] = "issuer"
			claims["iat"] = time.Now().Unix()
			claims["exp"] = time.Now().Add(1 * time.Minute).Unix()
			_, tokenStr, err := auth.TokenAuth.Encode(claims)
			Expect(err).NotTo(HaveOccurred())

			decodedClaims, err := auth.Validate(tokenStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(decodedClaims["iss"]).To(Equal("issuer"))
		})

		It("returns ErrExpired if the `exp` field is in the past", func() {
			claims := map[string]interface{}{}
			claims["iss"] = "issuer"
			claims["exp"] = time.Now().Add(-1 * time.Minute).Unix()
			_, tokenStr, err := auth.TokenAuth.Encode(claims)
			Expect(err).NotTo(HaveOccurred())

			_, err = auth.Validate(tokenStr)
			Expect(err).To(MatchError("token is expired"))
		})
	})

	Describe("CreateToken", func() {
		It("creates a valid token", func() {
			u := &model.User{
				ID:       "123",
				UserName: "johndoe",
				IsAdmin:  true,
			}
			tokenStr, err := auth.CreateToken(u)
			Expect(err).NotTo(HaveOccurred())

			claims, err := auth.Validate(tokenStr)
			Expect(err).NotTo(HaveOccurred())

			Expect(claims["iss"]).To(Equal(consts.JWTIssuer))
			Expect(claims["sub"]).To(Equal("johndoe"))
			Expect(claims["uid"]).To(Equal("123"))
			Expect(claims["adm"]).To(Equal(true))
			Expect(claims["exp"]).To(BeTemporally(">", time.Now()))
		})
	})

	Describe("TouchToken", func() {
		It("updates the expiration time", func() {
			yesterday := time.Now().Add(-oneDay)
			claims := map[string]interface{}{}
			claims["iss"] = "issuer"
			claims["exp"] = yesterday.Unix()
			token, _, err := auth.TokenAuth.Encode(claims)
			Expect(err).NotTo(HaveOccurred())

			touched, err := auth.TouchToken(token)
			Expect(err).NotTo(HaveOccurred())

			decodedClaims, err := auth.Validate(touched)
			Expect(err).NotTo(HaveOccurred())
			exp := decodedClaims["exp"].(time.Time)
			Expect(exp.Sub(yesterday)).To(BeNumerically(">=", oneDay))
		})
	})
})
