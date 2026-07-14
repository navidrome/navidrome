package jellyfin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("authenticate middleware", func() {
	var api *Router
	var ds *tests.MockDataStore
	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		auth.Init(ds)
		ur := ds.User(context.Background()).(*tests.MockedUserRepo)
		Expect(ur.Put(&model.User{ID: "u1", UserName: "alice", NewPassword: "secret"})).To(Succeed())
		api = &Router{ds: ds}
	})

	tokenFor := func(name string) string {
		t, err := auth.CreateToken(&model.User{ID: "u1", UserName: name})
		Expect(err).ToNot(HaveOccurred())
		return t
	}

	It("passes with a valid X-Emby-Token and injects the user", func() {
		var gotUser model.User
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUser, _ = request.UserFrom(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("X-Emby-Token", tokenFor("alice"))
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(gotUser.UserName).To(Equal("alice"))
	})

	It("passes with the recommended Authorization: MediaBrowser scheme and injects the user", func() {
		var gotUser model.User
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotUser, _ = request.UserFrom(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("Authorization", `MediaBrowser Token="`+tokenFor("alice")+`", Client="Test", DeviceId="dev1"`)
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(gotUser.UserName).To(Equal("alice"))
	})

	It("rejects a missing token with 401", func() {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})

	It("rejects a garbage token with 401 and does not call next", func() {
		nextCalled := false
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextCalled = true
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("X-Emby-Token", "not-a-jwt")
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
		Expect(nextCalled).To(BeFalse())
	})

	It("rejects a valid token whose subject user does not exist with 401", func() {
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
		t, err := auth.CreateToken(&model.User{ID: "x", UserName: "ghost"})
		Expect(err).ToNot(HaveOccurred())

		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("X-Emby-Token", t)
		api.authenticate(next).ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusUnauthorized))
	})
})

var _ = Describe("withPlayer middleware", func() {
	var api *Router
	var players *fakePlayers

	BeforeEach(func() {
		players = &fakePlayers{}
		api = &Router{ds: &tests.MockDataStore{}, players: players}
	})

	callWith := func() (model.Player, model.Transcoding, bool) {
		var gotPlayer model.Player
		var gotTrc model.Transcoding
		var hasTrc bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPlayer, _ = request.PlayerFrom(r.Context())
			gotTrc, hasTrc = request.TranscodingFrom(r.Context())
			w.WriteHeader(http.StatusOK)
		})
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/Audio/s1/stream", nil)
		r.Header.Set("X-Emby-Authorization", `MediaBrowser Client="Finamp", Device="Pixel", DeviceId="dev1", Version="1.0"`)
		api.withPlayer(next).ServeHTTP(w, r)
		return gotPlayer, gotTrc, hasTrc
	}

	It("injects the registered player into the context", func() {
		player, _, hasTrc := callWith()
		Expect(player.ID).To(Equal("dev1"))
		Expect(hasTrc).To(BeFalse())
	})

	It("injects the player's server-forced transcoding into the context", func() {
		players.trc = &model.Transcoding{ID: "t1", TargetFormat: "opus"}
		_, trc, hasTrc := callWith()
		Expect(hasTrc).To(BeTrue())
		Expect(trc.TargetFormat).To(Equal("opus"))
	})
})

var _ = Describe("tokenFromRequest", func() {
	It("accepts the recommended Authorization: MediaBrowser scheme", func() {
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("Authorization", `MediaBrowser Token="tok123", Client="Finamp", Device="Pixel", DeviceId="dev1", Version="1.0"`)
		Expect(tokenFromRequest(r)).To(Equal("tok123"))
	})

	It("prefers the Authorization scheme token over deprecated token headers", func() {
		r := httptest.NewRequest("GET", "/Items", nil)
		r.Header.Set("Authorization", `MediaBrowser Token="scheme-token"`)
		r.Header.Set("X-Emby-Token", "legacy-token")
		Expect(tokenFromRequest(r)).To(Equal("scheme-token"))
	})

	It("accepts the lowercase api_key query param", func() {
		r := httptest.NewRequest("GET", "/Items/s1/File?api_key=tok123", nil)
		Expect(tokenFromRequest(r)).To(Equal("tok123"))
	})

	It("accepts a PascalCase ApiKey query param once normalizeQueryKeys has folded it", func() {
		r := httptest.NewRequest("GET", "/Items/s1/File?ApiKey=tok123", nil)
		var got string
		invoke(func(_ http.ResponseWriter, r *http.Request) { got = tokenFromRequest(r) }, httptest.NewRecorder(), r)
		Expect(got).To(Equal("tok123"))
	})
})

