package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type failingConn struct {
	net.PacketConn
	closed bool
}

func (c *failingConn) ReadFrom([]byte) (int, net.Addr, error) {
	return 0, nil, errors.New("read failed")
}

func (c *failingConn) Close() error {
	c.closed = true
	return nil
}

var _ = Describe("Discovery", func() {
	var d *Discovery

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
		d = &Discovery{}
	})

	DescribeTable("discoveryAddress",
		func(setup func(), expected string) {
			setup()
			remote := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50000}
			Expect(discoveryAddress(context.Background(), remote)).To(Equal(expected))
		},
		Entry("uses the BaseURL host and scheme when set", func() {
			conf.Server.BaseScheme = "https"
			conf.Server.BaseHost = "music.example.com"
			conf.Server.BasePath = "/nd"
		}, "https://music.example.com/nd/jellyfin"),
		Entry("uses a specific bind Address with the Port", func() {
			conf.Server.Address = "192.168.1.10"
		}, "http://192.168.1.10:4533/jellyfin"),
		Entry("falls back to the interface facing the requester when Address is unspecified", func() {},
			"http://127.0.0.1:4533/jellyfin"),
		Entry("falls back to the interface facing the requester when Address is empty", func() {
			conf.Server.Address = ""
		}, "http://127.0.0.1:4533/jellyfin"),
		Entry("advertises https when TLS is configured", func() {
			conf.Server.TLSCert = "/path/cert.pem"
			conf.Server.TLSKey = "/path/key.pem"
		}, "https://127.0.0.1:4533/jellyfin"),
		Entry("advertises http when only the TLS cert is configured", func() {
			conf.Server.TLSCert = "/path/cert.pem"
		}, "http://127.0.0.1:4533/jellyfin"),
		Entry("keeps a path-only BaseURL as the path prefix", func() {
			conf.Server.BasePath = "/music"
		}, "http://127.0.0.1:4533/music/jellyfin"),
		Entry("does not double the slash when BasePath has a trailing slash", func() {
			conf.Server.BasePath = "/music/"
		}, "http://127.0.0.1:4533/music/jellyfin"),
	)

	DescribeTable("hasAdvertisableAddress",
		func(address, baseHost string, expected bool) {
			conf.Server.Address = address
			conf.Server.BaseHost = baseHost
			Expect(hasAdvertisableAddress()).To(Equal(expected))
		},
		Entry("TCP listener", "0.0.0.0", "", true),
		Entry("unix socket without a BaseURL host", "unix:/tmp/navidrome.sock", "", false),
		Entry("unix socket behind a proxy named by BaseURL", "unix:/tmp/navidrome.sock", "music.example.com", true),
	)

	It("closes the connection when the read loop fails", func() {
		fake := &failingConn{}
		d.ServeOn(context.Background(), fake)
		Expect(fake.closed).To(BeTrue())
	})

	Describe("ServeOn", func() {
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
				d.ServeOn(ctx, server)
			}()
			DeferCleanup(func() {
				cancel()
				Eventually(done).Should(BeClosed())
			})
		})

		send := func(msg string) {
			_, err := client.WriteTo([]byte(msg), server.LocalAddr())
			Expect(err).ToNot(HaveOccurred())
		}
		receive := func() ([]byte, error) {
			Expect(client.SetReadDeadline(time.Now().Add(time.Second))).To(Succeed())
			buf := make([]byte, 1024)
			n, _, err := client.ReadFrom(buf)
			return buf[:n], err
		}

		It("answers the discovery query with the server identity", func() {
			send("who is JellyfinServer?")
			res, err := receive()
			Expect(err).ToNot(HaveOccurred())

			var info discoveryInfo
			Expect(json.Unmarshal(res, &info)).To(Succeed())
			Expect(info.Address).To(Equal("http://127.0.0.1:4533/jellyfin"))
			Expect(info.Id).To(Equal(d.serverID(context.Background())))
			Expect(info.Name).To(Equal("Test Server"))
			Expect(string(res)).To(ContainSubstring(`"EndpointAddress":null`))
		})

		It("matches the query case-insensitively", func() {
			send("WHO IS JELLYFINSERVER?")
			_, err := receive()
			Expect(err).ToNot(HaveOccurred())
		})

		// The loop is serial, so a reply to the first packet would arrive before the second's.
		It("ignores unrelated packets", func() {
			send("who is PlexServer?")
			send("who is JellyfinServer?")
			_, err := receive()
			Expect(err).ToNot(HaveOccurred())
			Expect(client.SetReadDeadline(time.Now().Add(50 * time.Millisecond))).To(Succeed())
			_, _, err = client.ReadFrom(make([]byte, 1024))
			Expect(errors.Is(err, os.ErrDeadlineExceeded)).To(BeTrue())
		})

		It("stops and closes the socket when the context is cancelled", func() {
			cancel()
			Eventually(done).Should(BeClosed())
			_, _, err := server.ReadFrom(make([]byte, 1))
			Expect(err).To(HaveOccurred())
		})
	})
})
