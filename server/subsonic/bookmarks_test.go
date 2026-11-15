package subsonic

import (
	"context"
	"fmt"

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
	})

	Context("when some queued tracks fail to load", func() {
		BeforeEach(func() {
			ds.MockedPlayQueue = &tests.MockPlayQueueRepo{
				Queue:             &model.PlayQueue{UserID: "u1", Items: model.MediaFiles{{ID: "visible"}}},
				WithMediaFilesErr: fmt.Errorf("%w: chunk failed", model.ErrIncompleteQueue),
			}
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
	})
})
