package persistence

import (
	"time"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AppPasswordRepository", func() {
	var (
		repo   model.AppPasswordRepository
		userID string
	)

	BeforeEach(func() {
		ctx := log.NewContext(GinkgoT().Context())
		// The user repo constructor seeds the encryption key; the app password
		// repo piggybacks on that via NewUserRepository in its constructor.
		userRepo := NewUserRepository(ctx, GetDBXBuilder())
		userID = id.NewRandom()
		Expect(userRepo.Put(&model.User{
			ID:          userID,
			UserName:    "ap-user-" + userID,
			Name:        "AP User",
			NewPassword: "irrelevant",
		})).To(Succeed())

		repo = NewAppPasswordRepository(ctx, GetDBXBuilder())
	})

	Describe("Create", func() {
		It("returns plaintext and persists encrypted secret", func() {
			plain, ap, err := repo.Create(GinkgoT().Context(), userID, "phone", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(plain).ToNot(BeEmpty())
			Expect(ap.SecretEncrypted).ToNot(BeEmpty())
			Expect(ap.SecretEncrypted).ToNot(Equal(plain))
		})

		It("rejects empty user ID", func() {
			_, _, err := repo.Create(GinkgoT().Context(), "", "x", nil)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetActiveForUser", func() {
		It("decrypts and returns active passwords", func() {
			plain, _, err := repo.Create(GinkgoT().Context(), userID, "active", nil)
			Expect(err).ToNot(HaveOccurred())

			rows, err := repo.GetActiveForUser(GinkgoT().Context(), userID)
			Expect(err).ToNot(HaveOccurred())
			Expect(rows).To(HaveLen(1))
			Expect(rows[0].Secret).To(Equal(plain))
		})

		It("excludes expired passwords", func() {
			past := time.Now().Add(-time.Hour)
			_, _, err := repo.Create(GinkgoT().Context(), userID, "expired", &past)
			Expect(err).ToNot(HaveOccurred())

			rows, err := repo.GetActiveForUser(GinkgoT().Context(), userID)
			Expect(err).ToNot(HaveOccurred())
			Expect(rows).To(BeEmpty())
		})
	})

	Describe("Delete", func() {
		It("removes a password owned by the user", func() {
			_, ap, err := repo.Create(GinkgoT().Context(), userID, "to-delete", nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(repo.Delete(GinkgoT().Context(), ap.ID, userID)).To(Succeed())

			rows, err := repo.GetActiveForUser(GinkgoT().Context(), userID)
			Expect(err).ToNot(HaveOccurred())
			Expect(rows).To(BeEmpty())
		})

		It("refuses to delete a password owned by a different user", func() {
			_, ap, err := repo.Create(GinkgoT().Context(), userID, "other-owned", nil)
			Expect(err).ToNot(HaveOccurred())

			err = repo.Delete(GinkgoT().Context(), ap.ID, "someone-else")
			Expect(err).To(MatchError(model.ErrNotFound))
		})
	})

	Describe("UpdateLastUsedAt", func() {
		It("sets last_used_at on the row", func() {
			_, ap, err := repo.Create(GinkgoT().Context(), userID, "lu", nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(repo.UpdateLastUsedAt(GinkgoT().Context(), ap.ID)).To(Succeed())

			pubs, err := repo.ListForUser(GinkgoT().Context(), userID)
			Expect(err).ToNot(HaveOccurred())
			Expect(pubs).ToNot(BeEmpty())
			Expect(pubs[0].LastUsedAt).ToNot(BeNil())
		})
	})

	Describe("ListForUser", func() {
		It("returns metadata only, ordered newest first", func() {
			_, _, err := repo.Create(GinkgoT().Context(), userID, "first", nil)
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(10 * time.Millisecond)
			_, _, err = repo.Create(GinkgoT().Context(), userID, "second", nil)
			Expect(err).ToNot(HaveOccurred())

			pubs, err := repo.ListForUser(GinkgoT().Context(), userID)
			Expect(err).ToNot(HaveOccurred())
			Expect(pubs).To(HaveLen(2))
			Expect(pubs[0].Name).To(Equal("second"))
			Expect(pubs[1].Name).To(Equal("first"))
		})
	})
})
