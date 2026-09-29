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

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		DeferCleanup(configtest.SetupConfig())
		now = time.Now().UTC().Truncate(time.Second)
		svc = New(realDS)
		svc.SetClock(func() time.Time { return now })
	})

	Describe("Authenticate", func() {
		It("rejects the secret at once after logout", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u)
			Expect(svc.Logout(ctx, p)).To(Succeed())
			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("rejects the secret at once when another node revoked its grant", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := login(ctx, svc, u)
			Expect(realDS.Grant().DeleteForUser(ctx, u.ID, issued.Grant.ID)).To(Succeed())
			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("kills grants when the password changes anywhere else", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := login(ctx, svc, u)
			u.NewPassword = "reset-by-admin"
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("does not kill a grant kept by a password change made through another node", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u)

			other := New(realDS) // another node
			other.SetClock(func() time.Time { return now })
			Expect(other.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", false)).To(Succeed())

			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("does not delete a kept grant when the password changed between reading the grant and the user", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u)
			racing := New(afterFindDS{DataStore: realDS, after: func() {
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
			issued, p := login(ctx, svc, u)
			Expect(p.Scopes).To(ContainElement(ScopeAdmin))

			u.IsAdmin = false
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			demoted, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(demoted.Scopes).To(Equal([]string{ScopePassword, ScopeRead}))
		})

		It("rejects the secret after its user is deleted, and the grant row is gone", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := login(ctx, svc, u)
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
			_, err := New(realDS).Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})
	})

	Describe("grant management", func() {
		It("lists the user's grants and marks the current one", func() {
			u := createUser(ctx, "pw", false)
			first, _ := login(ctx, svc, u)
			_, p := login(ctx, svc, u)
			grants, total, err := svc.ListGrants(ctx, p, 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(2)))
			Expect(grants).To(HaveLen(2))
			Expect([]string{grants[0].ID, grants[1].ID}).To(ContainElements(first.Grant.ID, p.GrantID))
		})

		It("lists only grants on the user's current epoch", func() {
			u := createUser(ctx, "pw", false)
			login(ctx, svc, u)
			u.NewPassword = "reset-by-admin" // old-UI reset leaves the old grant on the previous epoch
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())
			issued, err := svc.Login(ctx, u.UserName, "reset-by-admin", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			p, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).ToNot(HaveOccurred())

			grants, total, err := svc.ListGrants(ctx, p, 0, 10)
			Expect(err).ToNot(HaveOccurred())
			Expect(total).To(Equal(int64(1)))
			Expect(grants).To(HaveLen(1))
			Expect(grants[0].ID).To(Equal(issued.Grant.ID))
		})

		It("logs out successfully when the grant is already gone", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u)
			Expect(realDS.Grant().DeleteForUser(ctx, u.ID, p.GrantID)).To(Succeed()) // another node

			Expect(svc.Logout(ctx, p)).To(Succeed())
			_, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("refuses to revoke another user's grant", func() {
			alice := createUser(ctx, "pw", false)
			bob := createUser(ctx, "pw", false)
			aliceGrant, _ := login(ctx, svc, alice)
			_, bobP := login(ctx, svc, bob)
			Expect(svc.RevokeGrant(ctx, bobP, aliceGrant.Grant.ID)).To(MatchError(model.ErrNotFound))
			_, err := svc.Authenticate(ctx, aliceGrant.Secret, "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("rejects the secret at once after its grant is revoked", func() {
			u := createUser(ctx, "pw", false)
			other, _ := login(ctx, svc, u)
			_, p := login(ctx, svc, u)
			Expect(svc.RevokeGrant(ctx, p, other.Grant.ID)).To(Succeed())
			_, err := svc.Authenticate(ctx, other.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})
	})

	Describe("ChangePassword", func() {
		It("revokes other grants by default and keeps the caller's", func() {
			u := createUser(ctx, "pw", false)
			other, _ := login(ctx, svc, u)
			mine, p := login(ctx, svc, u)
			Expect(svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)).To(Succeed())

			_, err := svc.Authenticate(ctx, mine.Secret, "")
			Expect(err).ToNot(HaveOccurred())
			_, err = svc.Authenticate(ctx, other.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))

			_, err = svc.Login(ctx, u.UserName, "pw2", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("keeps every grant when revokeOthers is false", func() {
			u := createUser(ctx, "pw", false)
			other, _ := login(ctx, svc, u)
			_, p := login(ctx, svc, u)
			Expect(svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", false)).To(Succeed())
			_, err := svc.Authenticate(ctx, other.Secret, "")
			Expect(err).ToNot(HaveOccurred())
		})

		It("rejects a wrong current password without changing anything", func() {
			u := createUser(ctx, "pw", false)
			_, p := login(ctx, svc, u)
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "wrong", "pw2", true)
			Expect(err).To(MatchError(ErrCurrentPasswordMismatch))
			_, err = svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("is forbidden for non-admins when user editing is off", func() {
			conf.Server.EnableUserEditing = false
			u := createUser(ctx, "pw", false)
			_, p := login(ctx, svc, u)
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)
			Expect(err).To(MatchError(model.ErrNotAuthorized))
		})

		It("does not revive grants killed by an earlier reset when keeping grants", func() {
			u := createUser(ctx, "pw", false)
			killed, _ := login(ctx, svc, u)
			u.NewPassword = "reset-by-admin" // old-UI reset: the killed grant stays on the old epoch until presented
			Expect(realDS.User().Put(ctx, &u)).To(Succeed())

			issued2, err := svc.Login(ctx, u.UserName, "reset-by-admin", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			p2, err := svc.Authenticate(ctx, issued2.Secret, "")
			Expect(err).ToNot(HaveOccurred())
			Expect(svc.ChangePassword(request.WithUser(ctx, p2.User), p2, "reset-by-admin", "pw3", false)).To(Succeed())

			_, err = svc.Authenticate(ctx, killed.Secret, "")
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("rejects a caller whose grant was revoked before the change ran", func() {
			u := createUser(ctx, "pw", false)
			_, p := login(ctx, svc, u)
			Expect(realDS.Grant().DeleteForUser(ctx, u.ID, p.GrantID)).To(Succeed())
			err := svc.ChangePassword(request.WithUser(ctx, p.User), p, "pw", "pw2", true)
			Expect(err).To(MatchError(model.ErrInvalidAuth))
			_, err = svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
		})

		It("rejects a caller naming another user's grant", func() {
			alice := createUser(ctx, "pw", false)
			bob := createUser(ctx, "pw", false)
			_, aliceP := login(ctx, svc, alice)
			bobGrant, _ := login(ctx, svc, bob)
			forged := &Principal{User: aliceP.User, GrantID: bobGrant.Grant.ID}
			err := svc.ChangePassword(request.WithUser(ctx, alice), forged, "pw", "pw2", true)
			Expect(err).To(MatchError(model.ErrInvalidAuth))
		})

		It("rolls back the password and epoch when a grant update fails", func() {
			u := createUser(ctx, "pw", false)
			issued, p := login(ctx, svc, u)
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
})

// afterFindDS calls after between finding the grant by its secret and reading its user.
type afterFindDS struct {
	model.DataStore
	after func()
}

func (d afterFindDS) Grant() model.GrantRepository {
	return afterFindGrants{GrantRepository: d.DataStore.Grant(), after: d.after}
}

type afterFindGrants struct {
	model.GrantRepository
	after func()
}

func (g afterFindGrants) FindBySecretHash(ctx context.Context, hash string) (*model.Grant, error) {
	found, err := g.GrantRepository.FindBySecretHash(ctx, hash)
	g.after()
	return found, err
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
