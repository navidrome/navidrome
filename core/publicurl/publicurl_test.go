package publicurl_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPublicURL(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Public URL Suite")
}

var _ = Describe("Public URL Utilities", func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	Describe("PublicURL", func() {
		When("ShareURL is set", func() {
			BeforeEach(func() {
				conf.Server.ShareURL = "https://share.example.com"
			})

			It("uses ShareURL as the base", func() {
				r, _ := http.NewRequest("GET", "http://localhost/test", nil)
				result := publicurl.PublicURL(r, "/path/to/resource", nil)
				Expect(result).To(Equal("https://share.example.com/path/to/resource"))
			})

			It("includes query parameters", func() {
				r, _ := http.NewRequest("GET", "http://localhost/test", nil)
				params := url.Values{"size": []string{"300"}, "format": []string{"png"}}
				result := publicurl.PublicURL(r, "/image/123", params)
				Expect(result).To(ContainSubstring("https://share.example.com/image/123"))
				Expect(result).To(ContainSubstring("size=300"))
				Expect(result).To(ContainSubstring("format=png"))
			})

			It("works without a request", func() {
				result := publicurl.PublicURL(nil, "/path/to/resource", nil)
				Expect(result).To(Equal("https://share.example.com/path/to/resource"))
			})
		})

		When("ShareURL is not set", func() {
			BeforeEach(func() {
				conf.Server.ShareURL = ""
			})

			It("falls back to AbsoluteURL with request", func() {
				r, _ := http.NewRequest("GET", "https://myserver.com/test", nil)
				r.Host = "myserver.com"
				result := publicurl.PublicURL(r, "/path/to/resource", nil)
				Expect(result).To(Equal("https://myserver.com/path/to/resource"))
			})

			It("falls back to localhost without request", func() {
				result := publicurl.PublicURL(nil, "/path/to/resource", nil)
				Expect(result).To(Equal("http://localhost/path/to/resource"))
			})
		})
	})

	Describe("AbsoluteURL", func() {
		When("BaseHost is set", func() {
			BeforeEach(func() {
				conf.Server.BaseHost = "configured.example.com"
				conf.Server.BaseScheme = "https"
				conf.Server.BasePath = ""
			})

			It("uses BaseHost and BaseScheme", func() {
				r, _ := http.NewRequest("GET", "http://localhost/test", nil)
				result := publicurl.AbsoluteURL(r, "/path/to/resource", nil)
				Expect(result).To(Equal("https://configured.example.com/path/to/resource"))
			})

			It("defaults to http scheme if BaseScheme is empty", func() {
				conf.Server.BaseScheme = ""
				r, _ := http.NewRequest("GET", "http://localhost/test", nil)
				result := publicurl.AbsoluteURL(r, "/path/to/resource", nil)
				Expect(result).To(Equal("http://configured.example.com/path/to/resource"))
			})
		})

		When("BaseHost is not set", func() {
			BeforeEach(func() {
				conf.Server.BaseHost = ""
				conf.Server.BasePath = ""
			})

			It("extracts host from request", func() {
				r, _ := http.NewRequest("GET", "https://request.example.com/test", nil)
				r.Host = "request.example.com"
				result := publicurl.AbsoluteURL(r, "/path/to/resource", nil)
				Expect(result).To(Equal("https://request.example.com/path/to/resource"))
			})

			It("falls back to localhost without request", func() {
				result := publicurl.AbsoluteURL(nil, "/path/to/resource", nil)
				Expect(result).To(Equal("http://localhost/path/to/resource"))
			})
		})

		When("BasePath is set", func() {
			BeforeEach(func() {
				conf.Server.BasePath = "/navidrome"
				conf.Server.BaseHost = "example.com"
				conf.Server.BaseScheme = "https"
			})

			It("prepends BasePath to the URL", func() {
				r, _ := http.NewRequest("GET", "http://localhost/test", nil)
				result := publicurl.AbsoluteURL(r, "/path/to/resource", nil)
				Expect(result).To(Equal("https://example.com/navidrome/path/to/resource"))
			})
		})

		It("passes through absolute URLs unchanged", func() {
			r, _ := http.NewRequest("GET", "http://localhost/test", nil)
			result := publicurl.AbsoluteURL(r, "https://other.example.com/path", nil)
			Expect(result).To(Equal("https://other.example.com/path"))
		})

		It("includes query parameters", func() {
			conf.Server.BaseHost = "example.com"
			conf.Server.BaseScheme = "https"
			r, _ := http.NewRequest("GET", "http://localhost/test", nil)
			params := url.Values{"key": []string{"value"}}
			result := publicurl.AbsoluteURL(r, "/path", params)
			Expect(result).To(Equal("https://example.com/path?key=value"))
		})
	})

	Describe("ImageURL", func() {
		BeforeEach(func() {
			conf.Server.ShareURL = "https://share.example.com"
			// Initialize JWT auth for token generation
			auth.TokenAuth = jwtauth.New("HS256", []byte("test secret"), nil)
		})

		It("generates a URL with the artwork token", func() {
			artID := model.NewArtworkID(model.KindAlbumArtwork, "album-123", nil)
			result := publicurl.ImageURL(nil, artID, 0)
			Expect(result).To(HavePrefix("https://share.example.com/share/img/"))
		})

		It("includes size parameter when provided", func() {
			artID := model.NewArtworkID(model.KindArtistArtwork, "artist-1", nil)
			result := publicurl.ImageURL(nil, artID, 300)
			Expect(result).To(ContainSubstring("size=300"))
		})

		It("omits size parameter when zero", func() {
			artID := model.NewArtworkID(model.KindMediaFileArtwork, "track-1", nil)
			result := publicurl.ImageURL(nil, artID, 0)
			Expect(result).ToNot(ContainSubstring("size="))
		})
	})
})
