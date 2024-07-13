package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
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
		router = New(ds, nil, nil, nil, nil, nil, nil, eventBroker, nil, playTracker, nil, nil, nil)
	})

	Context("Annotations", func() {
		var album model.Album
		var artist model.Artist
		var file model.MediaFile
		var podcast model.Podcast
		var episode model.PodcastEpisode
		var start time.Time

		ids := []string{"1111", "2222", "3333", "pd-123", "pe-321"}

		BeforeEach(func() {
			album = model.Album{ID: "1111"}
			_ = ds.Album(ctx).Put(&album)

			artist = model.Artist{ID: "2222"}
			_ = ds.Artist(ctx).Put(&artist)

			file = model.MediaFile{ID: "3333"}
			_ = ds.MediaFile(ctx).Put(&file)

			podcast = model.Podcast{ID: "123"}
			_ = ds.Podcast(ctx).Put(&podcast)

			episode = model.PodcastEpisode{ID: "321", PodcastId: "123"}
			_ = ds.PodcastEpisode(ctx).Put(&episode)

			start = time.Now()
		})

		Describe("Star/Unstar", func() {
			DescribeTable("star/unstar", func(id string) {
				r := newGetRequest("id=" + id)

				for _, doStar := range []bool{true, false} {
					var err error
					if doStar {
						_, err = router.Star(r)
					} else {
						_, err = router.Unstar(r)
					}
					if id == "fake" {
						Expect(err).To(Equal(model.ErrNotFound))
					} else {
						Expect(err).ToNot(HaveOccurred())
					}

					var starred bool
					var starredAt *time.Time

					for _, itemId := range ids {
						item, err := model.GetEntityByID(ctx, ds, itemId)
						Expect(err).To(BeNil())

						switch item := item.(type) {
						case *model.Album:
							starred = item.Starred
							starredAt = item.StarredAt
						case *model.Artist:
							starred = item.Starred
							starredAt = item.StarredAt
						case *model.MediaFile:
							starred = item.Starred
							starredAt = item.StarredAt
						case *model.Podcast:
							starred = item.Starred
							starredAt = item.StarredAt
						case *model.PodcastEpisode:
							starred = item.Starred
							starredAt = item.StarredAt
						default:
							Fail("Unexpected item type")
						}

						if itemId == id {
							Expect(starred).To(Equal(doStar))
							Expect(*starredAt).To(BeTemporally(">", start))
						} else {
							Expect(starred).To(BeFalse())
							Expect(starredAt).To(BeNil())
						}
					}
				}

			},
				Entry("album", "1111"),
				Entry("artist", "2222"),
				Entry("media file", "3333"),
				Entry("podcast", "pd-123"),
				Entry("podcast episode", "pe-321"),
				Entry("fake", "fake"))
		})

		Describe("Ratings", func() {
			DescribeTable("star/unstar", func(id string) {
				r := newGetRequest("id=" + id + "&rating=3")

				_, err := router.SetRating(r)
				if id == "fake" {
					Expect(err).To(Equal(model.ErrNotFound))
				} else {
					Expect(err).ToNot(HaveOccurred())
				}

				var rating int

				for _, itemId := range ids {
					item, err := model.GetEntityByID(ctx, ds, itemId)
					Expect(err).To(BeNil())

					switch item := item.(type) {
					case *model.Album:
						rating = item.Rating
					case *model.Artist:
						rating = item.Rating
					case *model.MediaFile:
						rating = item.Rating
					case *model.Podcast:
						rating = item.Rating
					case *model.PodcastEpisode:
						rating = item.Rating
					default:
						Fail("Unexpected item type")
					}

					if itemId == id {
						Expect(rating).To(Equal(3))
					} else {
						Expect(rating).To(Equal(0))
					}
				}
			},
				Entry("album", "1111"),
				Entry("artist", "2222"),
				Entry("media file", "3333"),
				Entry("podcast", "pd-123"),
				Entry("podcast episode", "pe-321"),
				Entry("fake", "fake"))
		})
	})

	Describe("Scrobble", func() {
		It("submit all scrobbles with only the id", func() {
			submissionTime := time.Now()
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
				_ = ds.PodcastEpisode(ctx).Put(&model.PodcastEpisode{ID: "12"})
				ctx = request.WithPlayer(ctx, model.Player{ID: "player-1"})
				req = newGetRequest("id=12", "submission=false")
				req = req.WithContext(ctx)
			})

			It("does not scrobble", func() {
				_, err := router.Scrobble(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.Submissions).To(BeEmpty())
			})

			It("registers a NowPlaying", func() {
				_, err := router.Scrobble(req)

				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.Playing).To(HaveLen(1))
				Expect(playTracker.Playing).To(HaveKey("player-1"))
			})

			It("does not register NowPlaying for podcast", func() {
				ctx = request.WithPlayer(ctx, model.Player{ID: "player-1"})
				req = newGetRequest("id=pe-12", "submission=false")
				req = req.WithContext(ctx)

				_, err := router.Scrobble(req)
				Expect(err).ToNot(HaveOccurred())
				Expect(playTracker.Playing).To(HaveLen(0))
			})
		})
	})
})

type fakePlayTracker struct {
	Submissions []scrobbler.Submission
	Playing     map[string]string
	Error       error
}

func (f *fakePlayTracker) NowPlaying(_ context.Context, playerId string, _ string, trackId string) error {
	if f.Error != nil {
		return f.Error
	}
	if f.Playing == nil {
		f.Playing = make(map[string]string)
	}
	f.Playing[playerId] = trackId
	return nil
}

func (f *fakePlayTracker) GetNowPlaying(_ context.Context) ([]scrobbler.NowPlayingInfo, error) {
	return nil, f.Error
}

func (f *fakePlayTracker) Submit(_ context.Context, submissions []scrobbler.Submission) error {
	if f.Error != nil {
		return f.Error
	}
	f.Submissions = append(f.Submissions, submissions...)
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

var _ events.Broker = (*fakeEventBroker)(nil)
