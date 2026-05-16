package subsonic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaAnnotationController", func() {
	var router *Router
	var ds model.DataStore
	var playTracker *fakePlayTracker
	var eventBroker *fakeEventBroker
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		ds = &tests.MockDataStore{}
		playTracker = &fakePlayTracker{}
		eventBroker = &fakeEventBroker{}
		router = New(ds, nil, nil, nil, nil, nil, nil, eventBroker, nil, playTracker, nil, nil, nil, nil, nil, nil)
	})

	Describe("Scrobble", func() {
		It("submit all scrobbles with only the id", func() {
			submissionTime := time.Now().Add(-time.Second)
			r := newGetRequest("id=12", "id=34")

			_, err := router.Scrobble(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(playTracker.Submissions).To(HaveLen(2))
			Expect(playTracker.Submissions[0].Timestamp).To(BeTemporally(">", submissionTime))
			Expect(playTracker.Submissions[0].TrackID).To(Equal("12"))
			Expect(playTracker.Submissions[1].Timestamp).To(BeTemporally(">", submissionTime))
			Expect(playTracker.Submissions[1].TrackID).To(Equal("34"))
		})

		It("submit all scrobbles with respective times", func() {
			time1 := time.Now().Add(-20 * time.Minute)
			t1 := time1.UnixMilli()
			time2 := time.Now().Add(-10 * time.Minute)
			t2 := time2.UnixMilli()
			r := newGetRequest("id=12", "id=34", fmt.Sprintf("time=%d", t1), fmt.Sprintf("time=%d", t2))

			_, err := router.Scrobble(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(playTracker.Submissions).To(HaveLen(2))
			Expect(playTracker.Submissions[0].Timestamp).To(BeTemporally("~", time1))
			Expect(playTracker.Submissions[0].TrackID).To(Equal("12"))
			Expect(playTracker.Submissions[1].Timestamp).To(BeTemporally("~", time2))
			Expect(playTracker.Submissions[1].TrackID).To(Equal("34"))
		})

		It("checks if number of ids match number of times", func() {
			r := newGetRequest("id=12", "id=34", "time=1111")

			_, err := router.Scrobble(r)

			Expect(err).To(HaveOccurred())
			Expect(playTracker.Submissions).To(BeEmpty())
		})

		Context("submission=false", func() {
			var req *http.Request
			BeforeEach(func() {
				_ = ds.MediaFile(ctx).Put(&model.MediaFile{ID: "12"})
				ctx = request.WithPlayer(ctx, model.Player{ID: "player-1"})
				req = newGetRequest("id=12", "submission=false")
				req = req.WithContext(ctx)
			})

			It("does not scrobble", func() {
				_, err := router.Scrobble(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.Submissions).To(BeEmpty())
			})

			It("registers a NowPlaying via ReportPlayback", func() {
				_, err := router.Scrobble(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.ReportedPlayback).To(HaveLen(1))
				Expect(playTracker.ReportedPlayback[0].MediaId).To(Equal("12"))
				Expect(playTracker.ReportedPlayback[0].State).To(Equal(scrobbler.StatePlaying))
				Expect(playTracker.ReportedPlayback[0].ClientId).To(Equal("player-1"))
			})
		})
	})

	Describe("Star/Unstar songs", func() {
		var mediaRepo *recordingMediaFileRepo

		BeforeEach(func() {
			mediaRepo = &recordingMediaFileRepo{}
			ds.(*tests.MockDataStore).MockedAlbum = &alwaysMissingAlbumRepo{}
			ds.(*tests.MockDataStore).MockedArtist = &alwaysMissingArtistRepo{}
			ds.(*tests.MockDataStore).MockedMediaFile = mediaRepo
		})

		It("stars a song by id", func() {
			resp, err := router.Star(newGetRequest("id=song-1"))

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(mediaRepo.SetStarCalls).To(Equal([]setStarCall{{Starred: true, ItemIDs: []string{"song-1"}}}))
			Expect(eventBroker.Events).To(HaveLen(1))
			Expect(eventBroker.Events[0].Data(eventBroker.Events[0])).To(Equal(`{"song":["song-1"]}`))
		})

		It("unstars a song by id", func() {
			resp, err := router.Unstar(newGetRequest("id=song-1"))

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
			Expect(mediaRepo.SetStarCalls).To(Equal([]setStarCall{{Starred: false, ItemIDs: []string{"song-1"}}}))
			Expect(eventBroker.Events).To(HaveLen(1))
			Expect(eventBroker.Events[0].Data(eventBroker.Events[0])).To(Equal(`{"song":["song-1"]}`))
		})

		It("stars returns error when no id parameter is provided", func() {
			_, err := router.Star(newGetRequest())

			Expect(err).To(HaveOccurred())
			Expect(mediaRepo.SetStarCalls).To(BeEmpty())
			Expect(eventBroker.Events).To(BeEmpty())
		})

		It("unstars returns error when no id parameter is provided", func() {
			_, err := router.Unstar(newGetRequest())

			Expect(err).To(HaveOccurred())
			Expect(mediaRepo.SetStarCalls).To(BeEmpty())
			Expect(eventBroker.Events).To(BeEmpty())
		})

		It("returns error and calls repository when star persistence fails", func() {
			mediaRepo.Err = errors.New("db failure")
			_, err := router.Star(newGetRequest("id=song-1"))

			Expect(err).To(HaveOccurred())
			Expect(mediaRepo.SetStarCalls).To(HaveLen(1))
			Expect(eventBroker.Events).To(BeEmpty())
		})

		It("returns error and calls repository when unstar persistence fails", func() {
			mediaRepo.Err = errors.New("db failure")
			_, err := router.Unstar(newGetRequest("id=song-1"))

			Expect(err).To(HaveOccurred())
			Expect(mediaRepo.SetStarCalls).To(HaveLen(1))
			Expect(eventBroker.Events).To(BeEmpty())
		})

		It("rejects unauthenticated star requests before favoriting the song", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/star?u=missing&v=1.16.1&c=test&id=song-1", nil)

			router.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
			Expect(mediaRepo.SetStarCalls).To(BeEmpty())
			Expect(eventBroker.Events).To(BeEmpty())
		})

		It("rejects unauthenticated unstar requests before unfavoriting the song", func() {
			w := httptest.NewRecorder()
			r := httptest.NewRequest("GET", "/unstar?u=missing&v=1.16.1&c=test&id=song-1", nil)

			router.ServeHTTP(w, r)

			Expect(w.Body.String()).To(ContainSubstring(`code="40"`))
			Expect(mediaRepo.SetStarCalls).To(BeEmpty())
			Expect(eventBroker.Events).To(BeEmpty())
		})
	})

	Describe("ReportPlayback", func() {
		It("returns error when mediaId is missing", func() {
			r := newGetRequest("mediaType=song", "positionMs=0", "state=playing")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when mediaType is missing", func() {
			r := newGetRequest("mediaId=123", "positionMs=0", "state=playing")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when positionMs is missing", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "state=playing")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error when state is missing", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid state value", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=invalid")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for negative positionMs", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=-1", "state=playing")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for NaN playbackRate", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=playing", "playbackRate=NaN")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for Inf playbackRate", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=playing", "playbackRate=Inf")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for negative playbackRate", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=playing", "playbackRate=-1.0")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("returns error for zero playbackRate", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=0", "state=playing", "playbackRate=0")
			_, err := router.ReportPlayback(r)
			Expect(err).To(HaveOccurred())
		})

		It("accepts mediaType=podcast without error", func() {
			r := newGetRequest("mediaId=123", "mediaType=podcast", "positionMs=0", "state=playing")
			ctx := request.WithPlayer(r.Context(), model.Player{ID: "p1"})
			r = r.WithContext(ctx)
			resp, err := router.ReportPlayback(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.Status).To(Equal(responses.StatusOK))
		})

		It("defaults playbackRate to 1.0 and ignoreScrobble to false", func() {
			r := newGetRequest("mediaId=123", "mediaType=song", "positionMs=5000", "state=playing")
			ctx := request.WithPlayer(r.Context(), model.Player{ID: "p1"})
			r = r.WithContext(ctx)
			_, err := router.ReportPlayback(r)
			Expect(err).ToNot(HaveOccurred())
			Expect(playTracker.ReportedPlayback).To(HaveLen(1))
			Expect(playTracker.ReportedPlayback[0].PlaybackRate).To(Equal(1.0))
			Expect(playTracker.ReportedPlayback[0].IgnoreScrobble).To(BeFalse())
			Expect(playTracker.ReportedPlayback[0].ClientId).To(Equal("p1"))
			Expect(playTracker.ReportedPlayback[0].ClientName).To(BeEmpty())
		})
	})
})

