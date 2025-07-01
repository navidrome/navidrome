package nativeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("getConfig", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	Context("when user is not admin", func() {
		It("returns unauthorized", func() {
			req := httptest.NewRequest("GET", "/config", nil)
			w := httptest.NewRecorder()
			ctx := request.WithUser(req.Context(), model.User{IsAdmin: false})

			getConfig(w, req.WithContext(ctx))

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})
	})

	Context("when user is admin", func() {
		It("returns config successfully", func() {
			req := httptest.NewRequest("GET", "/config", nil)
			w := httptest.NewRecorder()
			ctx := request.WithUser(req.Context(), model.User{IsAdmin: true})

			getConfig(w, req.WithContext(ctx))

			Expect(w.Code).To(Equal(http.StatusOK))
			var resp configResponse
			Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.ID).To(Equal("config"))
			Expect(resp.ConfigFile).To(Equal(conf.Server.ConfigFile))
			Expect(resp.Config).ToNot(BeEmpty())
		})

		It("redacts sensitive fields", func() {
			conf.Server.LastFM.ApiKey = "secretapikey123"
			conf.Server.Spotify.Secret = "spotifysecret456"
			conf.Server.PasswordEncryptionKey = "encryptionkey789"
			conf.Server.DevAutoCreateAdminPassword = "adminpassword123"
			conf.Server.Prometheus.Password = "prometheuspass"

			req := httptest.NewRequest("GET", "/config", nil)
			w := httptest.NewRecorder()
			ctx := request.WithUser(req.Context(), model.User{IsAdmin: true})
			getConfig(w, req.WithContext(ctx))

			Expect(w.Code).To(Equal(http.StatusOK))
			var resp configResponse
			Expect(json.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())

			// Check LastFM.ApiKey (partially masked)
			lastfm, ok := resp.Config["LastFM"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(lastfm["ApiKey"]).To(Equal("s*************3"))

			// Check Spotify.Secret (partially masked)
			spotify, ok := resp.Config["Spotify"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(spotify["Secret"]).To(Equal("s**************6"))

			// Check PasswordEncryptionKey (fully masked)
			Expect(resp.Config["PasswordEncryptionKey"]).To(Equal("****"))

			// Check DevAutoCreateAdminPassword (fully masked)
			Expect(resp.Config["DevAutoCreateAdminPassword"]).To(Equal("****"))

			// Check Prometheus.Password (fully masked)
			prometheus, ok := resp.Config["Prometheus"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(prometheus["Password"]).To(Equal("****"))
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

			// Check LastFM.ApiKey - should be preserved because it's sensitive
			lastfm, ok := resp.Config["LastFM"].(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(lastfm["ApiKey"]).To(Equal(""))

			// Empty sensitive values should remain empty - should be preserved because it's sensitive
			Expect(resp.Config["PasswordEncryptionKey"]).To(Equal(""))
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
