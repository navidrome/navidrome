package auth_test

import (
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAuth(t *testing.T) {
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Auth Test Suite")
}

const (
	oneDay = 24 * time.Hour
)

var _ = BeforeSuite(func() {
	conf.Server.SessionTimeout = 2 * oneDay
})

var _ = Describe("Auth", func() {

	BeforeEach(func() {
		ds := &tests.MockDataStore{
			MockedProperty: &tests.MockedPropertyRepo{},
		}
		auth.Init(ds)
	})

	Describe("Validate", func() {
		It("returns error with an invalid JWT token", func() {
			_, err := auth.Validate("invalid.token")
			Expect(err).To(HaveOccurred())
		})

		It("returns the claims from a valid JWT token", func() {
			claims := map[string]any{}
			claims["iss"] = "issuer"
			claims["iat"] = time.Now().Unix()
			claims["exp"] = time.Now().Add(1 * time.Minute).Unix()
			_, tokenStr, err := auth.TokenAuth.Encode(claims)
			Expect(err).NotTo(HaveOccurred())

			decodedClaims, err := auth.Validate(tokenStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(decodedClaims.Issuer).To(Equal("issuer"))
		})

		It("returns ErrExpired if the `exp` field is in the past", func() {
			claims := map[string]any{}
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

			Expect(claims.Issuer).To(Equal(consts.JWTIssuer))
			Expect(claims.Subject).To(Equal("johndoe"))
			Expect(claims.UserID).To(Equal("123"))
			Expect(claims.IsAdmin).To(Equal(true))
			Expect(claims.ExpiresAt).To(BeTemporally(">", time.Now()))
		})
	})

	Describe("Session/Public secret split", func() {
		claims := func() map[string]any {
			return map[string]any{"iss": "issuer", "exp": time.Now().Add(1 * time.Minute).Unix()}
		}

		It("verifies a session token via Validate but not ValidatePublic", func() {
			_, tokenStr, err := auth.TokenAuth.Encode(claims())
			Expect(err).NotTo(HaveOccurred())
			_, err = auth.Validate(tokenStr)
			Expect(err).NotTo(HaveOccurred())
			_, err = auth.ValidatePublic(tokenStr)
			Expect(err).To(HaveOccurred())
		})

		It("verifies a public token via ValidatePublic but not Validate", func() {
			_, tokenStr, err := auth.PublicTokenAuth.Encode(claims())
			Expect(err).NotTo(HaveOccurred())
			_, err = auth.ValidatePublic(tokenStr)
			Expect(err).NotTo(HaveOccurred())
			_, err = auth.Validate(tokenStr)
			Expect(err).To(HaveOccurred())
		})

		It("decodes public tokens minted by CreatePublicToken via PublicTokenAuth", func() {
			tokenStr, err := auth.CreatePublicToken(auth.Claims{ID: "art-1"})
			Expect(err).NotTo(HaveOccurred())
			claims, err := auth.ValidatePublic(tokenStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(claims.ID).To(Equal("art-1"))
			_, err = auth.Validate(tokenStr)
			Expect(err).To(HaveOccurred())
		})

		It("decodes expiring public tokens minted by CreateExpiringPublicToken via PublicTokenAuth", func() {
			exp := time.Now().Add(1 * time.Hour)
			tokenStr, err := auth.CreateExpiringPublicToken(exp, auth.Claims{ID: "art-2"})
			Expect(err).NotTo(HaveOccurred())
			claims, err := auth.ValidatePublic(tokenStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(claims.ID).To(Equal("art-2"))
			_, err = auth.Validate(tokenStr)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("TouchToken", func() {
		It("updates the expiration time", func() {
			yesterday := time.Now().Add(-oneDay)
			claims := map[string]any{}
			claims["iss"] = "issuer"
			claims["exp"] = yesterday.Unix()
			token, _, err := auth.TokenAuth.Encode(claims)
			Expect(err).NotTo(HaveOccurred())

			touched, err := auth.TouchToken(token)
			Expect(err).NotTo(HaveOccurred())

			decodedClaims, err := auth.Validate(touched)
			Expect(err).NotTo(HaveOccurred())
			Expect(decodedClaims.ExpiresAt.Sub(yesterday)).To(BeNumerically(">=", oneDay))
		})
	})

	Describe("CreateAPIToken", func() {
		var usr *model.User

		BeforeEach(func() {
			usr = &model.User{ID: "123", UserName: "johndoe", TokenEpoch: 4}
		})

		It("does not expire", func() {
			tokenStr, err := auth.CreateAPIToken(usr, auth.AudienceJellyfin)
			Expect(err).ToNot(HaveOccurred())

			claims, err := auth.Validate(tokenStr)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.ExpiresAt.IsZero()).To(BeTrue())
		})

		It("carries the audience and the user's epoch", func() {
			tokenStr, err := auth.CreateAPIToken(usr, auth.AudienceJellyfin)
			Expect(err).ToNot(HaveOccurred())

			claims, err := auth.Validate(tokenStr)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Audience).To(Equal([]string{"jellyfin"}))
			Expect(claims.Epoch).To(Equal(4))
			Expect(claims.Subject).To(Equal("johndoe"))
			Expect(claims.UserID).To(Equal("123"))
		})
	})

	Describe("CreateToken with an epoch", func() {
		It("carries the epoch and still expires", func() {
			usr := &model.User{ID: "123", UserName: "johndoe", TokenEpoch: 9}
			tokenStr, err := auth.CreateToken(usr)
			Expect(err).ToNot(HaveOccurred())

			claims, err := auth.Validate(tokenStr)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Epoch).To(Equal(9))
			Expect(claims.Audience).To(BeEmpty())
			Expect(claims.ExpiresAt).To(BeTemporally(">", time.Now()))
		})
	})

	Describe("TouchClaims", func() {
		It("preserves custom claims and refreshes the expiry", func() {
			tokenStr, err := auth.TouchClaims(auth.Claims{Subject: "johndoe", UserID: "123", Epoch: 5})
			Expect(err).ToNot(HaveOccurred())

			claims, err := auth.Validate(tokenStr)
			Expect(err).ToNot(HaveOccurred())
			Expect(claims.Epoch).To(Equal(5))
			Expect(claims.Subject).To(Equal("johndoe"))
			Expect(claims.ExpiresAt).To(BeTemporally(">", time.Now()))
		})
	})

	Describe("CheckClaims", func() {
		usr := model.User{ID: "123", UserName: "johndoe", TokenEpoch: 2}

		It("accepts a matching epoch and audience", func() {
			c := auth.Claims{Epoch: 2, Audience: []string{auth.AudienceJellyfin}}
			Expect(auth.CheckClaims(c, usr, auth.AudienceJellyfin)).To(Succeed())
		})

		It("accepts a token with no audience on any API", func() {
			c := auth.Claims{Epoch: 2}
			Expect(auth.CheckClaims(c, usr, auth.AudienceNative)).To(Succeed())
			Expect(auth.CheckClaims(c, usr, auth.AudienceJellyfin)).To(Succeed())
			Expect(auth.CheckClaims(c, usr, auth.AudienceSubsonic)).To(Succeed())
		})

		It("rejects a stale epoch", func() {
			c := auth.Claims{Epoch: 1, Audience: []string{auth.AudienceJellyfin}}
			Expect(auth.CheckClaims(c, usr, auth.AudienceJellyfin)).To(MatchError(auth.ErrTokenRevoked))
		})

		It("rejects a token minted for another API", func() {
			c := auth.Claims{Epoch: 2, Audience: []string{auth.AudienceJellyfin}}
			Expect(auth.CheckClaims(c, usr, auth.AudienceNative)).To(MatchError(auth.ErrWrongAudience))
			Expect(auth.CheckClaims(c, usr, auth.AudienceSubsonic)).To(MatchError(auth.ErrWrongAudience))
		})

		It("accepts a multi-audience token that includes this API", func() {
			c := auth.Claims{Epoch: 2, Audience: []string{"other", auth.AudienceNative}}
			Expect(auth.CheckClaims(c, usr, auth.AudienceNative)).To(Succeed())
		})

		It("accepts a pre-upgrade token against a never-bumped user", func() {
			fresh := model.User{ID: "456", UserName: "newbie"}
			Expect(auth.CheckClaims(auth.Claims{}, fresh, auth.AudienceNative)).To(Succeed())
		})

		It("accepts a token whose user id matches", func() {
			c := auth.Claims{UserID: "123", Epoch: 2}
			Expect(auth.CheckClaims(c, usr, auth.AudienceNative)).To(Succeed())
		})

		It("rejects a token for a deleted user recreated under the same name", func() {
			recreated := model.User{ID: "new-random-id", UserName: "johndoe"}
			c := auth.Claims{UserID: "123", Audience: []string{auth.AudienceJellyfin}}
			Expect(auth.CheckClaims(c, recreated, auth.AudienceJellyfin)).To(MatchError(auth.ErrWrongUser))
		})

		It("accepts a token that carries no user id", func() {
			fresh := model.User{ID: "456", UserName: "newbie"}
			Expect(auth.CheckClaims(auth.Claims{}, fresh, auth.AudienceNative)).To(Succeed())
		})
	})
})
