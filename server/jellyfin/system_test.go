package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
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
			Expect(ds.Property(ctx).Put(consts.JellyfinServerIDKey, "stable-id")).To(Succeed())

			r := &Router{ds: ds}
			props := ds.Property(ctx).(*tests.MockedPropertyRepo)
			props.Error = errors.New("database is locked")
			degraded := r.serverID(ctx)
			Expect(degraded).ToNot(BeEmpty())
			Expect(degraded).ToNot(Equal("stable-id")) // temporary value, not the (unreadable) stored one
			props.Error = nil

			// Once the DB recovers, the stored id is intact and served again.
			Expect(r.serverID(ctx)).To(Equal("stable-id"))
			stored, err := ds.Property(ctx).Get(consts.JellyfinServerIDKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(stored).To(Equal("stable-id"))
		})
	})
})