type fakePlayTracker struct {
	Submissions      []scrobbler.Submission
	ReportedPlayback []scrobbler.ReportPlaybackParams
	Error            error
}

func (f *fakePlayTracker) GetNowPlaying(_ context.Context) ([]scrobbler.PlaybackSession, error) {
	return nil, f.Error
}

func (f *fakePlayTracker) Submit(_ context.Context, submissions []scrobbler.Submission) error {
	if f.Error != nil {
		return f.Error
	}
	f.Submissions = append(f.Submissions, submissions...)
	return nil
}

func (f *fakePlayTracker) ReportPlayback(_ context.Context, params scrobbler.ReportPlaybackParams) error {
	if f.Error != nil {
		return f.Error
	}
	f.ReportedPlayback = append(f.ReportedPlayback, params)
	return nil
}

var _ scrobbler.PlayTracker = (*fakePlayTracker)(nil)

type setStarCall struct {
	Starred bool
	ItemIDs []string
}

type recordingMediaFileRepo struct {
	model.MediaFileRepository
	SetStarCalls []setStarCall
	Err          error
}

func (r *recordingMediaFileRepo) SetStar(starred bool, itemIDs ...string) error {
	r.SetStarCalls = append(r.SetStarCalls, setStarCall{
		Starred: starred,
		ItemIDs: append([]string(nil), itemIDs...),
	})
	return r.Err
}

type alwaysMissingAlbumRepo struct {
	model.AlbumRepository
}

func (r *alwaysMissingAlbumRepo) Exists(string) (bool, error) {
	return false, nil
}

type alwaysMissingArtistRepo struct {
	model.ArtistRepository
}

func (r *alwaysMissingArtistRepo) Exists(string) (bool, error) {
	return false, nil
}

type fakeEventBroker struct {
	http.Handler
	Events []events.Event
}

func (f *fakeEventBroker) SendMessage(_ context.Context, event events.Event) {
	f.Events = append(f.Events, event)
}

func (f *fakeEventBroker) SendBroadcastMessage(_ context.Context, event events.Event) {
	f.Events = append(f.Events, event)
}

var _ events.Broker = (*fakeEventBroker)(nil)
