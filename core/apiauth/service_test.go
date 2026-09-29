package apiauth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var meta = ClientMeta{Name: "Living room", Client: "TestApp", ClientVersion: "1.0"}

var _ = Describe("Service", func() {
	var ctx context.Context
	var svc *Service
	var now time.Time

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		now = time.Now().UTC().Truncate(time.Second)
		svc = New(realDS)
		svc.SetClock(func() time.Time { return now })
	})

	Describe("Login", func() {
		It("creates a grant storing all, the user's epoch and the client metadata", func() {
			u := createUser(ctx, "pw", false)
			issued, err := svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(issued.Secret).To(HavePrefix("ndg_"))
			Expect(issued.User.ID).To(Equal(u.ID))
			Expect(issued.User.Password).To(BeEmpty())
			Expect(issued.Grant.Scopes).To(Equal(model.Scopes{ScopeAll}))
			Expect(issued.Grant.Provider).To(Equal("password"))
			Expect(issued.Grant.Name).To(Equal("Living room"))
			Expect(issued.Grant.UserEpoch).To(Equal(u.TokenEpoch))

			stored, err := realDS.Grant().FindBySecretHash(ctx, hashSecret(issued.Secret))
			Expect(err).ToNot(HaveOccurred())
			Expect(stored.ID).To(Equal(issued.Grant.ID))
		})

		It("defaults the grant name to the client", func() {
			u := createUser(ctx, "pw", false)
			issued, err := svc.Login(ctx, u.UserName, "pw", ClientMeta{Client: "OnlyClient"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(issued.Grant.Name).To(Equal("OnlyClient"))
		})

		It("accepts the username in any case", func() {
			u := createUser(ctx, "pw", false)
			issued, err := svc.Login(ctx, strings.ToUpper(u.UserName), "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(issued.User.ID).To(Equal(u.ID))
		})

		It("stores only known requested scopes", func() {
			u := createUser(ctx, "pw", false)
			issued, err := svc.Login(ctx, u.UserName, "pw", meta, []string{"read", "future", "admin"})
			Expect(err).ToNot(HaveOccurred())
			Expect(issued.Grant.Scopes).To(Equal(model.Scopes{ScopeRead}))
		})

		It("fails with ErrInvalidAuth for bad credentials", func() {
			u := createUser(ctx, "pw", false)
			_, err := svc.Login(ctx, u.UserName, "wrong", meta, nil)
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})
	})

	Describe("Setup", func() {
		It("refuses when users exist", func() {
			createUser(ctx, "pw", false)
			_, err := svc.Setup(ctx, "newadmin", "pw", meta, nil)
			Expect(err).To(MatchError(auth.ErrSetupComplete))
		})
		// The empty-database path is covered end to end in server/apiv1, which owns a fresh DB.
	})

	Describe("Authenticate", func() {
		It("resolves the secret to its user and the grant's expanded scopes", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u, "pw", nil)
			Expect(p.User.ID).To(Equal(u.ID))
			Expect(p.GrantID).To(Equal(issued.Grant.ID))
			Expect(p.Scopes).To(Equal([]string{ScopePassword, ScopeRead}))
		})

		It("carries only the scopes stored on a narrow grant", func() {
			u := createUser(ctx, "pw", false)
			_, p := login(ctx, svc, u, "pw", []string{ScopePassword})
			Expect(p.Scopes).To(Equal([]string{ScopePassword}))
		})

		It("records the first use with the client IP", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := svc.Login(ctx, u.UserName, "pw", meta, nil)
			_, err := svc.Authenticate(ctx, issued.Secret, "10.0.0.9")
			Expect(err).ToNot(HaveOccurred())
			g, _ := realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(g.LastUsedAt).ToNot(BeNil())
			Expect(g.LastUsedIP).To(Equal("10.0.0.9"))
		})

		It("records use again only after the touch interval", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := svc.Login(ctx, u.UserName, "pw", meta, nil)
			_, err := svc.Authenticate(ctx, issued.Secret, "10.0.0.1")
			Expect(err).ToNot(HaveOccurred())

			now = now.Add(touchInterval - time.Second)
			_, err = svc.Authenticate(ctx, issued.Secret, "10.0.0.2")
			Expect(err).ToNot(HaveOccurred())
			g, _ := realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(g.LastUsedIP).To(Equal("10.0.0.1"))

			now = now.Add(2 * time.Second)
			_, err = svc.Authenticate(ctx, issued.Secret, "10.0.0.3")
			Expect(err).ToNot(HaveOccurred())
			g, _ = realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(g.LastUsedIP).To(Equal("10.0.0.3"))
			Expect(g.LastUsedAt.Equal(now)).To(BeTrue())
		})

		It("rejects unknown secrets", func() {
			_, err := svc.Authenticate(ctx, "ndg_unknown", "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("deletes and rejects a grant idle for 90 days, including one never used", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := svc.Login(ctx, u.UserName, "pw", meta, nil)
			now = now.Add(IdleExpiry + time.Second)
			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
			_, err = realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("keeps an idle grant that a concurrent request renewed before the delete ran", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := svc.Login(ctx, u.UserName, "pw", meta, nil)
			renewedAt := now.Add(IdleExpiry - time.Minute)
			racing := New(hookDS{DataStore: realDS, beforeDeleteIdle: func() {
				Expect(realDS.Grant().Touch(ctx, issued.Grant.ID, "10.0.0.2", renewedAt, renewedAt)).To(Succeed())
			}})
			now = now.Add(IdleExpiry + time.Second)
			racing.SetClock(func() time.Time { return now })

			_, err := racing.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
			g, err := realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(g.LastUsedIP).To(Equal("10.0.0.2"))
		})

		It("rejects and deletes a grant whose epoch is behind the user's", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := svc.Login(ctx, u.UserName, "pw", meta, nil)
			u.NewPassword = "changed-elsewhere"
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
			_, err = realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
		})

		It("does not delete a kept grant when the password changed between reading the grant and the user", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u, "pw", nil)
			racing := New(hookDS{DataStore: realDS, afterFind: func() {
				Expect(svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", false)).To(Succeed())
			}})
			racing.SetClock(func() time.Time { return now })

			_, err := racing.Authenticate(ctx, issued.Secret, "")
			Expect(err).ToNot(HaveOccurred())
			_, err = realDS.Grant().Get(ctx, p.GrantID)
			Expect(err).ToNot(HaveOccurred())
		})

		It("drops admin from a grant once its user is no longer an admin", func() {
			saved := KnownScopes
			KnownScopes = []string{ScopeRead, ScopePassword, ScopeAdmin}
			DeferCleanup(func() { KnownScopes = saved })
			u := createUser(ctx, "pw", true)
			issued, p := login(ctx, svc, u, "pw", nil)
			Expect(p.Scopes).To(ContainElement(ScopeAdmin))

			u.IsAdmin = false
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			demoted, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(demoted.Scopes).To(Equal([]string{ScopePassword, ScopeRead}))
		})

		It("rejects the secret after its user is deleted, and the grant row is gone", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := login(ctx, svc, u, "pw", nil)
			Expect(realDS.User().Delete(request.WithUser(ctx, model.User{IsAdmin: true}), u.ID)).To(Succeed())
			_, err := realDS.Grant().Get(ctx, issued.Grant.ID)
			Expect(err).To(MatchError(model.ErrNotFound))
			_, err = svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
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
			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})
	})

	Describe("grant management", func() {
		It("lists the user's grants and marks the current one", func() {
			u := createUser(ctx, "pw", false)
			first, _ := login(ctx, svc, u, "pw", nil)
			_, p := login(ctx, svc, u, "pw", nil)
			grants, total, err := svc.ListGrants(ctx, p, 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(grants).To(HaveLen(2))
			Expect([]string{grants[0].ID, grants[1].ID}).To(ContainElements(first.Grant.ID, p.GrantID))
		})

		It("lists only grants on the user's current epoch", func() {
			u := createUser(ctx, "pw", false)
			login(ctx, svc, u, "pw", nil)
			u.NewPassword = "reset-by-admin" // old-UI reset leaves the old grant on the previous epoch
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			issued, p := login(ctx, svc, u, "reset-by-admin", nil)

			grants, total, err := svc.ListGrants(ctx, p, 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(grants).To(HaveLen(1))
			Expect(grants[0].ID).To(Equal(issued.Grant.ID))
		})

		It("logs out, and succeeds again when the grant is already gone", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u, "pw", nil)
			Expect(svc.Logout(ctx, p)).To(Succeed())
			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))

			Expect(svc.Logout(ctx, p)).To(Succeed())
		})

		It("refuses to revoke another user's grant", func() {
			alice := createUser(ctx, "pw", false)
			bob := createUser(ctx, "pw", false)
			aliceGrant, _ := login(ctx, svc, alice, "pw", nil)
			_, bobP := login(ctx, svc, bob, "pw", nil)
			Expect(svc.RevokeGrant(ctx, bobP, aliceGrant.Grant.ID)).To(MatchError(model.ErrNotFound))
			_, err := svc.Authenticate(ctx, aliceGrant.Secret, "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("rejects the secret after its grant is revoked", func() {
			u := createUser(ctx, "pw", false)
			other, _ := login(ctx, svc, u, "pw", nil)
			_, p := login(ctx, svc, u, "pw", nil)
			Expect(svc.RevokeGrant(ctx, p, other.Grant.ID)).To(Succeed())
			_, err := svc.Authenticate(ctx, other.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})
	})

	Describe("ChangePassword", func() {
		It("revokes other grants by default and keeps the caller's", func() {
			u := createUser(ctx, "pw", false)
			other, _ := login(ctx, svc, u, "pw", nil)
			mine, p := login(ctx, svc, u, "pw", nil)
			Expect(svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)).To(Succeed())

			_, err := svc.Authenticate(ctx, mine.Secret, "")
			Expect(err).ToNot(HaveOccurred())
			_, err = svc.Authenticate(ctx, other.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))

			_, err = svc.Login(ctx, u.UserName, "pw2", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("keeps every grant, the caller's included, when revokeOthers is false", func() {
			u := createUser(ctx, "pw", false)
			other, _ := login(ctx, svc, u, "pw", nil)
			mine, p := login(ctx, svc, u, "pw", nil)
			Expect(svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", false)).To(Succeed())
			_, err := svc.Authenticate(ctx, other.Secret, "")
			Expect(err).ToNot(HaveOccurred())
			_, err = svc.Authenticate(ctx, mine.Secret, "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("rejects a wrong current password without changing anything", func() {
			u := createUser(ctx, "pw", false)
			_, p := login(ctx, svc, u, "pw", nil)
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "wrong", "pw2", true)
			Expect(err).To(MatchError(ErrCurrentPasswordMismatch))
			_, err = svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("is forbidden for non-admins when user editing is off", func() {
			conf.Server.EnableUserEditing = false
			u := createUser(ctx, "pw", false)
			_, p := login(ctx, svc, u, "pw", nil)
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("does not revive grants killed by an earlier reset when keeping grants", func() {
			u := createUser(ctx, "pw", false)
			killed, _ := login(ctx, svc, u, "pw", nil)
			u.NewPassword = "reset-by-admin" // old-UI reset: the killed grant stays on the old epoch until presented
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())

			_, p2 := login(ctx, svc, u, "reset-by-admin", nil)
			Expect(svc.ChangePassword(request.WithUser(ctx, p2.User), p2, "reset-by-admin", "pw3", false)).To(Succeed())

			_, err := svc.Authenticate(ctx, killed.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("rejects a caller whose grant was revoked before the change ran", func() {
			u := createUser(ctx, "pw", false)
			_, p := login(ctx, svc, u, "pw", nil)
			Expect(realDS.Grant().DeleteForUser(ctx, u.ID, p.GrantID)).To(Succeed())
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)
			Expect(err).To(MatchError(model.ErrInvalidAuth))
			_, err = svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("rejects a caller naming another user's grant", func() {
			alice := createUser(ctx, "pw", false)
			bob := createUser(ctx, "pw", false)
			_, aliceP := login(ctx, svc, alice, "pw", nil)
			bobGrant, _ := login(ctx, svc, bob, "pw", nil)
			forged := &Principal{User: aliceP.User, GrantID: bobGrant.Grant.ID}
			err := svc.ChangePassword(request.WithUser(ctx, alice), forged, "pw", "pw2", true)
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("rolls back the password and epoch when a grant update fails", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u, "pw", nil)
			failing := New(failingEpochDS{realDS})
			failing.SetClock(func() time.Time { return now })

			err := failing.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)
			Expect(err).To(MatchError(ContainSubstring("boom")))

			reloaded, _ := realDS.User().Get(ctx, u.ID)
			Expect(reloaded.TokenEpoch).To(Equal(u.TokenEpoch))
			_, err = svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			_, err = svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("PasswordChangeable", func() {
		It("follows EnableUserEditing for non-admins only", func() {
			conf.Server.EnableUserEditing = false
			Expect(PasswordChangeable(model.User{IsAdmin: true})).To(BeTrue())
			Expect(PasswordChangeable(model.User{})).To(BeFalse())
			conf.Server.EnableUserEditing = true
			Expect(PasswordChangeable(model.User{})).To(BeTrue())
		})
	})
})

// hookDS runs its optional callbacks inside grant lookups, to land a concurrent change mid-Authenticate.
type hookDS struct {
	model.DataStore
	afterFind        func()
	beforeDeleteIdle func()
}

func (d hookDS) Grant() model.GrantRepository {
	return hookGrants{GrantRepository: d.DataStore.Grant(), hooks: d}
}

type hookGrants struct {
	model.GrantRepository
	hooks hookDS
}

func (g hookGrants) FindBySecretHash(ctx context.Context, hash string) (*model.Grant, error) {
	found, err := g.GrantRepository.FindBySecretHash(ctx, hash)
	if g.hooks.afterFind != nil {
		g.hooks.afterFind()
	}
	return found, err
}

func (g hookGrants) DeleteIdle(ctx context.Context, idleSince time.Time) (int64, error) {
	if g.hooks.beforeDeleteIdle != nil {
		g.hooks.beforeDeleteIdle()
	}
	return g.GrantRepository.DeleteIdle(ctx, idleSince)
}

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
