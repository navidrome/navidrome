package publicurl_test

import (
	"context"
	"net/url"
	"testing"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
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
				result := publicurl.PublicURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("https://share.example.com/path/to/resource"))
			})

			It("includes query parameters", func() {
				params := url.Values{"size": []string{"300"}, "format": []string{"png"}}
				result := publicurl.PublicURL(context.Background(), "/image/123", params)
				Expect(result).To(ContainSubstring("https://share.example.com/image/123"))
				Expect(result).To(ContainSubstring("size=300"))
				Expect(result).To(ContainSubstring("format=png"))
			})

			It("works without a request", func() {
				result := publicurl.PublicURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("https://share.example.com/path/to/resource"))
			})
		})

		When("ShareURL includes a path", func() {
			BeforeEach(func() {
				conf.Server.ShareURL = "https://example.com/navi"
			})

			It("prepends the ShareURL path to the resource", func() {
				result := publicurl.PublicURL(context.Background(), "/share/img/hash", nil)
				Expect(result).To(Equal("https://example.com/navi/share/img/hash"))
			})

			It("prepends the ShareURL path and includes query parameters", func() {
				params := url.Values{"size": []string{"600"}}
				result := publicurl.PublicURL(context.Background(), "/share/img/hash", params)
				Expect(result).To(Equal("https://example.com/navi/share/img/hash?size=600"))
			})

			It("handles trailing slash in ShareURL path", func() {
				conf.Server.ShareURL = "https://example.com/navi/"
				result := publicurl.PublicURL(context.Background(), "/share/img/hash", nil)
				Expect(result).To(Equal("https://example.com/navi/share/img/hash"))
			})
		})

		When("ShareURL is not set", func() {
			BeforeEach(func() {
				conf.Server.ShareURL = ""
			})

			It("falls back to AbsoluteURL with request", func() {
				ctx := request.WithServerAddress(context.Background(), "https", "myserver.com")
				result := publicurl.PublicURL(ctx, "/path/to/resource", nil)
				Expect(result).To(Equal("https://myserver.com/path/to/resource"))
			})

			It("falls back to localhost on the configured port without request", func() {
				conf.Server.Port = 4533
				result := publicurl.PublicURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("http://localhost:4533/path/to/resource"))
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
				result := publicurl.AbsoluteURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("https://configured.example.com/path/to/resource"))
			})

			It("defaults to http scheme if BaseScheme is empty", func() {
				conf.Server.BaseScheme = ""
				result := publicurl.AbsoluteURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("http://configured.example.com/path/to/resource"))
			})
		})

		When("BaseHost is not set", func() {
			BeforeEach(func() {
				conf.Server.BaseHost = ""
				conf.Server.BasePath = ""
			})

			It("extracts host from request", func() {
				ctx := request.WithServerAddress(context.Background(), "https", "request.example.com")
				result := publicurl.AbsoluteURL(ctx, "/path/to/resource", nil)
				Expect(result).To(Equal("https://request.example.com/path/to/resource"))
			})

			It("falls back to localhost on the configured port without request", func() {
				conf.Server.Port = 4533
				result := publicurl.AbsoluteURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("http://localhost:4533/path/to/resource"))
			})

			It("uses a non-default port in the fallback", func() {
				conf.Server.Port = 8080
				result := publicurl.AbsoluteURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("http://localhost:8080/path/to/resource"))
			})

			It("uses https in the fallback when TLS is configured", func() {
				conf.Server.Port = 4533
				conf.Server.TLSCert = "cert.pem"
				conf.Server.TLSKey = "key.pem"
				result := publicurl.AbsoluteURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("https://localhost:4533/path/to/resource"))
			})

			It("stays on http when only the certificate is configured", func() {
				conf.Server.Port = 4533
				conf.Server.TLSCert = "cert.pem"
				result := publicurl.AbsoluteURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("http://localhost:4533/path/to/resource"))
			})
		})

		When("BasePath is set", func() {
			BeforeEach(func() {
				conf.Server.BasePath = "/navidrome"
				conf.Server.BaseHost = "example.com"
				conf.Server.BaseScheme = "https"
			})

			It("prepends BasePath to the URL", func() {
				result := publicurl.AbsoluteURL(context.Background(), "/path/to/resource", nil)
				Expect(result).To(Equal("https://example.com/navidrome/path/to/resource"))
			})
		})

		It("passes through absolute URLs unchanged", func() {
			result := publicurl.AbsoluteURL(context.Background(), "https://other.example.com/path", nil)
			Expect(result).To(Equal("https://other.example.com/path"))
		})

		It("includes query parameters", func() {
			conf.Server.BaseHost = "example.com"
			conf.Server.BaseScheme = "https"
			params := url.Values{"key": []string{"value"}}
			result := publicurl.AbsoluteURL(context.Background(), "/path", params)
			Expect(result).To(Equal("https://example.com/path?key=value"))
		})
	})

	Describe("ImageURL", func() {
		BeforeEach(func() {
			conf.Server.ShareURL = "https://share.example.com"
			// Initialize JWT auth for token generation
			auth.PublicTokenAuth = jwtauth.New("HS256", []byte("test secret"), nil)
		})

		It("generates a URL with the artwork token", func() {
			artID := model.NewArtworkID(model.KindAlbumArtwork, "album-123", nil)
			result := publicurl.ImageURL(context.Background(), artID, 0)
			Expect(result).To(HavePrefix("https://share.example.com/share/img/"))
		})

		It("includes size parameter when provided", func() {
			artID := model.NewArtworkID(model.KindArtistArtwork, "artist-1", nil)
			result := publicurl.ImageURL(context.Background(), artID, 300)
			Expect(result).To(ContainSubstring("size=300"))
		})

		It("omits size parameter when zero", func() {
			artID := model.NewArtworkID(model.KindMediaFileArtwork, "track-1", nil)
			result := publicurl.ImageURL(context.Background(), artID, 0)
			Expect(result).ToNot(ContainSubstring("size="))
		})
	})

	Describe("ImageURL address precedence", func() {
		var artID model.ArtworkID

		BeforeEach(func() {
			auth.PublicTokenAuth = jwtauth.New("HS256", []byte("test secret"), nil)
			artID = model.NewArtworkID(model.KindMediaFileArtwork, "track-1", nil)
		})

		It("uses the address of the request that triggered the call", func() {
			ctx := request.WithServerAddress(context.Background(), "https", "music.example.com")

			result := publicurl.ImageURL(ctx, artID, 300)
			Expect(result).To(HavePrefix("https://music.example.com/share/img/"))
			Expect(result).To(ContainSubstring("size=300"))
		})

		It("prefers ShareURL over the address in the context", func() {
			conf.Server.ShareURL = "https://share.example.com"
			ctx := request.WithServerAddress(context.Background(), "https", "music.example.com")

			result := publicurl.ImageURL(ctx, artID, 0)
			Expect(result).To(HavePrefix("https://share.example.com/share/img/"))
		})

		It("falls back to localhost on the configured port when no address is available", func() {
			conf.Server.Port = 4533
			result := publicurl.ImageURL(context.Background(), artID, 0)
			Expect(result).To(HavePrefix("http://localhost:4533/share/img/"))
		})
	})
})
