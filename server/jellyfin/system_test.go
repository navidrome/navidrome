package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("System", func() {
	var api *Router
	BeforeEach(func() { api = &Router{} })

	It("returns public system info without auth", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Jellyfin.ServerName = ""

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Info/Public", nil)
		api.getPublicSystemInfo(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
		var info dto.PublicSystemInfo
		Expect(json.Unmarshal(w.Body.Bytes(), &info)).To(Succeed())
		Expect(info.Id).ToNot(BeEmpty())
		Expect(info.Version).To(Equal(jellyfinVersion))
		Expect(info.ProductName).To(Equal("Jellyfin Server"))
		Expect(info.ServerName).To(HavePrefix("Navidrome"))
	})

	It("returns authenticated system info with the public fields plus library monitor support", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Jellyfin.ServerName = ""

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Info", nil)
		api.getSystemInfo(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("application/json"))
		var info dto.SystemInfo
		Expect(json.Unmarshal(w.Body.Bytes(), &info)).To(Succeed())
		Expect(info.Id).ToNot(BeEmpty())
		Expect(info.Version).To(Equal(jellyfinVersion))
		Expect(info.ProductName).To(Equal("Jellyfin Server"))
		Expect(info.ServerName).To(HavePrefix("Navidrome"))
		Expect(info.SupportsLibraryMonitor).To(BeTrue())
		Expect(info.HasPendingRestart).To(BeFalse())
		Expect(info.IsShuttingDown).To(BeFalse())
	})

	It("advertises a LocalAddress with the request scheme, host and Jellyfin base path", func() {
		DeferCleanup(configtest.SetupConfig())

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Info/Public", nil)
		r.Host = "music.example.com:4599"
		api.getPublicSystemInfo(w, r)

		var info dto.PublicSystemInfo
		Expect(json.Unmarshal(w.Body.Bytes(), &info)).To(Succeed())
		// Jellify connecting over HTTP sets its server base URL from LocalAddress; without it the
		// SDK `api` is undefined and sign-in crashes. It must include the /jellyfin mount path.
		Expect(info.LocalAddress).To(Equal("http://music.example.com:4599/jellyfin"))
	})

	It("responds to ping with the server name as plain text", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.Jellyfin.ServerName = ""

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Ping", nil)
		api.ping(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Header().Get("Content-Type")).To(ContainSubstring("text/plain"))
		// Plain text, not a JSON-quoted string: Jellyfin clients expect the bare server name.
		Expect(w.Body.String()).To(HavePrefix("Navidrome"))
	})

	DescribeTable("reports the caller's network location on /System/Endpoint",
		func(remoteAddr string, isLocal, isInNetwork bool) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/System/Endpoint", nil)
			r.RemoteAddr = remoteAddr
			api.getEndpointInfo(w, r)

			Expect(w.Code).To(Equal(http.StatusOK))
			// Finamp's connection test only probes for the key's presence, so it must always be emitted.
			Expect(w.Body.String()).To(ContainSubstring(`"IsInNetwork"`))
			var info dto.EndPointInfo
			Expect(json.Unmarshal(w.Body.Bytes(), &info)).To(Succeed())
			Expect(info.IsLocal).To(Equal(isLocal))
			Expect(info.IsInNetwork).To(Equal(isInNetwork))
		},
		Entry("loopback", "127.0.0.1:12345", true, true),
		Entry("IPv6 loopback", "[::1]:12345", true, true),
		Entry("LAN address", "192.168.1.20:54321", false, true),
		Entry("bare IP, as left by the RealIP middleware", "10.0.0.5", false, true),
		Entry("IPv4-mapped IPv6 LAN address", "[::ffff:172.16.0.9]:80", false, true),
		Entry("IPv6 link-local", "[fe80::1]:80", false, true),
		Entry("IPv6 unique-local", "[fd00::1]:80", false, true),
		// Jellyfin's default LAN set omits 169.254.0.0/16, so we do too.
		Entry("IPv4 link-local", "169.254.1.1:80", false, false),
		Entry("public address", "8.8.8.8:443", false, false),
		Entry("unparseable address", "not-an-ip", false, false),
	)

	It("reports IsLocal when the caller shares the connection's local address", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Endpoint", nil)
		r.RemoteAddr = "192.168.1.20:54321"
		ctx := context.WithValue(r.Context(), http.LocalAddrContextKey,
			&net.TCPAddr{IP: net.ParseIP("192.168.1.20"), Port: 4533})
		api.getEndpointInfo(w, r.WithContext(ctx))

		var info dto.EndPointInfo
		Expect(json.Unmarshal(w.Body.Bytes(), &info)).To(Succeed())
		Expect(info.IsLocal).To(BeTrue())
	})

	It("does not report IsLocal for a different host on the same LAN", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Endpoint", nil)
		r.RemoteAddr = "192.168.1.99:54321"
		ctx := context.WithValue(r.Context(), http.LocalAddrContextKey,
			&net.TCPAddr{IP: net.ParseIP("192.168.1.20"), Port: 4533})
		api.getEndpointInfo(w, r.WithContext(ctx))

		var info dto.EndPointInfo
		Expect(json.Unmarshal(w.Body.Bytes(), &info)).To(Succeed())
		Expect(info.IsLocal).To(BeFalse())
		Expect(info.IsInNetwork).To(BeTrue())
	})

	It("reports quick connect as disabled", func() {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/QuickConnect/Enabled", nil)
		api.quickConnectEnabled(w, r)

		Expect(w.Code).To(Equal(http.StatusOK))
		var enabled bool
		Expect(json.Unmarshal(w.Body.Bytes(), &enabled)).To(Succeed())
		Expect(enabled).To(BeFalse())
	})

	Context("serverID with a real DataStore", func() {
		var ctx context.Context
		var ds *tests.MockDataStore

		BeforeEach(func() {
			ctx = context.Background()
			ds = &tests.MockDataStore{}
		})

		It("persists the generated id so it can be read back by another Router sharing the same DataStore", func() {
			first := &Router{ds: ds}
			id := first.serverID(ctx)
			Expect(id).ToNot(BeEmpty())

			second := &Router{ds: ds}
			Expect(second.serverID(ctx)).To(Equal(id))
		})

		It("memoizes the id across repeated calls on the same Router", func() {
			r := &Router{ds: ds}
			id := r.serverID(ctx)
			Expect(r.serverID(ctx)).To(Equal(id))
			Expect(r.serverID(ctx)).To(Equal(id))
		})

		It("does not overwrite or pin over a stored id when the property read fails transiently", func() {
			Expect(ds.Property(ctx).Put(consts.JellyfinServerIDKey, "6ba7b8109dad11d180b400c04fd430c8")).To(Succeed())

			r := &Router{ds: ds}
			props := ds.Property(ctx).(*tests.MockedPropertyRepo)
			props.Error = errors.New("database is locked")
			degraded := r.serverID(ctx)
			Expect(degraded).ToNot(BeEmpty())
			Expect(degraded).ToNot(Equal("6ba7b8109dad11d180b400c04fd430c8")) // temporary value, not the (unreadable) stored one
			props.Error = nil

			// Once the DB recovers, the stored id is intact and served again.
			Expect(r.serverID(ctx)).To(Equal("6ba7b8109dad11d180b400c04fd430c8"))
			stored, err := ds.Property(ctx).Get(consts.JellyfinServerIDKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(stored).To(Equal("6ba7b8109dad11d180b400c04fd430c8"))
		})

		It("returns Jellyfin's no-dash GUID form", func() {
			r := &Router{ds: ds}
			Expect(r.serverID(ctx)).To(MatchRegexp("^[0-9a-f]{32}$"))
		})

		It("strips dashes from an already-persisted id", func() {
			Expect(ds.Property(ctx).Put(
				consts.JellyfinServerIDKey, "1b4e28ba-2fa1-11d2-883f-0016d3cca427")).To(Succeed())
			r := &Router{ds: ds}
			Expect(r.serverID(ctx)).To(Equal("1b4e28ba2fa111d2883f0016d3cca427"))
		})
	})
})
