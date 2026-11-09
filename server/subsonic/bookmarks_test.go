package subsonic

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bookmarks", func() {
	var router *Router
	var ds *tests.MockDataStore
	var mfRepo *tests.MockMediaFileRepo
	var ctx context.Context

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		ctx = request.WithUser(context.Background(), model.User{ID: "u1", UserName: "u1"})
		mfRepo = ds.MediaFile().(*tests.MockMediaFileRepo)
		mfRepo.SetData(model.MediaFiles{{ID: "visible"}})
	})

	Describe("CreateBookmark", func() {
		It("rejects an id the user cannot read", func() {
			r := newGetRequest("id=hidden", "position=1").WithContext(ctx)

			_, err := router.CreateBookmark(r)

			Expect(err).To(HaveOccurred())
			Expect(mapToSubsonicError(err).code).To(Equal(responses.ErrorDataNotFound))
			Expect(mfRepo.BookmarksAdded).To(BeEmpty())
		})

		It("accepts an id the user can read", func() {
			r := newGetRequest("id=visible", "position=1").WithContext(ctx)

			_, err := router.CreateBookmark(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(mfRepo.BookmarksAdded).To(ConsistOf("visible"))
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
