package e2e

import (
	"net/http"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Smoke test: proves the harness boots (DB, scan, snapshot, router, token auth) and the seeded
// library is queryable end-to-end. Broader per-endpoint coverage lives in the sibling files.
var _ = Describe("Smoke", func() {
	BeforeEach(func() { setupTestDB() })

	It("serves public system info without auth", func() {
		w := rawReq("GET", "/System/Info/Public", "")
		Expect(w.Code).To(Equal(http.StatusOK))
		var info map[string]any
		parseInto(w, &info)
		Expect(info).To(HaveKey("ServerName"))
		Expect(info).To(HaveKey("Version"))
	})

	It("rejects an authenticated endpoint without a token", func() {
		w := rawReq("GET", "/Items?IncludeItemTypes=MusicAlbum&Recursive=true", "")
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("lists the seeded albums for an authenticated user", func() {
		q := queryResult(get("/Items?IncludeItemTypes=MusicAlbum&Recursive=true"))
		Expect(q.TotalRecordCount).To(Equal(5))
		names := make([]string, 0, len(q.Items))
		for _, it := range q.Items {
			Expect(it.Type).To(Equal("MusicAlbum"))
			names = append(names, it.Name)
		}
		Expect(names).To(ConsistOf("Abbey Road", "Help!", "IV", "Kind of Blue", "Singles"))
	})

	It("resolves a seeded album id round-trip (encoded in the URL)", func() {
		id := albumID("Abbey Road")
		var item dto.BaseItemDto
		parseInto(get("/Items/"+enc(id)), &item)
		Expect(item.Id).To(Equal(enc(id)))
		Expect(item.Name).To(Equal("Abbey Road"))
		Expect(item.Type).To(Equal("MusicAlbum"))
	})
})