var _ = Describe("parseMediaBrowserAuth", func() {
	authFor := func(header string) mediaBrowserAuth {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("X-Emby-Authorization", header)
		return parseMediaBrowserAuth(r)
	}

	It("reads Finamp's raw (unencoded) field values", func() {
		a := authFor(`MediaBrowser Client="Finamp", Device="Pixel 8 Pro", DeviceId="dev1", Version="1.0", Token="tok"`)
		Expect(a.Client).To(Equal("Finamp"))
		Expect(a.Device).To(Equal("Pixel 8 Pro"))
		Expect(a.DeviceId).To(Equal("dev1"))
	})

	It("percent-decodes Jellify's URL-encoded field values", func() {
		a := authFor(`MediaBrowser Client="Jellify", Device="Pixel%208%20Pro", DeviceId="dev1", Version="1.0", Token="tok"`)
		Expect(a.Client).To(Equal("Jellify"))
		Expect(a.Device).To(Equal("Pixel 8 Pro"))
	})

	It("keeps a literal '%' that isn't valid percent-encoding", func() {
		a := authFor(`MediaBrowser Client="100% Player", Device="d"`)
		Expect(a.Client).To(Equal("100% Player"))
	})

	It("prefers the recommended Authorization header over the deprecated X-Emby-Authorization", func() {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", `MediaBrowser Client="New", DeviceId="dev-new"`)
		r.Header.Set("X-Emby-Authorization", `MediaBrowser Client="Old", DeviceId="dev-old"`)
		a := parseMediaBrowserAuth(r)
		Expect(a.Client).To(Equal("New"))
		Expect(a.DeviceId).To(Equal("dev-new"))
	})

	It("falls back to X-Emby-Authorization when Authorization carries a foreign scheme", func() {
		// A reverse proxy may inject Basic/Digest credentials; the client's MediaBrowser data must
		// still be honored.
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", `Digest username="proxy", realm="site"`)
		r.Header.Set("X-Emby-Authorization", `MediaBrowser Client="Finamp", DeviceId="dev1", Token="tok"`)
		a := parseMediaBrowserAuth(r)
		Expect(a.Client).To(Equal("Finamp"))
		Expect(a.Token).To(Equal("tok"))
	})

	It("rejects a foreign scheme even when its parameters mimic MediaBrowser fields", func() {
		r := httptest.NewRequest("GET", "/", nil)
		r.Header.Set("Authorization", `Custom Token="not-for-us"`)
		Expect(parseMediaBrowserAuth(r).Token).To(BeEmpty())
	})

	It("accepts the legacy Emby scheme spelling, like real Jellyfin", func() {
		a := authFor(`Emby Client="OldClient", DeviceId="dev1", Token="tok"`)
		Expect(a.Client).To(Equal("OldClient"))
		Expect(a.Token).To(Equal("tok"))
	})

	It("matches the scheme case-insensitively (HTTP auth schemes are)", func() {
		a := authFor(`mediabrowser Token="tok"`)
		Expect(a.Token).To(Equal("tok"))
	})
})

var _ = Describe("normalizeQueryKeys", func() {
	// keyFor runs a request through normalizeQueryKeys and reports the value the handler sees for
	// the given (lowercase) key — i.e. what a case-insensitive read would find.
	keyFor := func(rawQuery, key string) string {
		r := httptest.NewRequest("GET", "/Items?"+rawQuery, nil)
		var got string
		invoke(func(_ http.ResponseWriter, r *http.Request) { got = r.URL.Query().Get(key) }, httptest.NewRecorder(), r)
		return got
	}

	It("folds PascalCase (Finamp) and camelCase (Jellify) keys to lowercase", func() {
		Expect(keyFor("ParentId=abc", "parentid")).To(Equal("abc"))
		Expect(keyFor("parentId=abc", "parentid")).To(Equal("abc"))
	})

	It("leaves values untouched", func() {
		Expect(keyFor("IncludeItemTypes=MusicAlbum,Audio", "includeitemtypes")).To(Equal("MusicAlbum,Audio"))
	})

	It("passes already-lowercase keys through unchanged", func() {
		Expect(keyFor("container=mp3", "container")).To(Equal("mp3"))
	})

	It("merges values when two keys fold to the same name instead of dropping one", func() {
		r := httptest.NewRequest("GET", "/Items?Ids=aaa&ids=bbb", nil)
		var got []string
		invoke(func(_ http.ResponseWriter, r *http.Request) { got = r.URL.Query()["ids"] }, httptest.NewRecorder(), r)
		Expect(got).To(ConsistOf("aaa", "bbb"))
	})
})

