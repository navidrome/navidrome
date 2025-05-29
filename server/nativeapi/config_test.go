package nativeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("config endpoint", func() {
	It("rejects non admin users", func() {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		ctx := request.WithUser(req.Context(), model.User{IsAdmin: false})
		getConfig(w, req.WithContext(ctx))
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("returns sorted configuration entries for admins", func() {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		ctx := request.WithUser(req.Context(), model.User{IsAdmin: true})
		getConfig(w, req.WithContext(ctx))
		Expect(w.Code).To(Equal(http.StatusOK))
		var resp configResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.ID).To(Equal("config"))
		keys := make([]string, len(resp.Config))
		for i, e := range resp.Config {
			keys[i] = e.Key
		}
		Expect(sort.StringsAreSorted(keys)).To(BeTrue())
	})

	It("includes flattened struct fields", func() {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		ctx := request.WithUser(req.Context(), model.User{IsAdmin: true})
		getConfig(w, req.WithContext(ctx))
		Expect(w.Code).To(Equal(http.StatusOK))
		var resp configResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		values := map[string]string{}
		for _, e := range resp.Config {
			if s, ok := e.Value.(string); ok {
				values[e.Key] = s
			}
		}
		Expect(values).To(HaveKeyWithValue("Inspect.MaxRequests", "1"))
		Expect(values).To(HaveKeyWithValue("HTTPSecurityHeaders.CustomFrameOptionsValue", "DENY"))
	})

	It("includes the config file path", func() {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		ctx := request.WithUser(req.Context(), model.User{IsAdmin: true})
		getConfig(w, req.WithContext(ctx))
		Expect(w.Code).To(Equal(http.StatusOK))
		var resp configResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.ConfigFile).To(Not(BeEmpty()))
	})

	It("includes environment variable names", func() {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		ctx := request.WithUser(req.Context(), model.User{IsAdmin: true})
		getConfig(w, req.WithContext(ctx))
		Expect(w.Code).To(Equal(http.StatusOK))
		var resp configResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())

		// Create a map to check specific env var mappings
		envVars := map[string]string{}
		for _, e := range resp.Config {
			envVars[e.Key] = e.EnvVar
		}

		Expect(envVars).To(HaveKeyWithValue("MusicFolder", "ND_MUSICFOLDER"))
		Expect(envVars).To(HaveKeyWithValue("Scanner.Enabled", "ND_SCANNER_ENABLED"))
		Expect(envVars).To(HaveKeyWithValue("HTTPSecurityHeaders.CustomFrameOptionsValue", "ND_HTTPSECURITYHEADERS_CUSTOMFRAMEOPTIONSVALUE"))
	})
})
