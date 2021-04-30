package auth_test

import (
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"

	"github.com/dgrijalva/jwt-go"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	log.SetLevel(log.LevelCritical)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Test Suite")
}

const (
	testJWTSecret = "not so secret"
	oneDay        = 24 * time.Hour
)

var _ = Describe("Auth", func() {
	BeforeSuite(func() {
		conf.Server.SessionTimeout = 2 * oneDay
	})

	BeforeEach(func() {
		auth.Secret = []byte(testJWTSecret)
	})

	Describe("Validate", func() {
		It("returns error with an invalid JWT token", func() {
			_, err := auth.Validate("invalid.token")
			Expect(err).To(Not(BeNil()))
		})

		It("returns the claims from a valid JWT token", func() {
			token := jwt.New(jwt.SigningMethodHS256)
			claims := token.Claims.(jwt.MapClaims)
			claims["iss"] = "issuer"
			claims["exp"] = time.Now().Add(1 * time.Minute).Unix()
			tokenStr, _ := token.SignedString(auth.Secret)

			decodedClaims, err := auth.Validate(tokenStr)
			Expect(err).To(BeNil())
			Expect(decodedClaims["iss"]).To(Equal("issuer"))
		})

		It("returns ErrExpired if the `exp` field is in the past", func() {
			token := jwt.New(jwt.SigningMethodHS256)
			claims := token.Claims.(jwt.MapClaims)
			claims["iss"] = "issuer"
			claims["exp"] = time.Now().Add(-1 * time.Minute).Unix()
			tokenStr, _ := token.SignedString(auth.Secret)

			_, err := auth.Validate(tokenStr)
			Expect(err).To(MatchError("Token is expired"))
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
			Expect(err).To(BeNil())

			claims, err := auth.Validate(tokenStr)
			Expect(err).To(BeNil())

			Expect(claims["iss"]).To(Equal(consts.JWTIssuer))
			Expect(claims["sub"]).To(Equal("johndoe"))
			Expect(claims["uid"]).To(Equal("123"))
			Expect(claims["adm"]).To(Equal(true))

			exp := time.Unix(int64(claims["exp"].(float64)), 0)
			Expect(exp).To(BeTemporally(">", time.Now()))
		})
	})

	Describe("TouchToken", func() {
		It("updates the expiration time", func() {
			yesterday := time.Now().Add(-oneDay)
			token := jwt.New(jwt.SigningMethodHS256)
			claims := token.Claims.(jwt.MapClaims)
			claims["iss"] = "issuer"
			claims["exp"] = yesterday.Unix()

			touched, err := auth.TouchToken(token)
			Expect(err).To(BeNil())

			decodedClaims, err := auth.Validate(touched)
			Expect(err).To(BeNil())
			expiration := time.Unix(int64(decodedClaims["exp"].(float64)), 0)
			Expect(expiration.Sub(yesterday)).To(BeNumerically(">=", oneDay))
		})
	})
})
