package subsonic

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Album Lists", func() {
	var router *Router
	var ds model.DataStore
	var mockRepo *tests.MockAlbumRepo
	var mockTracker *mockPlayTracker
	var w *httptest.ResponseRecorder
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		mockRepo = ds.Album(ctx).(*tests.MockAlbumRepo)
		mockTracker = &mockPlayTracker{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, mockTracker, nil, nil, nil)
		w = httptest.NewRecorder()
	})

	Describe("GetAlbumList", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			mockRepo.SetData(model.Albums{
				{ID: "1"}, {ID: "2"},
			})
			resp, err := router.GetAlbumList(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList.Album[1].Id).To(Equal("2"))
			Expect(w.Header().Get("x-total-count")).To(Equal("2"))
			Expect(mockRepo.Options.Offset).To(Equal(10))
			Expect(mockRepo.Options.Max).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()
			_, err := router.GetAlbumList(w, r)

			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("should return error if call fails", func() {
			mockRepo.SetError(true)
			r := newGetRequest("type=newest")

			_, err := router.GetAlbumList(w, r)

			Expect(err).To(MatchError(errSubsonic))
			var subErr subError
			errors.As(err, &subErr)
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})
	})

	Describe("GetAlbumList2", func() {
		It("should return list of the type specified", func() {
			r := newGetRequest("type=newest", "offset=10", "size=20")
			mockRepo.SetData(model.Albums{
				{ID: "1"}, {ID: "2"},
			})
			resp, err := router.GetAlbumList2(w, r)

			Expect(err).ToNot(HaveOccurred())
			Expect(resp.AlbumList2.Album[0].Id).To(Equal("1"))
			Expect(resp.AlbumList2.Album[1].Id).To(Equal("2"))
			Expect(w.Header().Get("x-total-count")).To(Equal("2"))
			Expect(mockRepo.Options.Offset).To(Equal(10))
			Expect(mockRepo.Options.Max).To(Equal(20))
		})

		It("should fail if missing type parameter", func() {
			r := newGetRequest()

			_, err := router.GetAlbumList2(w, r)

			Expect(err).To(MatchError(req.ErrMissingParam))
		})

		It("should return error if call fails", func() {
			mockRepo.SetError(true)
			r := newGetRequest("type=newest")

			_, err := router.GetAlbumList2(w, r)

			Expect(err).To(MatchError(errSubsonic))
			var subErr subError
			errors.As(err, &subErr)
			Expect(subErr.code).To(Equal(responses.ErrorGeneric))
		})
	})

	Describe("GetNowPlaying", func() {
		var r *http.Request
		var baseResponse *responses.NowPlaying

		BeforeEach(func() {
			now := time.Now().Add(-1 * time.Minute)

			episode := model.PodcastEpisode{
				Annotations: model.Annotations{
					PlayCount: 1,
					PlayDate:  &time.Time{},
					Rating:    1,
					Starred:   true,
					StarredAt: &time.Time{},
				},
				ID:          "3333",
				Guid:        "3333",
				PodcastId:   "4444",
				Url:         "https://example.org",
				Title:       "Podcast title",
				Description: "Podcast description",
				Suffix:      "mp3",
				BitRate:     320,
				State:       consts.PodcastStatusCompleted,
			}

			mockTracker.data = []scrobbler.NowPlayingInfo{
				{
					MediaFile:  *episode.ToMediaFile(),
					Start:      now,
					Username:   "a",
					PlayerId:   "a",
					PlayerName: "a",
				},
				{
					MediaFile: model.MediaFile{
						Annotations: model.Annotations{
							PlayCount: 5,
							PlayDate:  &time.Time{},
							Rating:    3,
							Starred:   true,
							StarredAt: &time.Time{},
						},
						Bookmarkable: model.Bookmarkable{BookmarkPosition: 125},
						ID:           "1234",
						Title:        "Title",
						Album:        "Album",
						AlbumID:      "1234",
						Year:         2024,
						Artist:       "Artist",
						ArtistID:     "1234",
						Genre:        "Genre",
						Genres: model.Genres{
							{Name: "Genre"},
							{Name: "Genre 2"},
						},
						Path:           "this/is/a/fake/path.mp3",
						TrackNumber:    3,
						Duration:       33.5,
						Size:           1234,
						Suffix:         "mp3",
						BitRate:        128,
						DiscNumber:     1,
						Comment:        "Comment",
						Bpm:            100,
						MbzRecordingID: "1234",
						RgAlbumGain:    1,
						RgAlbumPeak:    1,
						RgTrackGain:    1,
						RgTrackPeak:    1,
						Channels:       2,
						SampleRate:     48000,
					},
					Start:      now,
					Username:   "b",
					PlayerId:   "b",
					PlayerName: "b",
				},
			}

			r = newGetRequest("")

			baseResponse = &responses.NowPlaying{
				Entry: []responses.NowPlayingEntry{
					{
						Child: responses.Child{
							Id:          "pe-3333",
							Title:       "Podcast title",
							Parent:      "pd-4444",
							Path:        "//Podcast title.mp3",
							CoverArt:    "pe-3333",
							ContentType: "audio/mpeg",
							Suffix:      "mp3",
							Starred:     &time.Time{},
							BitRate:     320,
							PlayCount:   1,
							Played:      &time.Time{},
							Created:     &time.Time{},
							AlbumId:     "pd-4444",
							Type:        "podcast",
							UserRating:  1,
							MediaType:   "podcast",
							Genres:      responses.ItemGenres{},
						},
						UserName:   "a",
						PlayerId:   1,
						PlayerName: "a",
						MinutesAgo: 1,
					},
					{
						Child: responses.Child{
							Id:               "1234",
							Parent:           "1234",
							Title:            "Title",
							Album:            "Album",
							AlbumId:          "1234",
							Artist:           "Artist",
							ArtistId:         "1234",
							Track:            3,
							Year:             2024,
							Genre:            "Genre",
							CoverArt:         "al-1234_0",
							Size:             1234,
							ContentType:      "audio/mpeg",
							Suffix:           "mp3",
							Starred:          &time.Time{},
							Duration:         33,
							BitRate:          128,
							Path:             "/Album/03 - Title.mp3",
							PlayCount:        5,
							Played:           &time.Time{},
							Created:          &time.Time{},
							Type:             "music",
							UserRating:       3,
							BookmarkPosition: 125,
							DiscNumber:       1,
							Bpm:              100,
							Comment:          "Comment",
							MediaType:        "song",
							MusicBrainzId:    "1234",
							Genres:           responses.ItemGenres{{Name: "Genre"}, {Name: "Genre 2"}},
							ReplayGain: responses.ReplayGain{
								TrackGain: 1,
								AlbumGain: 1,
								TrackPeak: 1,
								AlbumPeak: 1,
							},
							ChannelCount: 2,
							SamplingRate: 48000,
						},
						UserName:   "b",
						PlayerId:   2,
						PlayerName: "b",
						MinutesAgo: 1,
					},
				},
			}
		})

		It("should return podcast and media file", func() {
			resp, err := router.GetNowPlaying(r)
			Expect(err).To(BeNil())
			Expect(resp.NowPlaying).To(Equal(baseResponse))
		})

		It("should return real path", func() {
			player := model.Player{ReportRealPath: true}
			ctx := request.WithPlayer(ctx, player)
			r := r.WithContext(ctx)

			resp, err := router.GetNowPlaying(r)
			Expect(err).To(BeNil())
			baseResponse.Entry[0].Path = "4444/3333.mp3"
			baseResponse.Entry[1].Path = "this/is/a/fake/path.mp3"
			Expect(resp.NowPlaying).To(Equal(baseResponse))
		})
	})
})

type mockPlayTracker struct {
	scrobbler.PlayTracker
	data []scrobbler.NowPlayingInfo
	err  error
}

func (m *mockPlayTracker) GetNowPlaying(ctx context.Context) ([]scrobbler.NowPlayingInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.data, nil
}
