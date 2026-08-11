package public

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("handlePlayback", func() {
	var (
		ds         *tests.MockDataStore
		streamer   stream.MediaStreamer
		router     http.Handler
		rawBytes   []byte
		otherBytes []byte
		fixture    = "tests/fixtures/test.mp3"
		missing    string
	)

	mutateTokenMediaID := func(token, mediaID string) string {
		parts := strings.Split(token, ".")
		Expect(parts).To(HaveLen(3))
		payload, err := base64.RawURLEncoding.DecodeString(parts[1])
		Expect(err).NotTo(HaveOccurred())
		claims := map[string]any{}
		Expect(json.Unmarshal(payload, &claims)).To(Succeed())
		claims["id"] = mediaID
		payload, err = json.Marshal(claims)
		Expect(err).NotTo(HaveOccurred())
		parts[1] = base64.RawURLEncoding.EncodeToString(payload)
		return strings.Join(parts, ".")
	}

	makeRequest := func(method, target string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		return w
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		auth.PublicTokenAuth = jwtauth.New("HS256", []byte("test-secret"), nil)
		cacheDir, err := os.MkdirTemp("", "playback-capability-cache")
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() { _ = os.RemoveAll(cacheDir) })
		conf.Server.CacheFolder = conf.NewDir(cacheDir)
		conf.Server.TranscodingCacheSize = "100MB"
		conf.Server.EnableSharing = false
		conf.Server.BaseHost = "lan.example"
		conf.Server.BaseScheme = "https"
		conf.Server.BasePath = ""

		rawBytes, err = os.ReadFile(fixture)
		Expect(err).NotTo(HaveOccurred())
		otherBytes = []byte("DIFFERENT")
		otherPath := path.Join(cacheDir, "other.mp3")
		Expect(os.WriteFile(otherPath, otherBytes, 0o644)).To(Succeed())
		missing = path.Join(cacheDir, "missing.mp3")

		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		ds.MediaFile(context.TODO()).(*tests.MockMediaFileRepo).SetData(model.MediaFiles{
			{ID: "mf-123", Title: "Test Song", Path: fixture, Suffix: "mp3", Duration: 257, Size: int64(len(rawBytes))},
			{ID: "mf-456", Title: "Other Song", Path: otherPath, Suffix: "mp3", Duration: 1, Size: int64(len(otherBytes))},
			{ID: "mf-unreadable", Title: "Missing File", Path: missing, Suffix: "mp3", Duration: 1},
		})
		cache := stream.NewTranscodingCache()
		Eventually(func() bool { return cache.Available(context.TODO()) }).Should(BeTrue())
		streamer = stream.NewMediaStreamer(ds, tests.NewMockFFmpeg("unused"), cache)

		pub := &Router{ds: ds, streamer: streamer}
		mount := chi.NewRouter()
		mount.Mount(consts.URLPathPublic, pub.routes())
		router = mount
	})

	It("streams the raw media bytes without normal authentication", func() {
		token, err := auth.CreatePlaybackToken("mf-123")
		Expect(err).NotTo(HaveOccurred())

		w := makeRequest(http.MethodGet, consts.URLPathPublic+"/playback/"+token)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Bytes()).To(Equal(rawBytes))
		Expect(w.Header().Get("Content-Type")).To(Equal("audio/mpeg"))
	})

	It("supports HEAD requests", func() {
		token, err := auth.CreatePlaybackToken("mf-123")
		Expect(err).NotTo(HaveOccurred())

		w := makeRequest(http.MethodHead, consts.URLPathPublic+"/playback/"+token)
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Len()).To(Equal(0))
		Expect(w.Header().Get("Content-Type")).To(Equal("audio/mpeg"))
	})

	It("supports range requests", func() {
		token, err := auth.CreatePlaybackToken("mf-123")
		Expect(err).NotTo(HaveOccurred())

		req := httptest.NewRequest(http.MethodGet, consts.URLPathPublic+"/playback/"+token, nil)
		req.Header.Set("Range", "bytes=0-3")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		Expect(w.Code).To(Equal(http.StatusPartialContent))
		Expect(w.Body.Bytes()).To(Equal(rawBytes[:4]))
	})

	It("rejects an invalid token", func() {
		w := makeRequest(http.MethodGet, consts.URLPathPublic+"/playback/not-a-token")
		Expect(w.Code).To(Equal(http.StatusBadRequest))
	})

	It("returns 404 when the media file does not exist", func() {
		token, err := auth.CreatePlaybackToken("missing")
		Expect(err).NotTo(HaveOccurred())
		w := makeRequest(http.MethodGet, consts.URLPathPublic+"/playback/"+token)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("rejects a token whose media id was mutated without resigning", func() {
		token, err := auth.CreatePlaybackToken("mf-123")
		Expect(err).NotTo(HaveOccurred())
		mutated := mutateTokenMediaID(token, "mf-456")

		w := makeRequest(http.MethodGet, consts.URLPathPublic+"/playback/"+mutated)
		Expect(w.Code).To(Equal(http.StatusBadRequest))
		Expect(w.Body.Bytes()).NotTo(Equal(otherBytes))
	})

	It("returns 500 when the backing file cannot be opened", func() {
		token, err := auth.CreatePlaybackToken("mf-unreadable")
		Expect(err).NotTo(HaveOccurred())
		w := makeRequest(http.MethodGet, consts.URLPathPublic+"/playback/"+token)
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
	})

	It("serves a generated playback URL through the mounted public router", func() {
		playbackURL, err := publicurl.PlaybackURL("mf-123")
		Expect(err).NotTo(HaveOccurred())
		u, err := url.Parse(playbackURL)
		Expect(err).NotTo(HaveOccurred())

		w := makeRequest(http.MethodGet, u.RequestURI())
		Expect(w.Code).To(Equal(http.StatusOK))
		Expect(w.Body.Bytes()).To(Equal(rawBytes))

		rangeReq := httptest.NewRequest(http.MethodGet, u.RequestURI(), nil)
		rangeReq.Header.Set("Range", "bytes=1-4")
		rangeW := httptest.NewRecorder()
		router.ServeHTTP(rangeW, rangeReq)
		Expect(rangeW.Code).To(Equal(http.StatusPartialContent))
		Expect(rangeW.Body.Bytes()).To(Equal(rawBytes[1:5]))
	})
})
