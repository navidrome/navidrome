package public

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("handleCastStream", func() {
	var (
		ctx      context.Context
		ds       *tests.MockDataStore
		streamer *mockStreamer
		pub      *Router
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableExternalServices = true
		ctx = GinkgoT().Context()
		auth.PublicTokenAuth = jwtauth.New("HS256", []byte("test-cast-secret"), nil)
		auth.TokenAuth = jwtauth.New("HS256", []byte("test-session-secret"), nil)
		ds = &tests.MockDataStore{MockedTranscoding: &tests.MockTranscodingRepo{}}
		streamer = &mockStreamer{}
		pub = &Router{ds: ds, streamer: streamer, decider: stream.NewTranscodeDecider(ds, tests.NewMockFFmpeg(""))}
	})

	makeRequest := func(token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/public/cast/token?%3Aid="+token, nil)
		w := httptest.NewRecorder()
		pub.handleCastStream(w, r)
		return w
	}

	seedUserAndTrack := func(owner model.User, mf model.MediaFile) {
		userRepo := tests.CreateMockUserRepo()
		Expect(userRepo.Put(ctx, &owner)).To(Succeed())
		ds.MockedUser = userRepo
		mfRepo := tests.CreateMockMediaFileRepo()
		mfRepo.SetData(model.MediaFiles{mf})
		ds.MockedMediaFile = mfRepo
	}

	It("streams an authorized track for a valid short-lived cast token", func() {
		seedUserAndTrack(
			model.User{ID: "u1", UserName: "alice", TokenEpoch: 2, Libraries: model.Libraries{{ID: 1}}},
			model.MediaFile{ID: "song-1", LibraryID: 1, Suffix: "flac", SampleRate: 44100, Channels: 2},
		)

		token, err := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour), auth.Claims{
			Audience: []string{auth.AudienceCast},
			UserID:   "u1",
			Epoch:    2,
			ID:       "song-1",
			Format:   "mp3",
			BitRate:  320,
		})
		Expect(err).NotTo(HaveOccurred())

		makeRequest(token)
		Expect(streamer.called).To(BeTrue())
		Expect(streamer.req.Format).To(Equal("mp3"))
		Expect(streamer.req.BitRate).To(Equal(320))
	})

	It("rejects expired, non-expiring, wrong-audience, and revoked-epoch tokens", func() {
		seedUserAndTrack(
			model.User{ID: "u1", UserName: "alice", TokenEpoch: 3, Libraries: model.Libraries{{ID: 1}}},
			model.MediaFile{ID: "song-1", LibraryID: 1, Suffix: "mp3"},
		)

		expired, _ := auth.CreateExpiringPublicToken(time.Now().Add(-time.Minute), auth.Claims{
			Audience: []string{auth.AudienceCast}, UserID: "u1", Epoch: 3, ID: "song-1", Format: "raw",
		})
		Expect(makeRequest(expired).Code).To(Equal(http.StatusBadRequest))

		noExp, _ := auth.CreatePublicToken(auth.Claims{
			Audience: []string{auth.AudienceCast}, UserID: "u1", Epoch: 3, ID: "song-1", Format: "raw",
		})
		Expect(makeRequest(noExp).Code).To(Equal(http.StatusBadRequest))

		shareToken, _ := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour), auth.Claims{
			ShareID: "sh1", UserID: "u1", Epoch: 3, ID: "song-1", Format: "raw",
		})
		Expect(makeRequest(shareToken).Code).To(Equal(http.StatusBadRequest))

		staleEpoch, _ := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour), auth.Claims{
			Audience: []string{auth.AudienceCast}, UserID: "u1", Epoch: 2, ID: "song-1", Format: "raw",
		})
		Expect(makeRequest(staleEpoch).Code).To(Equal(http.StatusUnauthorized))
		Expect(streamer.called).To(BeFalse())
	})

	It("returns 404 when the track is outside the user's accessible libraries or external services are disabled", func() {
		seedUserAndTrack(
			model.User{ID: "u1", UserName: "alice", TokenEpoch: 1, Libraries: model.Libraries{{ID: 1}}},
			model.MediaFile{ID: "song-2", LibraryID: 2, Suffix: "mp3"},
		)

		token, _ := auth.CreateExpiringPublicToken(time.Now().Add(time.Hour), auth.Claims{
			Audience: []string{auth.AudienceCast}, UserID: "u1", Epoch: 1, ID: "song-2", Format: "raw",
		})
		Expect(makeRequest(token).Code).To(Equal(http.StatusNotFound))

		conf.Server.EnableExternalServices = false
		Expect(makeRequest(token).Code).To(Equal(http.StatusNotFound))
		Expect(streamer.called).To(BeFalse())
	})
})
