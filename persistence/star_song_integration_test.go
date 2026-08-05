package persistence

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Integration tests: Backend (real persistence layer) <-> Database (real migrated SQLite).
//
// No mocks, no stubs: SetStar issues real SQL against the migrated `annotation` table
// seeded by the suite's BeforeSuite, and every verification reads it back through the
// real annotated-query join — the same path the Subsonic getStarred endpoint uses.
//
// starredIDs returns the IDs currently in the favorites/starred list, using exactly the
// filter that filter.ByStarred() applies in the getStarred endpoint.
var _ = Describe("Star/Unstar an existing song (Backend<->DB integration)", func() {
	var repo model.MediaFileRepository

	starredIDs := func() []string {
		starred, err := repo.GetAll(model.QueryOptions{Filters: squirrel.Eq{"starred": true}})
		Expect(err).ToNot(HaveOccurred())
		ids := make([]string, 0, len(starred))
		for _, mf := range starred {
			ids = append(ids, mf.ID)
		}
		return ids
	}

	BeforeEach(func() {
		// The logged-in user scopes every annotation (loggedUser(ctx).ID). adminUser
		// (ID "userid") is seeded and associated with the default test library, so it
		// stands in for the authenticated user a real /rest/star request would carry.
		ctx := request.WithUser(context.Background(), adminUser)
		repo = NewMediaFileRepository(ctx, GetDBXBuilder())
	})

	// BE-INT-01 — Star an existing song successfully.
	Describe("Star an existing song", func() {
		// songRadioactivity (ID "1003") is an existing song seeded by BeforeSuite.
		const songID = "1003"

		BeforeEach(func() {
			// Known starting state: the song must not be starred yet.
			Expect(repo.SetStar(false, songID)).To(Succeed())
		})

		AfterEach(func() {
			// Leave the shared in-memory DB clean for other specs.
			Expect(repo.SetStar(false, songID)).To(Succeed())
		})

		It("stars the song and returns it in the starred list", func() {
			// Pre-condition: the existing song is reachable and not starred.
			before, err := repo.Get(songID)
			Expect(err).ToNot(HaveOccurred())
			Expect(before.Starred).To(BeFalse())

			// Act — the "star request": real upsert into the annotation table.
			Expect(repo.SetStar(true, songID)).To(Succeed())

			// Verify (1): re-reading the song through the real annotated query shows it
			// persisted, with a starred_at timestamp.
			after, err := repo.Get(songID)
			Expect(err).ToNot(HaveOccurred())
			Expect(after.Starred).To(BeTrue())
			Expect(after.StarredAt).ToNot(BeNil())

			// Verify (2): the song appears in the favorites/starred list.
			Expect(starredIDs()).To(ContainElement(songID))
		})
	})

	// BE-INT-02 — Unstar a previously starred song successfully.
	Describe("Unstar a previously starred song", func() {
		// songDayInALife (ID "1001") is an existing song seeded by BeforeSuite.
		const songID = "1001"

		BeforeEach(func() {
			// Pre-condition for unstar: the song must already be starred.
			Expect(repo.SetStar(true, songID)).To(Succeed())
		})

		AfterEach(func() {
			// Leave the shared in-memory DB clean for other specs.
			Expect(repo.SetStar(false, songID)).To(Succeed())
		})

		It("unstars the song and removes it from the starred list", func() {
			// Pre-condition: the existing song is reachable and currently starred.
			before, err := repo.Get(songID)
			Expect(err).ToNot(HaveOccurred())
			Expect(before.Starred).To(BeTrue())

			// Act — the "unstar request": real upsert into the annotation table.
			Expect(repo.SetStar(false, songID)).To(Succeed())

			// Verify (1): re-reading the song through the real annotated query shows it
			// is no longer starred.
			after, err := repo.Get(songID)
			Expect(err).ToNot(HaveOccurred())
			Expect(after.Starred).To(BeFalse())

			// Verify (2): the song no longer appears in the favorites/starred list.
			Expect(starredIDs()).ToNot(ContainElement(songID))
		})
	})
})
