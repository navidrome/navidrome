package apiauth

import (
	"context"
	"encoding/hex"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("signer", func() {
	var ctx context.Context
	var now time.Time
	clock := func() time.Time { return now }

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		now = time.Now().UTC().Truncate(time.Second)
		Expect(realDS.Property().Delete(ctx, consts.JWTAPIv1SecretKey)).To(Or(Succeed(), MatchError(model.ErrNotFound)))
	})

	It("creates a 256-bit key", func() {
		key, err := loadKey(ctx, realDS)
		Expect(err).ToNot(HaveOccurred())
		raw, err := hex.DecodeString(key)
		Expect(err).ToNot(HaveOccurred())
		Expect(raw).To(HaveLen(32))
	})

	It("keeps using a stored key created in the older format", func() {
		enc, err := utils.Encrypt(ctx, auth.EncryptionKey(), "legacy22charskeyABCDEF")
		Expect(err).ToNot(HaveOccurred())
		Expect(realDS.Property().Put(ctx, consts.JWTAPIv1SecretKey, enc)).To(Succeed())
		Expect(loadKey(ctx, realDS)).To(Equal("legacy22charskeyABCDEF"))
	})

	It("round-trips claims", func() {
		s, err := loadSigner(ctx, realDS, clock)
		Expect(err).ToNot(HaveOccurred())
		tok, err := s.sign(claims{UserID: "u1", GrantID: "g1", Scopes: []string{"read", "password"},
			IssuedAt: now, ExpiresAt: now.Add(time.Hour)})
		Expect(err).ToNot(HaveOccurred())

		c, err := s.parse(tok)
		Expect(err).ToNot(HaveOccurred())
		Expect(c.UserID).To(Equal("u1"))
		Expect(c.GrantID).To(Equal("g1"))
		Expect(c.Scopes).To(Equal([]string{"read", "password"}))
		Expect(c.ExpiresAt).To(BeTemporally("==", now.Add(time.Hour)))
	})

	It("round-trips a token with no scopes", func() {
		s, _ := loadSigner(ctx, realDS, clock)
		tok, _ := s.sign(claims{UserID: "u1", GrantID: "g1", IssuedAt: now, ExpiresAt: now.Add(time.Hour)})

		c, err := s.parse(tok)
		Expect(err).ToNot(HaveOccurred())
		Expect(c.Scopes).To(BeEmpty())
	})

	It("reports expiry distinctly", func() {
		s, _ := loadSigner(ctx, realDS, clock)
		tok, _ := s.sign(claims{UserID: "u1", GrantID: "g1", IssuedAt: now, ExpiresAt: now.Add(time.Hour)})
		now = now.Add(time.Hour + clockSkew + time.Second)
		_, err := s.parse(tok)
		Expect(err).To(MatchError(ErrTokenExpired))
	})

	It("accepts a token issued slightly ahead of the verifier's clock", func() {
		s, _ := loadSigner(ctx, realDS, clock)
		tok, _ := s.sign(claims{UserID: "u1", GrantID: "g1", IssuedAt: now.Add(5 * time.Second), ExpiresAt: now.Add(time.Hour)})
		_, err := s.parse(tok)
		Expect(err).ToNot(HaveOccurred())
	})

	It("rejects a token issued further ahead than the allowed skew", func() {
		s, _ := loadSigner(ctx, realDS, clock)
		tok, _ := s.sign(claims{UserID: "u1", GrantID: "g1", IssuedAt: now.Add(clockSkew + time.Second), ExpiresAt: now.Add(time.Hour)})
		_, err := s.parse(tok)
		Expect(err).To(MatchError(model.ErrInvalidAuth))
	})

	It("rejects garbage and tokens signed with another key", func() {
		s, _ := loadSigner(ctx, realDS, clock)
		_, err := s.parse("not-a-token")
		Expect(err).To(MatchError(model.ErrInvalidAuth))

		other := &signer{}
		*other = *s
		other.auth = newJWTAuth([]byte("another-key"), clock)
		tok, _ := other.sign(claims{UserID: "u1", GrantID: "g1", IssuedAt: now, ExpiresAt: now.Add(time.Hour)})
		_, err = s.parse(tok)
		Expect(err).To(MatchError(model.ErrInvalidAuth))
	})

	It("rejects a correctly signed token with no exp, or with another audience", func() {
		s, _ := loadSigner(ctx, realDS, clock)
		_, noExp, err := s.auth.Encode(map[string]any{
			jwt.SubjectKey: "u1", jwt.AudienceKey: []string{Audience}, "gid": "g1", "scope": "read",
		})
		Expect(err).ToNot(HaveOccurred())
		_, err = s.parse(noExp)
		Expect(err).To(MatchError(model.ErrInvalidAuth))

		_, otherAud, _ := s.auth.Encode(map[string]any{
			jwt.SubjectKey: "u1", jwt.AudienceKey: []string{"navidrome-other"}, jwt.ExpirationKey: now.Add(time.Hour),
			"gid": "g1", "scope": "read",
		})
		_, err = s.parse(otherAud)
		Expect(err).To(MatchError(model.ErrInvalidAuth))
	})

	loadConcurrently := func() []*signer {
		var wg sync.WaitGroup
		signers := make([]*signer, 4)
		for i := range signers {
			wg.Add(1)
			go func() {
				defer GinkgoRecover()
				defer wg.Done()
				var err error
				signers[i], err = loadSigner(ctx, realDS, clock)
				Expect(err).ToNot(HaveOccurred())
			}()
		}
		wg.Wait()
		return signers
	}

	expectSameKey := func(signers []*signer) {
		tok, _ := signers[0].sign(claims{UserID: "u1", GrantID: "g1", IssuedAt: now, ExpiresAt: now.Add(time.Hour)})
		for _, s := range signers[1:] {
			_, err := s.parse(tok)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
		}
	}

	It("gives every concurrent loader the same key", func() {
		expectSameKey(loadConcurrently())
	})

	It("gives every concurrent loader the same replacement for an unreadable key", func() {
		Expect(realDS.Property().Put(ctx, consts.JWTAPIv1SecretKey, "not-decryptable")).To(Succeed())
		expectSameKey(loadConcurrently())
	})
})
