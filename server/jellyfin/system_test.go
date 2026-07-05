package jellyfin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/server/jellyfin/dto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("System", func() {
	var api *Router
	BeforeEach(func() { api = &Router{} })

	It("returns public system info without auth", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Info/Public", nil)
		api.getPublicSystemInfo(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
		var info dto.PublicSystemInfo
		Expect(json.Unmarshal(w.Body.Bytes(), &info)).To(Succeed())
		Expect(info.Id).ToNot(BeEmpty())
		Expect(info.Version).ToNot(BeEmpty())
		Expect(info.ProductName).To(Equal("Jellyfin Server"))
	})

	It("responds to ping with the server name", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Ping", nil)
		api.ping(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		var name string
		Expect(json.Unmarshal(w.Body.Bytes(), &name)).To(Succeed())
		Expect(name).ToNot(BeEmpty())
	})

	It("reports quick connect as disabled", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/QuickConnect/Enabled", nil)
		api.quickConnectEnabled(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		var enabled bool
		Expect(json.Unmarshal(w.Body.Bytes(), &enabled)).To(Succeed())
		Expect(enabled).To(BeFalse())
	})
})
