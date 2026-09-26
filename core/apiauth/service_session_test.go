package apiauth

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service: sessions", func() {
	var ctx context.Context
	var svc *Service
	var now time.Time

	login := func(u model.User) (*Issued, *Principal, *AccessToken) {
		issued, err := svc.Login(ctx, u.UserName, "pw", meta, nil)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		p, err := svc.ResolveGrant(ctx, issued.Secret, "")
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		tok, err := svc.Mint(ctx, p, nil)
		ExpectWithOffset(1, err).ToNot(HaveOccurred())
		return issued, p, tok
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		now = time.Now().UTC().Truncate(time.Second)
		svc = New(realDS)
		svc.SetClock(func() time.Time { return now })
	})

	Describe("Authenticate", func() {
		It("returns the token's scopes and marks the grant used", func() {
			u := createUser(ctx, "pw", false)
			issued, _, tok := login(u)
			now = now.Add(10 * time.Minute)
			p, err := svc.Authenticate(ctx, tok.Token, "10.1.1.1")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.GrantID).To(Equal(issued.Grant.ID))
			Expect(p.Scopes).To(Equal(tok.Scopes))
			g, _ := realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(g.LastUsedIP).To(Equal("10.1.1.1"))
		})

		It("reports an expired token as ErrTokenExpired", func() {
			u := createUser(ctx, "pw", false)
			_, _, tok := login(u)
			now = now.Add(TokenTTL + clockSkew + time.Second)
			_, err := svc.Authenticate(ctx, tok.Token, "")
			Expect(err).To(MatchError(ErrTokenExpired))
		})

		It("rejects a token at once on the node that revoked its grant", func() {
			u := createUser(ctx, "pw", false)
			_, p, tok := login(u)
			_, err := svc.Authenticate(ctx, tok.Token, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.Logout(ctx, p)).To(Succeed())
			_, err = svc.Authenticate(ctx, tok.Token, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("stops a token revoked on another node within the cache time", func() {
			u := createUser(ctx, "pw", false)
			issued, _, tok := login(u)
			_, err := svc.Authenticate(ctx, tok.Token, "")
			Expect(err).ToNot(HaveOccurred())

			Expect(realDS.Grant().Delete(ctx, issued.Grant.ID)).To(Succeed()) // another node
			_, err = svc.Authenticate(ctx, tok.Token, "")
			Expect(err).ToNot(HaveOccurred()) // still cached
			now = now.Add(cacheTTL)
			_, err = svc.Authenticate(ctx, tok.Token, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("kills grants when the password changes anywhere else", func() {
			u := createUser(ctx, "pw", false)
			_, _, tok := login(u)
			u.NewPassword = "reset-by-admin"
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			_, err := svc.Authenticate(ctx, tok.Token, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("does not kill a grant kept by a password change made through another node", func() {
			u := createUser(ctx, "pw", false)
			_, p, tok := login(u)
			_, err := svc.Authenticate(ctx, tok.Token, "") // caches the old epoch
			Expect(err).ToNot(HaveOccurred())

			other := New(realDS) // another node
			other.SetClock(func() time.Time { return now })
			Expect(other.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", false)).To(Succeed())

			_, err = svc.Authenticate(ctx, tok.Token, "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("rejects a token carrying admin once the user is no longer an admin", func() {
			saved := KnownScopes
			KnownScopes = []string{ScopeRead, ScopePassword, ScopeAdmin}
			DeferCleanup(func() { KnownScopes = saved })
			u := createUser(ctx, "pw", true)
			_, p, tok := login(u)
			Expect(tok.Scopes).To(ContainElement(ScopeAdmin))

			u.IsAdmin = false
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			_, err := svc.Authenticate(ctx, tok.Token, "")
			Expect(err).To(MatchError(ErrInsufficientScope))

			fresh, err := svc.Mint(ctx, &Principal{User: u, GrantID: p.GrantID, Scopes: Expand(model.Scopes{ScopeAll}, false)}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(fresh.Scopes).ToNot(ContainElement(ScopeAdmin))
		})

		It("rejects a live token after its user is deleted, and the grant row is gone", func() {
			u := createUser(ctx, "pw", false)
			issued, _, tok := login(u)
			Expect(realDS.User().Delete(request.WithUser(ctx, model.User{IsAdmin: true}), u.ID)).To(Succeed())
			_, err := realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
			now = now.Add(cacheTTL)
			_, err = svc.Authenticate(ctx, tok.Token, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("rejects a token whose grant belongs to another user, even across an epoch change", func() {
			alice := createUser(ctx, "pw", false)
			bob := createUser(ctx, "pw", false)
			bobGrant, _, _ := login(bob)
			alice.NewPassword = "bumped"
			Expect(realDS.User().Put(ctx, &alice)).To(Succeed())

			sg, err := svc.signer()
			Expect(err).ToNot(HaveOccurred())
			tok, err := sg.sign(claims{UserID: alice.ID, GrantID: bobGrant.Grant.ID, Scopes: []string{ScopeRead}, IssuedAt: now, ExpiresAt: now.Add(TokenTTL)})
			Expect(err).ToNot(HaveOccurred())
			_, err = svc.Authenticate(ctx, tok, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("does not delete a kept grant when the user was read before a password change", func() {
			u := createUser(ctx, "pw", false)
			_, p, _ := login(u)
			stale, err := realDS.User().Get(ctx, u.ID) // read before the change lands
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", false)).To(Succeed())

			_, _, err = svc.liveGrant(ctx, p.GrantID, stale)
			Expect(err).ToNot(HaveOccurred())
			_, err = realDS.Grant().Get(ctx, p.GrantID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("leaves a login that raced a password change with a dead grant", func() {
			u := createUser(ctx, "pw", false)
			reached, release := make(chan struct{}), make(chan struct{})
			svc.SetCheckers(func(ds model.DataStore) []CredentialChecker {
				return []CredentialChecker{pausingChecker{inner: dbChecker{ds: ds}, reached: reached, release: release}}
			})
			var issued *Issued
			var loginErr error
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				issued, loginErr = svc.Login(ctx, u.UserName, "pw", meta, nil)
			}()
			<-reached // credentials (and the old epoch) were read
			u.NewPassword = "changed-meanwhile"
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			close(release)
			<-done

			Expect(loginErr).ToNot(HaveOccurred())
			_, err := New(realDS).ResolveGrant(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})
	})

	Describe("grant management", func() {
		It("lists the user's grants and marks the current one", func() {
			u := createUser(ctx, "pw", false)
			first, _, _ := login(u)
			_, p, _ := login(u)
			grants, total, err := svc.ListGrants(ctx, p, 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(grants).To(HaveLen(2))
			Expect([]string{grants[0].ID, grants[1].ID}).To(ContainElements(first.Grant.ID, p.GrantID))
		})

		It("lists only grants on the user's current epoch", func() {
			u := createUser(ctx, "pw", false)
			login(u)
			u.NewPassword = "reset-by-admin" // old-UI reset leaves the old grant on the previous epoch
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			issued, err := svc.Login(ctx, u.UserName, "reset-by-admin", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			p, err := svc.ResolveGrant(ctx, issued.Secret, "")
			Expect(err).ToNot(HaveOccurred())

			grants, total, err := svc.ListGrants(ctx, p, 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(grants).To(HaveLen(1))
			Expect(grants[0].ID).To(Equal(issued.Grant.ID))
		})

		It("logs out successfully when the grant is already gone", func() {
			u := createUser(ctx, "pw", false)
			_, p, tok := login(u)
			_, err := svc.Authenticate(ctx, tok.Token, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(realDS.Grant().Delete(ctx, p.GrantID)).To(Succeed()) // another node

			Expect(svc.Logout(ctx, p)).To(Succeed())
			_, err = svc.Authenticate(ctx, tok.Token, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("refuses to revoke another user's grant", func() {
			alice := createUser(ctx, "pw", false)
			bob := createUser(ctx, "pw", false)
			aliceGrant, _, _ := login(alice)
			_, bobP, _ := login(bob)
			Expect(svc.RevokeGrant(ctx, bobP, aliceGrant.Grant.ID)).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("ChangePassword", func() {
		It("revokes other grants by default and keeps the caller's", func() {
			u := createUser(ctx, "pw", false)
			_, _, otherTok := login(u)
			_, p, myTok := login(u)
			Expect(svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)).To(Succeed())

			_, err := svc.Authenticate(ctx, myTok.Token, "")
			Expect(err).ToNot(HaveOccurred())
			now = now.Add(cacheTTL)
			_, err = svc.Authenticate(ctx, otherTok.Token, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))

			_, err = svc.Login(ctx, u.UserName, "pw2", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("keeps every grant when revokeOthers is false", func() {
			u := createUser(ctx, "pw", false)
			_, _, otherTok := login(u)
			_, p, _ := login(u)
			Expect(svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", false)).To(Succeed())
			now = now.Add(cacheTTL)
			_, err := svc.Authenticate(ctx, otherTok.Token, "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("rejects a wrong current password without changing anything", func() {
			u := createUser(ctx, "pw", false)
			_, p, _ := login(u)
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "wrong", "pw2", true)
			Expect(err).To(MatchError(ErrCurrentPasswordMismatch))
			_, err = svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("is forbidden for non-admins when user editing is off", func() {
			conf.Server.EnableUserEditing = false
			u := createUser(ctx, "pw", false)
			_, p, _ := login(u)
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("does not revive grants killed by an earlier reset when keeping grants", func() {
			u := createUser(ctx, "pw", false)
			killed, _, _ := login(u)
			u.NewPassword = "reset-by-admin" // old-UI reset: the killed grant stays on the old epoch until presented
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())

			issued2, err := svc.Login(ctx, u.UserName, "reset-by-admin", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			p2, err := svc.ResolveGrant(ctx, issued2.Secret, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.ChangePassword(request.WithUser(ctx, p2.User), p2, "reset-by-admin", "pw3", false)).To(Succeed())

			_, err = svc.ResolveGrant(ctx, killed.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("rejects a caller whose grant was revoked before the change ran", func() {
			u := createUser(ctx, "pw", false)
			_, p, _ := login(u)
			Expect(realDS.Grant().Delete(ctx, p.GrantID)).To(Succeed())
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)
			Expect(err).To(MatchError(model.ErrInvalidAuth))
			_, err = svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("rolls back the password and epoch when a grant update fails", func() {
			u := createUser(ctx, "pw", false)
			_, p, _ := login(u)
			failing := New(failingEpochDS{realDS})
			failing.SetClock(func() time.Time { return now })

			err := failing.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)
			Expect(err).To(MatchError(ContainSubstring("boom")))

			reloaded, _ := realDS.User().Get(ctx, u.ID)
			Expect(reloaded.TokenEpoch).To(Equal(u.TokenEpoch))
			_, err = svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			_, err = svc.Authenticate(ctx, mustMint(svc, ctx, p), "")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

type pausingChecker struct {
	inner            CredentialChecker
	reached, release chan struct{}
}

func (c pausingChecker) Check(ctx context.Context, username, password string) (CredentialResult, error) {
	res, err := c.inner.Check(ctx, username, password)
	close(c.reached)
	<-c.release
	return res, err
}

// failingEpochDS makes SetEpoch fail inside WithTxImmediate, to prove the whole change rolls back.
type failingEpochDS struct{ model.DataStore }

func (f failingEpochDS) WithTxImmediate(block func(tx model.DataStore) error, scope ...string) error {
	return f.DataStore.WithTxImmediate(func(tx model.DataStore) error {
		return block(failingEpochTx{tx})
	}, scope...)
}

type failingEpochTx struct{ model.DataStore }

func (f failingEpochTx) Grant() model.GrantRepository { return failingGrants{f.DataStore.Grant()} }

type failingGrants struct{ model.GrantRepository }

func (failingGrants) SetEpoch(context.Context, string, int, int, string) error {
	return errors.New("boom")
}

func mustMint(svc *Service, ctx context.Context, p *Principal) string {
	tok, err := svc.Mint(ctx, p, nil)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return tok.Token
}
