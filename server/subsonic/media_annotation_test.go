package subsonic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaAnnotationController", func() {
	var router *Router
	var ds model.DataStore
	var playTracker *fakePlayTracker
	var eventBroker *fakeEventBroker
	var ctx context.Context

	var userCtx context.Context

	// putOwnedPlaylist stores u1's playlist pl-1, which the specs annotate as u1.
	putOwnedPlaylist := func() {
		GinkgoHelper()
		owner := model.User{ID: "u1", UserName: "u1"}
		putUsers(ctx, ds, owner)
		userCtx = request.WithUser(ctx, owner)
		Expect(ds.Playlist().Put(userCtx, &model.Playlist{ID: "pl-1", Name: "My Playlist", OwnerID: "u1"})).To(Succeed())
	}
	storedPlaylist := func() *model.Playlist {
		GinkgoHelper()
		pls, err := ds.Playlist().Get(userCtx, "pl-1")
		Expect(err).ToNot(HaveOccurred())
		return pls
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		playTracker = &fakePlayTracker{}
		eventBroker = &fakeEventBroker{}
		router = New(ds, nil, nil, nil, nil, nil, nil, eventBroker, nil, playTracker, nil, nil, nil, nil, nil, nil)
	})

	Describe("Scrobble", func() {
		It("submit all scrobbles with only the id", func() {
			// Back-date the baseline so the assertion still passes on platforms
			// with millisecond clock resolution (e.g. Windows).
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
				persistencetest.PutMediaFiles(ctx, ds, model.MediaFile{ID: "12"})
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

	Describe("Star/Unstar playlists", func() {
		BeforeEach(func() {
			putOwnedPlaylist()
		})

		It("stars a playlist by dispatching to the Playlist repo", func() {
			r := newGetRequest("id=pl-1").WithContext(userCtx)

			_, err := router.Star(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist().Starred).To(BeTrue())
		})

		It("unstars a playlist by dispatching to the Playlist repo", func() {
			Expect(ds.Playlist().SetStar(userCtx, true, "pl-1")).To(Succeed())
			r := newGetRequest("id=pl-1").WithContext(userCtx)

			_, err := router.Unstar(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist().Starred).To(BeFalse())
		})
	})

	Describe("SetRating playlists", func() {
		BeforeEach(func() {
			putOwnedPlaylist()
		})

		It("rates a playlist by dispatching to the Playlist repo", func() {
			r := newGetRequest("id=pl-1", "rating=4").WithContext(userCtx)

			_, err := router.SetRating(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(storedPlaylist().Rating).To(Equal(4))
		})
	})

	Describe("SetRating range validation", func() {
		BeforeEach(func() {
			putOwnedPlaylist()
		})

		DescribeTable("rejects ratings outside the 0-5 range allowed by the Subsonic spec",
			func(rating string) {
				_, err := router.SetRating(newGetRequest("id=pl-1", "rating=4").WithContext(userCtx))
				Expect(err).ToNot(HaveOccurred())

				r := newGetRequest("id=pl-1", "rating="+rating).WithContext(userCtx)

				_, err = router.SetRating(r)

				Expect(err).To(MatchError(errSubsonic))
				var subErr subError
				Expect(errors.As(err, &subErr)).To(BeTrue())
				Expect(subErr.code).To(Equal(responses.ErrorGeneric))
				Expect(storedPlaylist().Rating).To(Equal(4))
			},
			Entry("above maximum", "6"),
			Entry("well above maximum", "1000"),
			Entry("negative", "-3"),
			// These wrap to 0 and 5 when narrowed to a 32-bit int.
			Entry("2^32 (wraps to 0 on 32-bit)", "4294967296"),
			Entry("2^32+5 (wraps to 5 on 32-bit)", "4294967301"),
		)

		DescribeTable("accepts ratings inside the 0-5 range",
			func(rating int) {
				if rating == 0 {
					Expect(ds.Playlist().SetRating(userCtx, 4, "pl-1")).To(Succeed())
				}
				r := newGetRequest("id=pl-1", fmt.Sprintf("rating=%d", rating)).WithContext(userCtx)

				_, err := router.SetRating(r)

				Expect(err).ToNot(HaveOccurred())
				Expect(storedPlaylist().Rating).To(Equal(rating))
			},
			Entry("zero clears the rating", 0),
			Entry("middle of the range", 3),
			Entry("maximum", 5),
		)
	})

	Describe("Star with an unresolvable id", func() {
		It("skips the id without broadcasting an empty (wildcard) refresh", func() {
			r := newGetRequest("id=does-not-exist")

			_, err := router.Star(r)

			Expect(err).ToNot(HaveOccurred())
			Expect(eventBroker.Events).To(BeEmpty())
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
