package nativeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("config endpoint", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	It("rejects non admin users", func() {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		ctx := request.WithUser(req.Context(), model.User{IsAdmin: false})
		getConfig(w, req.WithContext(ctx))
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("returns configuration entries", func() {
		req := httptest.NewRequest("GET", "/config", nil)
		w := httptest.NewRecorder()
		ctx := request.WithUser(req.Context(), model.User{IsAdmin: true})
		getConfig(w, req.WithContext(ctx))
		Expect(w.Code).To(Equal(http.StatusOK))
		var resp configResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
		Expect(resp.ID).To(Equal("config"))

		// Verify that we have both Dev and non-Dev fields
		var hasDevFields = false
		var hasNonDevFields = false
		for _, e := range resp.Config {
			if strings.HasPrefix(e.Key, "Dev") {
				hasDevFields = true
			} else {
				hasNonDevFields = true
			}
		}

		Expect(hasDevFields).To(BeTrue(), "Should have Dev* configuration fields")
		Expect(hasNonDevFields).To(BeTrue(), "Should have non-Dev configuration fields")
		Expect(len(resp.Config)).To(BeNumerically(">", 0), "Should return configuration entries")
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

	Context("redaction functionality", func() {
		It("redacts sensitive values with partial masking for long values", func() {
			// Set up test values
			conf.Server.LastFM.ApiKey = "ba46f0e84a123456"
			conf.Server.Spotify.Secret = "verylongsecret123"

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

			Expect(values).To(HaveKeyWithValue("LastFM.ApiKey", "b**************6"))
			Expect(values).To(HaveKeyWithValue("Spotify.Secret", "v***************3"))
		})

		It("redacts sensitive values with full masking for short values", func() {
			// Set up test values with short secrets
			conf.Server.LastFM.Secret = "short"
			conf.Server.Spotify.ID = "abc123"

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

			Expect(values).To(HaveKeyWithValue("LastFM.Secret", "****"))
			Expect(values).To(HaveKeyWithValue("Spotify.ID", "****"))
		})

		It("fully masks password fields", func() {
			// Set up test values for password fields
			conf.Server.DevAutoCreateAdminPassword = "adminpass123"
			conf.Server.Prometheus.Password = "prometheuspass"

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

			Expect(values).To(HaveKeyWithValue("DevAutoCreateAdminPassword", "****"))
			Expect(values).To(HaveKeyWithValue("Prometheus.Password", "****"))
		})

		It("does not redact non-sensitive values", func() {
			conf.Server.MusicFolder = "/path/to/music"
			conf.Server.Port = 4533

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

			Expect(values).To(HaveKeyWithValue("MusicFolder", "/path/to/music"))
			Expect(values).To(HaveKeyWithValue("Port", "4533"))
		})

		It("handles empty sensitive values", func() {
			conf.Server.LastFM.ApiKey = ""
			conf.Server.PasswordEncryptionKey = ""

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

			// Empty sensitive values should remain empty
			Expect(values["LastFM.ApiKey"]).To(Equal(""))
			Expect(values["PasswordEncryptionKey"]).To(Equal(""))
		})
	})
})

var _ = Describe("redactValue function", func() {
	It("partially masks long sensitive values", func() {
		Expect(redactValue("LastFM.ApiKey", "ba46f0e84a")).To(Equal("b********a"))
		Expect(redactValue("Spotify.Secret", "verylongsecret123")).To(Equal("v***************3"))
	})

	It("fully masks long sensitive values that should be completely hidden", func() {
		Expect(redactValue("PasswordEncryptionKey", "1234567890")).To(Equal("****"))
		Expect(redactValue("DevAutoCreateAdminPassword", "1234567890")).To(Equal("****"))
		Expect(redactValue("Prometheus.Password", "1234567890")).To(Equal("****"))
	})

	It("fully masks short sensitive values", func() {
		Expect(redactValue("LastFM.Secret", "short")).To(Equal("****"))
		Expect(redactValue("Spotify.ID", "abc")).To(Equal("****"))
		Expect(redactValue("PasswordEncryptionKey", "12345")).To(Equal("****"))
		Expect(redactValue("DevAutoCreateAdminPassword", "short")).To(Equal("****"))
		Expect(redactValue("Prometheus.Password", "short")).To(Equal("****"))
	})

	It("does not mask non-sensitive values", func() {
		Expect(redactValue("MusicFolder", "/path/to/music")).To(Equal("/path/to/music"))
		Expect(redactValue("Port", "4533")).To(Equal("4533"))
		Expect(redactValue("SomeOtherField", "secretvalue")).To(Equal("secretvalue"))
	})

	It("handles empty values", func() {
		Expect(redactValue("LastFM.ApiKey", "")).To(Equal(""))
		Expect(redactValue("NonSensitive", "")).To(Equal(""))
	})

	It("handles edge case values", func() {
		Expect(redactValue("LastFM.ApiKey", "a")).To(Equal("****"))
		Expect(redactValue("LastFM.ApiKey", "ab")).To(Equal("****"))
		Expect(redactValue("LastFM.ApiKey", "abcdefg")).To(Equal("a*****g"))
	})
})
