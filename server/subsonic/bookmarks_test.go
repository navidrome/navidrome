package subsonic

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bookmarks", func() {
	var router *Router
	var ds model.DataStore
	var ctx context.Context

	bookmarkedIDs := func() []string {
		GinkgoHelper()
		bookmarks, err := ds.MediaFile().GetBookmarks(ctx)
		Expect(err).ToNot(HaveOccurred())
		ids := make([]string, len(bookmarks))
		for i, b := range bookmarks {
			ids[i] = b.Item.ID
		}
		return ids
	}

	BeforeEach(func() {
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "u1", UserName: "u1"})
		ds = persistence.New(db.Db())
		Expect(ds.Library().Put(ctx, &model.Library{ID: 2, Name: "Private", Path: "/private"})).To(Succeed())
		putUsers(ctx, ds, model.User{ID: "u1", UserName: "u1"})
		persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "visible"}, model.MediaFile{ID: "hidden", LibraryID: 2})
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	})

	Describe("CreateBookmark", func() {
		It("rejects an id the user cannot read", func() {
			r := newGetRequest("id=hidden", "position=1").WithContext(ctx)

			_, err := router.CreateBookmark(r)

			Expect(err).To(HaveOccurred())
			Expect(mapToSubsonicError(err).code).To(Equal(responses.ErrorDataNotFound))
			Expect(bookmarkedIDs()).To(BeEmpty())
		})

		It("accepts an id the user can read", func() {
			r := newGetRequest("id=visible", "position=1").WithContext(ctx)

			_, err := router.CreateBookmark(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(bookmarkedIDs()).To(ConsistOf("visible"))
		})
	})

	Context("when some queued tracks fail to load", func() {
		BeforeEach(func() {
			Expect(ds.PlayQueue().Store(ctx, &model.PlayQueue{UserID: "u1", Items: model.MediaFiles{{ID: "visible"}}})).To(Succeed())
			router = New(persistencetest.Stub(ds, persistencetest.Repos{PlayQueue: func(r model.PlayQueueRepository) model.PlayQueueRepository {
				return incompletePlayQueueRepo{r}
			}}),
				nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		})

		It("getPlayQueue returns the loaded tracks", func() {
			resp, err := router.GetPlayQueue(newGetRequest().WithContext(ctx))

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PlayQueue.Entry).To(HaveLen(1))
		})

		It("getPlayQueueByIndex returns the loaded tracks", func() {
			resp, err := router.GetPlayQueueByIndex(newGetRequest().WithContext(ctx))

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.PlayQueueByIndex.Entry).To(HaveLen(1))
		})

		It("accepts a zero position", func() {
			r := newGetRequest("id=visible", "position=0").WithContext(ctx)

			_, err := router.CreateBookmark(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(mfRepo.BookmarksAdded).To(ConsistOf("visible"))
		})

		DescribeTable("rejects negative positions instead of storing them",
			func(position string) {
				r := newGetRequest("id=visible", "position="+position).WithContext(ctx)

				_, err := router.CreateBookmark(r)

				Expect(err).To(HaveOccurred())
				Expect(mapToSubsonicError(err).code).To(Equal(responses.ErrorGeneric))
				Expect(mfRepo.BookmarksAdded).To(BeEmpty())
			},
			Entry("minus one", "-1"),
			Entry("large negative", "-9223372036854775808"),
		)
	})

	Describe("SavePlayQueue", func() {
		var pqRepo *tests.MockPlayQueueRepo

		BeforeEach(func() {
			pqRepo = ds.PlayQueue().(*tests.MockPlayQueueRepo)
		})

		It("clamps a negative position to zero and stores the queue", func() {
			r := newGetRequest("id=1", "current=1", "position=-5").WithContext(ctx)

			_, err := router.SavePlayQueue(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(pqRepo.Queue).ToNot(BeNil())
			Expect(pqRepo.Queue.Position).To(BeEquivalentTo(0))
		})

		It("stores a zero position", func() {
			r := newGetRequest("id=1", "current=1", "position=0").WithContext(ctx)

			_, err := router.SavePlayQueue(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(pqRepo.Queue).ToNot(BeNil())
			Expect(pqRepo.Queue.Position).To(BeEquivalentTo(0))
		})
	})

	Describe("SavePlayQueueByIndex", func() {
		var pqRepo *tests.MockPlayQueueRepo

		BeforeEach(func() {
			pqRepo = ds.PlayQueue().(*tests.MockPlayQueueRepo)
		})

		It("clamps a negative position to zero and stores the queue", func() {
			r := newGetRequest("id=1", "currentIndex=0", "position=-5").WithContext(ctx)

			_, err := router.SavePlayQueueByIndex(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(pqRepo.Queue).ToNot(BeNil())
			Expect(pqRepo.Queue.Position).To(BeEquivalentTo(0))
		})
	})
})

type incompletePlayQueueRepo struct{ model.PlayQueueRepository }

func (r incompletePlayQueueRepo) RetrieveWithMediaFiles(ctx context.Context, userId string) (*model.PlayQueue, error) {
	pq, err := r.PlayQueueRepository.RetrieveWithMediaFiles(ctx, userId)
	if err != nil {
		return pq, err
	}
	return pq, fmt.Errorf("%w: chunk failed", model.ErrIncompleteQueue)
}
