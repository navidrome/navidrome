package jellyfin

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakePlayTracker is a local double for scrobbler.PlayTracker, mirroring
// server/subsonic's fakePlayTracker.
type fakePlayTracker struct {
	scrobbler.PlayTracker
	reported  []scrobbler.ReportPlaybackParams
	submitted []scrobbler.Submission
}

func (f *fakePlayTracker) ReportPlayback(_ context.Context, p scrobbler.ReportPlaybackParams) error {
	f.reported = append(f.reported, p)
	return nil
}

func (f *fakePlayTracker) Submit(_ context.Context, s []scrobbler.Submission) error {
	f.submitted = append(f.submitted, s...)
	return nil
}

// fakePlayers is a local double for core.Players, used to exercise withPlayer.
type fakePlayers struct {
	core.Players
	err error
}

func (f *fakePlayers) Register(_ context.Context, id, client, _, _ string) (*model.Player, *model.Transcoding, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	return &model.Player{ID: id, Client: client}, nil, nil
}

var _ = Describe("Sessions", func() {
	var api *Router
	var pt *fakePlayTracker

	authed := func(r *http.Request) *http.Request {
		ctx := request.WithUser(context.Background(), model.User{ID: "u1", UserName: "alice"})
		ctx = request.WithPlayer(ctx, model.Player{ID: "p1", Client: "Finamp"})
		return r.WithContext(ctx)
	}

	BeforeEach(func() {
		pt = &fakePlayTracker{}
		api = &Router{ds: &tests.MockDataStore{}, scrobbler: pt}
	})

	Describe("reportPlaybackStart", func() {
		It("reports playback start with the item id and position", func() {
			w := httptest.NewRecorder()
			r := authed(httptest.NewRequest("POST", "/Sessions/Playing", strings.NewReader(`{"ItemId":"s1","PositionTicks":10000000}`)))

			api.reportPlaybackStart(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(pt.reported).To(HaveLen(1))
			Expect(pt.reported[0].MediaId).To(Equal("s1"))
			Expect(pt.reported[0].PositionMs).To(Equal(int64(1000)))
			Expect(pt.reported[0].State).To(Equal(scrobbler.StatePlaying))
			Expect(pt.reported[0].ClientId).To(Equal("p1"))
			Expect(pt.reported[0].ClientName).To(Equal("Finamp"))
		})

		It("falls back to the ItemId query param when the body has none", func() {
			w := httptest.NewRecorder()
			r := authed(httptest.NewRequest("POST", "/Sessions/Playing?ItemId=s2", nil))

			api.reportPlaybackStart(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(pt.reported).To(HaveLen(1))
			Expect(pt.reported[0].MediaId).To(Equal("s2"))
		})
	})

	Describe("reportPlaybackProgress", func() {
		It("reports the playing state when not paused", func() {
			w := httptest.NewRecorder()
			r := authed(httptest.NewRequest("POST", "/Sessions/Playing/Progress", strings.NewReader(`{"ItemId":"s1","PositionTicks":20000000,"IsPaused":false}`)))

			api.reportPlaybackProgress(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(pt.reported).To(HaveLen(1))
			Expect(pt.reported[0].State).To(Equal(scrobbler.StatePlaying))
			Expect(pt.reported[0].PositionMs).To(Equal(int64(2000)))
		})

		It("reports the paused state when IsPaused is true", func() {
			w := httptest.NewRecorder()
			r := authed(httptest.NewRequest("POST", "/Sessions/Playing/Progress", strings.NewReader(`{"ItemId":"s1","PositionTicks":20000000,"IsPaused":true}`)))

			api.reportPlaybackProgress(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(pt.reported).To(HaveLen(1))
			Expect(pt.reported[0].State).To(Equal(scrobbler.StatePaused))
		})
	})

	Describe("reportPlaybackStopped", func() {
		It("reports the stopped state and submits a scrobble", func() {
			before := time.Now().Add(-time.Second)
			w := httptest.NewRecorder()
			r := authed(httptest.NewRequest("POST", "/Sessions/Playing/Stopped", strings.NewReader(`{"ItemId":"s1","PositionTicks":600000000}`)))

			api.reportPlaybackStopped(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))

			Expect(pt.reported).To(HaveLen(1))
			Expect(pt.reported[0].MediaId).To(Equal("s1"))
			Expect(pt.reported[0].State).To(Equal(scrobbler.StateStopped))
			Expect(pt.reported[0].PositionMs).To(Equal(int64(60000)))
			// IgnoreScrobble must be set: Submit below is the sole play-count/scrobble trigger,
			// otherwise ReportPlayback's own threshold logic would double-count the play.
			Expect(pt.reported[0].IgnoreScrobble).To(BeTrue())

			Expect(pt.submitted).To(HaveLen(1))
			Expect(pt.submitted[0].TrackID).To(Equal("s1"))
			Expect(pt.submitted[0].Timestamp).To(BeTemporally(">", before))
		})
	})

	Describe("postCapabilities", func() {
		It("returns 204 No Content and does not touch the scrobbler", func() {
			w := httptest.NewRecorder()
			r := authed(httptest.NewRequest("POST", "/Sessions/Capabilities", strings.NewReader(`{"SupportsMediaControl":true}`)))

			api.postCapabilities(w, r)

			Expect(w.Code).To(Equal(http.StatusNoContent))
			Expect(pt.reported).To(BeEmpty())
			Expect(pt.submitted).To(BeEmpty())
		})
	})
})

var _ = Describe("withPlayer middleware", func() {
	var api *Router
	var fp *fakePlayers

	BeforeEach(func() {
		fp = &fakePlayers{}
		api = &Router{ds: &tests.MockDataStore{}, players: fp}
	})

	It("registers a player from the Emby device info and injects it into the context", func() {
		var gotPlayer model.Player
		var gotOk bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPlayer, gotOk = request.PlayerFrom(r.Context())
			w.WriteHeader(http.StatusNoContent)
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Sessions/Playing", nil)
		r.Header.Set("X-Emby-Authorization", `MediaBrowser Client="Finamp", Device="Pixel", DeviceId="dev1", Version="1.0"`)

		api.withPlayer(next).ServeHTTP(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(gotOk).To(BeTrue())
		Expect(gotPlayer.ID).To(Equal("dev1"))
		Expect(gotPlayer.Client).To(Equal("Finamp"))
	})

	It("fails open (no player in context) when registration errors", func() {
		fp.err = errors.New("boom")
		var gotOk bool
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, gotOk = request.PlayerFrom(r.Context())
			w.WriteHeader(http.StatusNoContent)
		})

		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/Sessions/Playing", nil)

		api.withPlayer(next).ServeHTTP(w, r)

		Expect(w.Code).To(Equal(http.StatusNoContent))
		Expect(gotOk).To(BeFalse())
	})
})
