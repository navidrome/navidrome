package apiauth

import (
	"context"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// renewingDS runs renew right before DeleteIdle, as a node resolving the grant meanwhile would.
type renewingDS struct {
	model.DataStore
	renew func()
}

func (d renewingDS) Grant() model.GrantRepository {
	return renewingGrants{GrantRepository: d.DataStore.Grant(), renew: d.renew}
}

type renewingGrants struct {
	model.GrantRepository
	renew func()
}

func (g renewingGrants) DeleteIdle(ctx context.Context, idleSince time.Time) (int64, error) {
	g.renew()
	return g.GrantRepository.DeleteIdle(ctx, idleSince)
}

var meta = ClientMeta{Name: "Living room", Client: "TestApp", ClientVersion: "1.0"}

var _ = Describe("Service: grants", func() {
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
			issued, err := svc.Login(ctx, u.UserName, "pw", meta, nil)
			Expect(err).ToNot(HaveOccurred())
			p, err := svc.Authenticate(ctx, issued.Secret, "10.0.0.9")
			Expect(err).ToNot(HaveOccurred())
			Expect(p.User.ID).To(Equal(u.ID))
			Expect(p.GrantID).To(Equal(issued.Grant.ID))
			Expect(p.Scopes).To(Equal([]string{ScopePassword, ScopeRead}))
		})

		It("carries only the scopes stored on a narrow grant", func() {
			u := createUser(ctx, "pw", false)
			issued, err := svc.Login(ctx, u.UserName, "pw", meta, []string{ScopePassword})
			Expect(err).ToNot(HaveOccurred())
			p, err := svc.Authenticate(ctx, issued.Secret, "")
			Expect(err).ToNot(HaveOccurred())
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

		It("keeps an idle grant that another node renewed before the delete ran", func() {
			u := createUser(ctx, "pw", false)
			issued, _ := svc.Login(ctx, u.UserName, "pw", meta, nil)
			renewedAt := now.Add(IdleExpiry - time.Minute)
			racing := New(renewingDS{DataStore: realDS, renew: func() {
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
