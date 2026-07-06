package jellyfin

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("awaitSimilar", func() {
	var api *Router
	ctxFor := func(userID string) context.Context {
		return request.WithUser(context.Background(), model.User{ID: userID})
	}

	BeforeEach(func() {
		api = &Router{}
	})

	It("returns the fetch result when it completes within the quick wait", func() {
		res := api.awaitSimilar(ctxFor("u1"), "id1", 20, func(context.Context) dto.QueryResult {
			return result([]dto.BaseItemDto{{Name: "fast"}}, 1, 0)
		})
		Expect(res.Items).To(HaveLen(1))
		Expect(res.Items[0].Name).To(Equal("fast"))
	})

	It("returns an empty result (does not block the client) when the fetch exceeds the quick wait", func() {
		res := api.awaitSimilar(ctxFor("u1"), "id2", 20, func(context.Context) dto.QueryResult {
			time.Sleep(2 * similarQuickWait) // slow external lookup; finishes caching in the background
			return result([]dto.BaseItemDto{{Name: "late"}}, 1, 0)
		})
		Expect(res.Items).To(BeEmpty())
		Expect(res.TotalRecordCount).To(Equal(0))
	})

	It("dedupes concurrent identical requests into a single fetch", func() {
		var calls atomic.Int32
		release := make(chan struct{})
		fetch := func(context.Context) dto.QueryResult {
			calls.Add(1)
			<-release
			return result([]dto.BaseItemDto{{Name: "shared"}}, 1, 0)
		}
		var wg sync.WaitGroup
		for range 5 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				api.awaitSimilar(ctxFor("u1"), "id3", 20, fetch)
			}()
		}
		// Let all five reach the flight, then release the single in-flight fetch.
		Eventually(calls.Load).Should(Equal(int32(1)))
		close(release)
		wg.Wait()
		Expect(calls.Load()).To(Equal(int32(1)))
	})

	It("does not share fetches across users (items embed the user's annotations)", func() {
		var calls atomic.Int32
		fetch := func(context.Context) dto.QueryResult {
			calls.Add(1)
			return result(nil, 0, 0)
		}
		api.awaitSimilar(ctxFor("u1"), "id4", 20, fetch)
		api.awaitSimilar(ctxFor("u2"), "id4", 20, fetch)
		Expect(calls.Load()).To(Equal(int32(2)))
	})

	It("hands the fetch a deadline-bounded background context", func() {
		var deadline time.Time
		var hasDeadline bool
		api.awaitSimilar(ctxFor("u1"), "id5", 20, func(ctx context.Context) dto.QueryResult {
			deadline, hasDeadline = ctx.Deadline()
			return result(nil, 0, 0)
		})
		Expect(hasDeadline).To(BeTrue(), "background fetch must not be able to run forever")
		Expect(time.Until(deadline)).To(BeNumerically("<=", similarFetchTimeout))
	})
})
