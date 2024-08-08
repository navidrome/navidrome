package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Browsing", func() {
	var router *Router
	var ds model.DataStore
	var r *http.Request
	ctx := request.WithUser(context.TODO(), model.User{ID: "1234"})

	var fullEpisode model.PodcastEpisode
	var fullMediafile model.MediaFile

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		fullEpisode = model.PodcastEpisode{
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
			ImageUrl:    "https://example.org/ep.png",
			Description: "Podcast description",
			PublishDate: &time.Time{},
			Suffix:      "mp3",
			Duration:    100,
			BitRate:     320,
			Size:        1000,
			State:       consts.PodcastStatusCompleted,
		}

		_ = ds.PodcastEpisode(ctx).Put(&fullEpisode)

		fullMediafile = model.MediaFile{
			Annotations: model.Annotations{
				PlayCount: 5,
				PlayDate:  &time.Time{},
				Rating:    3,
				Starred:   true,
				StarredAt: &time.Time{},
			},
			ID:       "1234",
			Title:    "Title",
			Album:    "full-album",
			AlbumID:  "al-1",
			Year:     2024,
			Artist:   "artist",
			ArtistID: "ar-1",
			Genre:    "Genre",
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
		}

		_ = ds.MediaFile(ctx).Put(&fullMediafile)
		_ = ds.MediaFile(ctx).Put(&model.MediaFile{ID: "mf-2"})
		conf.Server.Podcast.Enabled = true
	})

	Describe("CreateBookmark", func() {
		// Note: In practice, more would be updated in the backend. For simplicity,
		// the mock will JUST update bookmark position
		DescribeTable("create bookmark", func(id string, position, epPosition, mfPosition int64) {
			r = newGetRequest(fmt.Sprintf("id=%s&position=%d", id, position))
			resp, err := router.CreateBookmark(r)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(newResponse()))

			Expect(fullEpisode.BookmarkPosition).To(Equal(epPosition))
			Expect(fullMediafile.BookmarkPosition).To(Equal(mfPosition))
		},
			Entry("podcast", "pe-3333", int64(4), int64(4), int64(0)),
			Entry("mediafile", "1234", int64(100), int64(0), int64(100)))
	})

	Describe("DeleteBookmark", func() {
		DescribeTable("delete bookmark", func(id string) {
			fullEpisode.BookmarkPosition = 123
			fullMediafile.BookmarkPosition = 321
			r = newGetRequest("id=" + id)
			resp, err := router.DeleteBookmark(r)
			Expect(err).To(BeNil())
			Expect(resp).To(Equal(newResponse()))

			if model.IsPodcastEpisodeId(id) {
				Expect(fullEpisode.BookmarkPosition).To(Equal(int64(0)))
				Expect(fullMediafile.BookmarkPosition).To(Equal(int64(321)))
			} else {
				Expect(fullEpisode.BookmarkPosition).To(Equal(int64(123)))
				Expect(fullMediafile.BookmarkPosition).To(Equal(int64(0)))
			}
		},
			Entry("podcast", "pe-3333"),
			Entry("mediafile", "1234"))
	})
})