var _ = Describe("throttleStreams", func() {
	// serve fires n concurrent requests through the middleware and reports the highest number that
	// were ever inside the handler at once.
	serve := func(limit, n int) int32 {
		var inFlight, peak int32
		release := make(chan struct{})
		h := throttleStreams(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cur := atomic.AddInt32(&inFlight, 1)
			for {
				old := atomic.LoadInt32(&peak)
				if cur <= old || atomic.CompareAndSwapInt32(&peak, old, cur) {
					break
				}
			}
			<-release // hold the slot until every request has had a chance to enter
			atomic.AddInt32(&inFlight, -1)
		}))

		var wg sync.WaitGroup
		for range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/Items", nil))
			}()
		}
		// Give the admitted requests time to pile up before letting them finish.
		time.Sleep(100 * time.Millisecond)
		close(release)
		wg.Wait()
		return atomic.LoadInt32(&peak)
	}

	It("admits no more than the limit at once", func() {
		Expect(serve(2, 8)).To(Equal(int32(2)))
	})

	It("queues the excess rather than rejecting it", func() {
		// All 8 still complete — they wait for a slot instead of getting a 429.
		var served int32
		release := make(chan struct{})
		h := throttleStreams(2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-release
			atomic.AddInt32(&served, 1)
		}))
		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/Items", nil))
			}()
		}
		close(release)
		wg.Wait()
		Expect(served).To(Equal(int32(8)))
	})

	// chi's ThrottleBacklog panics on a non-positive limit, so a user disabling the cap must not
	// crash the server at startup.
	It("is disabled, not panicking, when the limit is zero", func() {
		Expect(func() { serve(0, 4) }).ToNot(Panic())
		Expect(serve(0, 4)).To(BeNumerically(">", int32(1)))
	})
})

var _ = Describe("caseInsensitivePaths", func() {
	var handler http.Handler
	var gotID, gotContainer string

	BeforeEach(func() {
		gotID, gotContainer = "", ""
		r := chi.NewRouter()
		// Routes are registered lowercase, mirroring the real router.
		r.Get("/foo/{id}/bar", func(w http.ResponseWriter, req *http.Request) {
			gotID = chi.URLParam(req, "id")
			w.WriteHeader(http.StatusOK)
		})
		r.Get("/audio/{id}/stream.{container}", func(w http.ResponseWriter, req *http.Request) {
			gotContainer = chi.URLParam(req, "container")
			w.WriteHeader(http.StatusOK)
		})
		// A second route reusing the "bar" segment name at a different position.
		r.Get("/bar/{id}", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler = caseInsensitivePaths(r)
	})

	serve := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}

	It("routes a mixed-case request to its lowercase-registered route", func() {
		Expect(serve("/FOO/abc/BAR").Code).To(Equal(http.StatusOK))
	})

	It("routes both routes that share a segment name, regardless of casing", func() {
		Expect(serve("/Foo/abc/Bar").Code).To(Equal(http.StatusOK))
		Expect(serve("/BAR/abc").Code).To(Equal(http.StatusOK))
	})

	It("lowercases the mixed literal.extension segment so the route and container match", func() {
		w := serve("/Audio/abc/STREAM.MP3")
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(gotContainer).To(Equal("mp3"))
	})

	It("lowercases id/param segments (safe: Jellyfin ids are lowercase hex)", func() {
		serve("/foo/DEADBEEF/bar")
		Expect(gotID).To(Equal("deadbeef"))
	})

	It("normalizes the RoutePath branch when mounted under a parent", func() {
		parent := chi.NewRouter()
		parent.Mount("/jellyfin", handler)
		w := httptest.NewRecorder()
		parent.ServeHTTP(w, httptest.NewRequest("GET", "/jellyfin/FOO/abc/BAR", nil))
		Expect(w.Code).To(Equal(http.StatusOK))
	})
})
