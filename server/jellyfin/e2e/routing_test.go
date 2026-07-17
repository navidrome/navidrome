package e2e

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Routing", func() {
	BeforeEach(func() { setupTestDB() })

	It("routes authenticated endpoints case-insensitively", func() {
		// Lowercase path variant of GET /Items — real clients (jellyfin-apiclient-python) send these.
		lower := queryResult(get("/items?IncludeItemTypes=MusicAlbum&Recursive=true"))
		Expect(lower.TotalRecordCount).To(Equal(5))
	})

	It("returns a JSON 404 for an unknown route", func() {
		w := get("/Nonexistent/Route")
		Expect(w.Code).To(Equal(http.StatusNotFound))
		Expect(w.Header().Get("Content-Type")).To(HavePrefix("application/json"))
		Expect(w.Body.String()).To(ContainSubstring("{}"))
	})

	It("returns 404 for an unsupported method on a known path", func() {
		// PUT isn't registered for /Items; the MethodNotAllowed handler maps to the same JSON 404.
		w := jReq(adminUser, "PUT", "/Items", "")
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})
})
