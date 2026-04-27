package public

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type mockStreamer struct {
	req    stream.Request
	called bool
}

func (m *mockStreamer) NewStream(_ context.Context, _ *model.MediaFile, r stream.Request) (*stream.Stream, error) {
	m.called = true
	m.req = r
	return nil, errors.New("mock: not implemented")
}

var _ = Describe("decodeStreamInfo", func() {
	BeforeEach(func() {
		auth.TokenAuth = jwtauth.New("HS256", []byte("test-secret"), nil)
	})

	It("decodes a valid token with all fields", func() {
		claims := auth.Claims{ID: "mf-123", Format: "mp3", BitRate: 192, ShareID: "share123"}
		token, _ := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour), claims)
		info, err := decodeStreamInfo(token)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.id).To(Equal("mf-123"))
		Expect(info.format).To(Equal("mp3"))
		Expect(info.bitrate).To(Equal(192))
		Expect(info.shareID).To(Equal("share123"))
	})

	It("rejects an expired token", func() {
		claims := auth.Claims{ID: "mf-123", ShareID: "share123"}
		token, _ := auth.CreateExpiringPublicToken(time.Now().Add(-time.Hour), claims)
		_, err := decodeStreamInfo(token)
		Expect(err).To(HaveOccurred())
	})

	It("accepts a token without exp (non-expiring share)", func() {
		claims := auth.Claims{ID: "mf-123", ShareID: "share123"}
		token, _ := auth.CreatePublicToken(claims)
		info, err := decodeStreamInfo(token)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.id).To(Equal("mf-123"))
		Expect(info.shareID).To(Equal("share123"))
	})

	It("rejects a token without an id claim", func() {
		claims := auth.Claims{ShareID: "share123"}
		token, _ := auth.CreatePublicToken(claims)
		_, err := decodeStreamInfo(token)
		Expect(err).To(HaveOccurred())
	})

	It("rejects an invalid token string", func() {
		_, err := decodeStreamInfo("not-a-valid-token")
		Expect(err).To(HaveOccurred())
	})

	It("handles tokens without shareID (backward compat)", func() {
		claims := auth.Claims{ID: "mf-123", Format: "opus"}
		token, _ := auth.CreatePublicToken(claims)
		info, err := decodeStreamInfo(token)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.id).To(Equal("mf-123"))
		Expect(info.format).To(Equal("opus"))
		Expect(info.shareID).To(BeEmpty())
	})
})

var _ = Describe("encodeMediafileShare", func() {
	BeforeEach(func() {
		auth.TokenAuth = jwtauth.New("HS256", []byte("test-secret"), nil)
	})

	It("includes the share ID in the token", func() {
		exp := P(time.Now().Add(time.Hour))
		s := model.Share{ID: "shareABC", Format: "mp3", MaxBitRate: 320, ExpiresAt: exp}
		token := encodeMediafileShare(s, "mf-999")
		info, err := decodeStreamInfo(token)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.shareID).To(Equal("shareABC"))
		Expect(info.id).To(Equal("mf-999"))
		Expect(info.format).To(Equal("mp3"))
		Expect(info.bitrate).To(Equal(320))
	})

	It("creates a non-expiring token when share has no expiry", func() {
		s := model.Share{ID: "shareXYZ", ExpiresAt: nil}
		token := encodeMediafileShare(s, "mf-111")
		info, err := decodeStreamInfo(token)
		Expect(err).NotTo(HaveOccurred())
		Expect(info.shareID).To(Equal("shareXYZ"))
		Expect(info.id).To(Equal("mf-111"))
	})
})

var _ = Describe("handleStream", func() {
	var ds *tests.MockDataStore
	var shareRepo *tests.MockShareRepo
	var streamer *mockStreamer
	var pub *Router

	BeforeEach(func() {
		auth.TokenAuth = jwtauth.New("HS256", []byte("test-secret"), nil)
		ds = &tests.MockDataStore{}
		shareRepo = &tests.MockShareRepo{}
		ds.MockedShare = shareRepo
		streamer = &mockStreamer{}
		pub = &Router{ds: ds, streamer: streamer}
	})

	makeRequest := func(token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/public/s/token?%3Aid="+token, nil)
		w := httptest.NewRecorder()
		pub.handleStream(w, r)
		return w
	}

	It("passes all validation and reaches the streamer for a valid token", func() {
		shareRepo.ID = "share123"
		mfRepo := tests.CreateMockMediaFileRepo()
		mfRepo.SetData(model.MediaFiles{{ID: "mf-123", Title: "Test Song"}})
		ds.MockedMediaFile = mfRepo

		claims := auth.Claims{ID: "mf-123", Format: "mp3", BitRate: 192, ShareID: "share123"}
		token, _ := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour), claims)
		makeRequest(token)

		Expect(streamer.called).To(BeTrue())
		Expect(streamer.req.Format).To(Equal("mp3"))
		Expect(streamer.req.BitRate).To(Equal(192))
	})

	It("returns 400 for an expired token", func() {
		claims := auth.Claims{ID: "mf-123", ShareID: "share123"}
		token, _ := auth.CreateExpiringPublicToken(time.Now().Add(-time.Hour), claims)
		w := makeRequest(token)
		Expect(w.Code).To(Equal(http.StatusBadRequest))
	})

	It("returns 404 when share has been deleted", func() {
		shareRepo.ID = "other-share"
		claims := auth.Claims{ID: "mf-123", ShareID: "deleted-share"}
		token, _ := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour), claims)
		w := makeRequest(token)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 410 when share has been set to expired", func() {
		shareRepo.ID = "share123"
		expired := time.Now().Add(-time.Hour)
		shareRepo.Entity = &model.Share{ID: "share123", ExpiresAt: &expired}

		claims := auth.Claims{ID: "mf-123", ShareID: "share123"}
		token, _ := auth.CreatePublicToken(claims)
		w := makeRequest(token)
		Expect(w.Code).To(Equal(http.StatusGone))
	})

	It("returns 500 when share lookup fails", func() {
		shareRepo.Error = errors.New("db error")
		claims := auth.Claims{ID: "mf-123", ShareID: "share123"}
		token, _ := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour), claims)
		w := makeRequest(token)
		Expect(w.Code).To(Equal(http.StatusInternalServerError))
	})

	It("skips share check for tokens without shareID (backward compat)", func() {
		claims := auth.Claims{ID: "mf-123"}
		token, _ := auth.CreatePublicToken(claims)
		w := makeRequest(token)
		// Should get past share check, then fail on media file lookup (no mock data)
		Expect(w.Code).To(Equal(http.StatusNotFound))
	})

	It("returns 400 for an invalid token", func() {
		w := makeRequest("not-a-valid-token")
		Expect(w.Code).To(Equal(http.StatusBadRequest))
	})
})
