package plugins

import (
	"context"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/plugins/host/artwork"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ArtworkService", func() {
	var svc *artworkServiceImpl

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		// Setup auth for tests
		auth.TokenAuth = jwtauth.New("HS256", []byte("super secret"), nil)
		svc = &artworkServiceImpl{}
	})

	Context("with ShareURL configured", func() {
		BeforeEach(func() {
			conf.Server.ShareURL = "https://music.example.com"
		})

		It("returns artist artwork URL", func() {
			resp, err := svc.GetArtistUrl(context.Background(), &artwork.GetArtworkUrlRequest{Id: "123", Size: 300})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Url).To(ContainSubstring("https://music.example.com"))
			Expect(resp.Url).To(ContainSubstring("size=300"))
		})

		It("returns album artwork URL", func() {
			resp, err := svc.GetAlbumUrl(context.Background(), &artwork.GetArtworkUrlRequest{Id: "456"})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Url).To(ContainSubstring("https://music.example.com"))
		})

		It("returns track artwork URL", func() {
			resp, err := svc.GetTrackUrl(context.Background(), &artwork.GetArtworkUrlRequest{Id: "789", Size: 150})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Url).To(ContainSubstring("https://music.example.com"))
			Expect(resp.Url).To(ContainSubstring("size=150"))
		})
	})

	Context("without ShareURL configured", func() {
		It("returns localhost URLs", func() {
			resp, err := svc.GetArtistUrl(context.Background(), &artwork.GetArtworkUrlRequest{Id: "123"})
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Url).To(ContainSubstring("http://localhost"))
		})
	})
})
