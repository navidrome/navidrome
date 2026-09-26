package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GrantRepository", func() {
	var ctx context.Context
	var repo model.GrantRepository
	var now time.Time

	newGrant := func(userID, hash string) *model.Grant {
		return &model.Grant{UserID: userID, Name: "TV", Client: "TestApp", Scopes: model.Scopes{"all"},
			Provider: "password", SecretHash: hash, CreatedAt: now}
	}

	BeforeEach(func() {
		ctx = log.NewContext(GinkgoT().Context())
		repo = NewGrantRepository(GetDBXBuilder())
		now = time.Now().UTC().Truncate(time.Second)
		DeferCleanup(func() {
			_, _ = GetDBXBuilder().NewQuery("delete from api_grant").Execute()
		})
	})

	It("stores a grant and finds it by id and by secret hash", func() {
		g := newGrant(adminUser.ID, "hash-1")
		g.Scopes = model.Scopes{"read", "password"}
		Expect(repo.Put(ctx, g)).To(Succeed())
		Expect(g.ID).ToNot(BeEmpty())

		byID, err := repo.Get(ctx, g.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(byID.Scopes).To(Equal(model.Scopes{"read", "password"}))
		Expect(byID.LastUsedAt).To(BeNil())
		Expect(byID.LastUsedIP).To(BeEmpty())

		byHash, err := repo.FindBySecretHash(ctx, "hash-1")
		Expect(err).ToNot(HaveOccurred())
		Expect(byHash.ID).To(Equal(g.ID))
	})

	It("returns ErrNotFound for unknown ids and hashes", func() {
		_, err := repo.Get(ctx, "nope")
		Expect(err).To(MatchError(model.ErrNotFound))
		_, err = repo.FindBySecretHash(ctx, "nope")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("lists and counts only the user's non-idle grants by lastUsedAt, never-used ones last", func() {
		old := newGrant(adminUser.ID, "h-old")
		old.CreatedAt = now.Add(-100 * 24 * time.Hour)
		usedEarly := newGrant(adminUser.ID, "h-used-early")
		usedEarly.CreatedAt = now.Add(-10 * time.Hour)
		earlyUse := now.Add(-5 * time.Hour)
		usedEarly.LastUsedAt = &earlyUse
		usedLate := newGrant(adminUser.ID, "h-used-late")
		usedLate.CreatedAt = now.Add(-10 * time.Hour)
		lateUse := now.Add(-time.Hour)
		usedLate.LastUsedAt = &lateUse
		freshNeverUsed := newGrant(adminUser.ID, "h-fresh") // newer than both uses, but never used
		other := newGrant(regularUser.ID, "h-other")
		for _, g := range []*model.Grant{old, usedEarly, usedLate, freshNeverUsed, other} {
			Expect(repo.Put(ctx, g)).To(Succeed())
		}
		idleSince := now.Add(-90 * 24 * time.Hour)

		list, err := repo.GetAllForUser(ctx, adminUser.ID, idleSince, 0, 10)
		Expect(err).ToNot(HaveOccurred())
		Expect([]string{list[0].ID, list[1].ID, list[2].ID}).To(Equal([]string{usedLate.ID, usedEarly.ID, freshNeverUsed.ID}))

		Expect(repo.CountForUser(ctx, adminUser.ID, idleSince)).To(Equal(int64(3)))

		page, err := repo.GetAllForUser(ctx, adminUser.ID, idleSince, 1, 1)
		Expect(err).ToNot(HaveOccurred())
		Expect(page).To(HaveLen(1))
		Expect(page[0].ID).To(Equal(usedEarly.ID))
	})

	It("deletes a grant only for its owner", func() {
		g := newGrant(adminUser.ID, "h-own")
		Expect(repo.Put(ctx, g)).To(Succeed())
		Expect(repo.DeleteForUser(ctx, regularUser.ID, g.ID)).To(MatchError(model.ErrNotFound))
		Expect(repo.DeleteForUser(ctx, adminUser.ID, g.ID)).To(Succeed())
		_, err := repo.Get(ctx, g.ID)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("moves epochs forward and deletes grants left on other epochs", func() {
		keep := newGrant(adminUser.ID, "h-keep")
		drop := newGrant(adminUser.ID, "h-drop")
		Expect(repo.Put(ctx, keep)).To(Succeed())
		Expect(repo.Put(ctx, drop)).To(Succeed())

		Expect(repo.SetEpoch(ctx, adminUser.ID, 0, 3, keep.ID)).To(Succeed())
		Expect(repo.DeleteOtherEpochs(ctx, adminUser.ID, 3)).To(Succeed())

		kept, err := repo.Get(ctx, keep.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(kept.UserEpoch).To(Equal(3))
		_, err = repo.Get(ctx, drop.ID)
		Expect(err).To(MatchError(model.ErrNotFound))

		Expect(repo.SetEpoch(ctx, adminUser.ID, 3, 4, "")).To(Succeed())
		kept, _ = repo.Get(ctx, keep.ID)
		Expect(kept.UserEpoch).To(Equal(4))
	})

	It("never moves a grant that is not on fromEpoch", func() {
		stale := newGrant(adminUser.ID, "h-stale") // left behind by an earlier password change
		stale.UserEpoch = 1
		current := newGrant(adminUser.ID, "h-current")
		current.UserEpoch = 2
		Expect(repo.Put(ctx, stale)).To(Succeed())
		Expect(repo.Put(ctx, current)).To(Succeed())

		Expect(repo.SetEpoch(ctx, adminUser.ID, 2, 3, "")).To(Succeed())
		got, _ := repo.Get(ctx, stale.ID)
		Expect(got.UserEpoch).To(Equal(1))
		got, _ = repo.Get(ctx, current.ID)
		Expect(got.UserEpoch).To(Equal(3))
	})

	It("deletes by epoch only while the row is still on it", func() {
		g := newGrant(adminUser.ID, "h-cond")
		g.UserEpoch = 5
		Expect(repo.Put(ctx, g)).To(Succeed())
		Expect(repo.DeleteIfEpoch(ctx, g.ID, 4)).To(Succeed())
		_, err := repo.Get(ctx, g.ID)
		Expect(err).ToNot(HaveOccurred())
		Expect(repo.DeleteIfEpoch(ctx, g.ID, 5)).To(Succeed())
		_, err = repo.Get(ctx, g.ID)
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("touches a never-used grant, then throttles until notSince passes", func() {
		g := newGrant(adminUser.ID, "h-touch")
		Expect(repo.Put(ctx, g)).To(Succeed())

		Expect(repo.Touch(ctx, g.ID, "10.0.0.1", now, now.Add(-5*time.Minute))).To(Succeed())
		got, _ := repo.Get(ctx, g.ID)
		Expect(got.LastUsedAt).ToNot(BeNil())
		Expect(got.LastUsedAt.UTC()).To(BeTemporally("==", now))
		Expect(got.LastUsedIP).To(Equal("10.0.0.1"))

		later := now.Add(time.Minute)
		Expect(repo.Touch(ctx, g.ID, "10.0.0.2", later, later.Add(-5*time.Minute))).To(Succeed())
		got, _ = repo.Get(ctx, g.ID)
		Expect(got.LastUsedIP).To(Equal("10.0.0.1"))

		muchLater := now.Add(6 * time.Minute)
		Expect(repo.Touch(ctx, g.ID, "10.0.0.3", muchLater, muchLater.Add(-5*time.Minute))).To(Succeed())
		got, _ = repo.Get(ctx, g.ID)
		Expect(got.LastUsedIP).To(Equal("10.0.0.3"))
	})

	It("deletes idle grants, using created_at for never-used ones", func() {
		idle := newGrant(adminUser.ID, "h-idle")
		idle.CreatedAt = now.Add(-100 * 24 * time.Hour)
		usedRecently := newGrant(adminUser.ID, "h-used-recently")
		usedRecently.CreatedAt = now.Add(-100 * 24 * time.Hour)
		recentUse := now.Add(-time.Hour)
		usedRecently.LastUsedAt = &recentUse
		Expect(repo.Put(ctx, idle)).To(Succeed())
		Expect(repo.Put(ctx, usedRecently)).To(Succeed())

		n, err := repo.DeleteIdle(ctx, now.Add(-90*24*time.Hour))
		Expect(err).ToNot(HaveOccurred())
		Expect(n).To(Equal(int64(1)))
		_, err = repo.Get(ctx, usedRecently.ID)
		Expect(err).ToNot(HaveOccurred())
	})

	It("deletes a user's grants when the user is deleted", func() {
		users := NewUserRepository(GetDBXBuilder())
		u := model.User{ID: "grant-owner", UserName: "grant-owner", NewPassword: "pw"}
		Expect(users.Put(ctx, &u)).To(Succeed())
		g := newGrant(u.ID, "h-cascade")
		Expect(repo.Put(ctx, g)).To(Succeed())

		Expect(users.Delete(request.WithUser(ctx, adminUser), u.ID)).To(Succeed())
		_, err := repo.Get(ctx, g.ID)
		Expect(err).To(MatchError(model.ErrNotFound))
	})
})
