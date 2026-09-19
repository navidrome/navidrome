package jellyfin

import (
	"net"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Discovery", func() {
	var api *Router

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Jellyfin.ServerName = "Test Server"
		conf.Server.Address = "0.0.0.0"
		conf.Server.Port = 4533
		conf.Server.BaseHost = ""
		conf.Server.BaseScheme = ""
		conf.Server.BasePath = ""
		conf.Server.TLSCert = ""
		api = &Router{}
	})

	Describe("discoveryAddress", func() {
		remote := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50000}

		It("uses the BaseURL host and scheme when set", func() {
			conf.Server.BaseScheme = "https"
			conf.Server.BaseHost = "music.example.com"
			conf.Server.BasePath = "/nd"
			Expect(api.discoveryAddress(remote)).To(Equal("https://music.example.com/nd/jellyfin"))
		})

		It("uses a specific bind Address with the Port", func() {
			conf.Server.Address = "192.168.1.10"
			Expect(api.discoveryAddress(remote)).To(Equal("http://192.168.1.10:4533/jellyfin"))
		})

		It("falls back to the interface facing the requester when Address is unspecified", func() {
			Expect(api.discoveryAddress(remote)).To(Equal("http://127.0.0.1:4533/jellyfin"))
		})

		It("falls back to the interface facing the requester when Address is not an IP", func() {
			conf.Server.Address = "unix:/tmp/navidrome.sock"
			Expect(api.discoveryAddress(remote)).To(Equal("http://127.0.0.1:4533/jellyfin"))
		})

		It("advertises https when TLS is configured", func() {
			conf.Server.TLSCert = "/path/cert.pem"
			Expect(api.discoveryAddress(remote)).To(Equal("https://127.0.0.1:4533/jellyfin"))
		})

		It("keeps a path-only BaseURL as the path prefix", func() {
			conf.Server.BasePath = "/music"
			Expect(api.discoveryAddress(remote)).To(Equal("http://127.0.0.1:4533/music/jellyfin"))
		})
	})
})
