package e2e

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/navidrome/navidrome/server/jellyfin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Auto discovery", func() {
	BeforeEach(func() { setupTestDB() })

	It("advertises the same Id and Name as /System/Info/Public", func() {
		server, err := net.ListenPacket("udp4", "127.0.0.1:0")
		Expect(err).ToNot(HaveOccurred())
		client, err := net.ListenPacket("udp4", "127.0.0.1:0")
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(client.Close)

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			jellyfin.NewDiscovery(ds).ServeOn(ctx, server)
		}()
		DeferCleanup(func() {
			cancel()
			Eventually(done).Should(BeClosed())
		})

		_, err = client.WriteTo([]byte("who is JellyfinServer?"), server.LocalAddr())
		Expect(err).ToNot(HaveOccurred())
		Expect(client.SetReadDeadline(time.Now().Add(time.Second))).To(Succeed())
		buf := make([]byte, 1024)
		n, _, err := client.ReadFrom(buf)
		Expect(err).ToNot(HaveOccurred())
		var reply map[string]any
		Expect(json.Unmarshal(buf[:n], &reply)).To(Succeed())

		var pub map[string]any
		parseInto(rawReq("GET", "/System/Info/Public", ""), &pub)
		Expect(reply["Id"]).To(Equal(pub["Id"]))
		Expect(reply["Name"]).To(Equal(pub["ServerName"]))
		Expect(reply["Address"]).To(HaveSuffix("/jellyfin"))
	})
})
