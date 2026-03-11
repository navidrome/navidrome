package auth_test

import (
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Claims", func() {
	Describe("ToMap", func() {
		It("includes only non-zero fields", func() {
			c := auth.Claims{
				Issuer:  "ND",
				Subject: "johndoe",
				UserID:  "123",
				IsAdmin: true,
			}
			m := c.ToMap()
			Expect(m).To(HaveKeyWithValue("iss", "ND"))
			Expect(m).To(HaveKeyWithValue("sub", "johndoe"))
			Expect(m).To(HaveKeyWithValue("uid", "123"))
			Expect(m).To(HaveKeyWithValue("adm", true))
			Expect(m).NotTo(HaveKey("exp"))
			Expect(m).NotTo(HaveKey("iat"))
			Expect(m).NotTo(HaveKey("id"))
			Expect(m).NotTo(HaveKey("f"))
			Expect(m).NotTo(HaveKey("b"))
		})

		It("includes expiration and issued-at when set", func() {
			now := time.Now()
			c := auth.Claims{
				IssuedAt:  now,
				ExpiresAt: now.Add(time.Hour),
			}
			m := c.ToMap()
			Expect(m).To(HaveKey("iat"))
			Expect(m).To(HaveKey("exp"))
		})

		It("includes custom claims for public tokens", func() {
			c := auth.Claims{
				ID:      "al-123",
				Format:  "mp3",
				BitRate: 192,
			}
			m := c.ToMap()
			Expect(m).To(HaveKeyWithValue("id", "al-123"))
			Expect(m).To(HaveKeyWithValue("f", "mp3"))
			Expect(m).To(HaveKeyWithValue("b", 192))
		})
	})

	Describe("ClaimsFromToken", func() {
		It("round-trips session claims through encode/decode", func() {
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			now := time.Now().Truncate(time.Second)
			original := auth.Claims{
				Issuer:  "ND",
				Subject: "johndoe",
				UserID:  "123",
				IsAdmin: true,
			}
			m := original.ToMap()
			m["iat"] = now.UTC().Unix()
			token, _, err := tokenAuth.Encode(m)
			Expect(err).NotTo(HaveOccurred())

			c := auth.ClaimsFromToken(token)
			Expect(c.Issuer).To(Equal("ND"))
			Expect(c.Subject).To(Equal("johndoe"))
			Expect(c.UserID).To(Equal("123"))
			Expect(c.IsAdmin).To(BeTrue())
			Expect(c.IssuedAt.UTC()).To(Equal(now.UTC()))
		})

		It("round-trips public token claims through encode/decode", func() {
			tokenAuth := jwtauth.New("HS256", []byte("test-secret"), nil)
			original := auth.Claims{
				Issuer:  "ND",
				ID:      "al-456",
				Format:  "opus",
				BitRate: 128,
			}
			token, _, err := tokenAuth.Encode(original.ToMap())
			Expect(err).NotTo(HaveOccurred())

			c := auth.ClaimsFromToken(token)
			Expect(c.Issuer).To(Equal("ND"))
			Expect(c.ID).To(Equal("al-456"))
			Expect(c.Format).To(Equal("opus"))
			Expect(c.BitRate).To(Equal(128))
		})
	})

})
