package jellyfin

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("awaitSimilar", func() {
	It("returns the fetch result when it completes within the quick wait", func() {
		res := awaitSimilar(context.Background(), func(context.Context) dto.QueryResult {
			return result([]dto.BaseItemDto{{Name: "fast"}}, 1, 0)
		})
		Expect(res.Items).To(HaveLen(1))
		Expect(res.Items[0].Name).To(Equal("fast"))
	})

	It("returns an empty result (does not block the client) when the fetch exceeds the quick wait", func() {
		res := awaitSimilar(context.Background(), func(context.Context) dto.QueryResult {
			time.Sleep(2 * similarQuickWait) // slow external lookup; finishes caching in the background
			return result([]dto.BaseItemDto{{Name: "late"}}, 1, 0)
		})
		Expect(res.Items).To(BeEmpty())
		Expect(res.TotalRecordCount).To(Equal(0))
	})
})
