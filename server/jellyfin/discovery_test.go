package jellyfin

import (
	"context"
	"encoding/json"
	"net"
	"time"

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
		conf.Server.TLSKey = ""
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
			conf.Server.TLSKey = "/path/key.pem"
			Expect(api.discoveryAddress(remote)).To(Equal("https://127.0.0.1:4533/jellyfin"))
		})

		It("advertises http when only the TLS cert is configured", func() {
			conf.Server.TLSCert = "/path/cert.pem"
			Expect(api.discoveryAddress(remote)).To(Equal("http://127.0.0.1:4533/jellyfin"))
		})

		It("keeps a path-only BaseURL as the path prefix", func() {
			conf.Server.BasePath = "/music"
			Expect(api.discoveryAddress(remote)).To(Equal("http://127.0.0.1:4533/music/jellyfin"))
		})

		It("does not double the slash when BasePath has a trailing slash", func() {
			conf.Server.BasePath = "/music/"
			Expect(api.discoveryAddress(remote)).To(Equal("http://127.0.0.1:4533/music/jellyfin"))
		})
	})

	Describe("ServeDiscoveryOn", func() {
		var (
			server net.PacketConn
			client net.PacketConn
			cancel context.CancelFunc
			done   chan struct{}
		)

		BeforeEach(func() {
			var err error
			server, err = net.ListenPacket("udp4", "127.0.0.1:0")
			Expect(err).ToNot(HaveOccurred())
			client, err = net.ListenPacket("udp4", "127.0.0.1:0")
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(client.Close)

			var ctx context.Context
			ctx, cancel = context.WithCancel(context.Background())
			done = make(chan struct{})
			go func() {
				defer close(done)
				api.ServeDiscoveryOn(ctx, server)
			}()
			DeferCleanup(func() {
				cancel()
				Eventually(done).Should(BeClosed())
			})
		})

		ask := func(msg string, wait time.Duration) ([]byte, error) {
			_, err := client.WriteTo([]byte(msg), server.LocalAddr())
			Expect(err).ToNot(HaveOccurred())
			Expect(client.SetReadDeadline(time.Now().Add(wait))).To(Succeed())
			buf := make([]byte, 1024)
			n, _, err := client.ReadFrom(buf)
			return buf[:n], err
		}

		It("answers the discovery query with the server identity", func() {
			res, err := ask("who is JellyfinServer?", time.Second)
			Expect(err).ToNot(HaveOccurred())

			var info discoveryInfo
			Expect(json.Unmarshal(res, &info)).To(Succeed())
			Expect(info.Address).To(Equal("http://127.0.0.1:4533/jellyfin"))
			Expect(info.Id).To(Equal(api.serverID(context.Background())))
			Expect(info.Name).To(Equal("Test Server"))
			Expect(string(res)).To(ContainSubstring(`"EndpointAddress":null`))
		})

		It("matches the query case-insensitively", func() {
			_, err := ask("WHO IS JELLYFINSERVER?", time.Second)
			Expect(err).ToNot(HaveOccurred())
		})

		It("ignores unrelated packets", func() {
			_, err := ask("who is PlexServer?", 200*time.Millisecond)
			Expect(err).To(HaveOccurred())
		})

		It("stops and closes the socket when the context is cancelled", func() {
			cancel()
			Eventually(done).Should(BeClosed())
			_, _, err := server.ReadFrom(make([]byte, 1))
			Expect(err).To(HaveOccurred())
		})
	})
})
