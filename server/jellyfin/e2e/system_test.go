package e2e

import (
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("System", func() {
	BeforeEach(func() { setupTestDB() })

	Describe("GET /System/Info/Public", func() {
		It("returns public server info without authentication", func() {
			w := rawReq("GET", "/System/Info/Public", "")
			Expect(w.Code).To(Equal(http.StatusOK))
			var info map[string]any
			parseInto(w, &info)
			Expect(info["ServerName"]).To(HavePrefix("Navidrome"))
			Expect(info["ProductName"]).To(Equal("Jellyfin Server"))
			Expect(info["StartupWizardCompleted"]).To(BeTrue())
			Expect(info["Id"]).ToNot(BeEmpty())
			Expect(info["Version"]).ToNot(BeEmpty())
		})

		It("routes case-insensitively (lowercase path)", func() {
			w := rawReq("GET", "/system/info/public", "")
			Expect(w.Code).To(Equal(http.StatusOK))
		})
	})

	Describe("GET/POST /System/Ping", func() {
		It("answers GET with a plain-text server name", func() {
			w := rawReq("GET", "/System/Ping", "")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(HavePrefix("text/plain"))
			Expect(w.Body.String()).To(HavePrefix("Navidrome"))
		})

		It("answers POST identically", func() {
			w := rawReq("POST", "/System/Ping", "")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(strings.TrimSpace(w.Body.String())).To(HavePrefix("Navidrome"))
		})
	})

	Describe("GET /QuickConnect/Enabled", func() {
		It("reports QuickConnect disabled", func() {
			w := rawReq("GET", "/QuickConnect/Enabled", "")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(strings.TrimSpace(w.Body.String())).To(Equal("false"))
		})
	})
})
