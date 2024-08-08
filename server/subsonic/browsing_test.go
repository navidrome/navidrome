package subsonic

import (
	"context"
	"net/http"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Browsing", func() {
	var router *Router
	var ds model.DataStore
	var r *http.Request
	ctx := log.NewContext(context.TODO())

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		router = New(ds, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

		_ = ds.PodcastEpisode(ctx).Put(&model.PodcastEpisode{
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
		})

		_ = ds.PodcastEpisode(ctx).Put(&model.PodcastEpisode{
			ID:        "empty",
			Title:     "Podcast title",
			PodcastId: "4444",
		})

		_ = ds.Podcast(ctx).Put(&model.Podcast{
			Annotations: model.Annotations{
				PlayCount: 10,
				PlayDate:  &time.Time{},
				Rating:    2,
				Starred:   true,
				StarredAt: &time.Time{},
			},
			ID:          "4444",
			Url:         "https://example.org/feed.rss",
			Title:       "podcast",
			Description: "Description",
			ImageUrl:    "https://example.org/image.png",
		})

		_ = ds.MediaFile(ctx).Put(&model.MediaFile{
			Annotations: model.Annotations{
				PlayCount: 5,
				PlayDate:  &time.Time{},
				Rating:    3,
				Starred:   true,
				StarredAt: &time.Time{},
			},
			Bookmarkable: model.Bookmarkable{BookmarkPosition: 125},
			ID:           "mf-1",
			Title:        "Title",
			Album:        "full-album",
			AlbumID:      "al-1",
			Year:         2024,
			Artist:       "artist",
			ArtistID:     "ar-1",
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
		})
		_ = ds.MediaFile(ctx).Put(&model.MediaFile{ID: "mf-2"})

		_ = ds.Album(ctx).Put(&model.Album{
			Annotations: model.Annotations{
				PlayCount: 5,
				PlayDate:  &time.Time{},
				Rating:    3,
				Starred:   true,
				StarredAt: &time.Time{},
			},
			ID:                   "al-1",
			LibraryID:            1,
			Name:                 "full-album",
			Artist:               "artist",
			AlbumArtist:          "artist",
			ArtistID:             "ar-1",
			AlbumArtistID:        "ar-1",
			MaxYear:              2024,
			MinYear:              2024,
			Date:                 "date",
			MaxOriginalYear:      2024,
			MinOriginalYear:      2024,
			OriginalDate:         "original",
			ReleaseDate:          "release",
			Releases:             5,
			Comment:              "comment",
			Duration:             1234,
			Size:                 1234,
			Genre:                "genre",
			Genres:               model.Genres{{Name: "1"}, {Name: "2"}},
			Discs:                model.Discs{1: "1", 2: "2"},
			SortAlbumName:        "1",
			SortArtistName:       "artist",
			SortAlbumArtistName:  "artist",
			OrderAlbumName:       "1",
			OrderAlbumArtistName: "artist",
			CatalogNum:           "2345678",
			MbzAlbumID:           "mbz",
			MbzAlbumArtistID:     "mbz",
			MbzAlbumType:         "official",
			MbzAlbumComment:      "comment",
			Description:          "album description",
			SmallImageUrl:        "small",
			MediumImageUrl:       "medium",
			LargeImageUrl:        "large",
			ExternalUrl:          "https://example.org",
			CreatedAt:            time.Time{}.Add(1 * time.Minute),
			UpdatedAt:            time.Time{}.Add(1 * time.Minute),
		})
		_ = ds.Album(ctx).Put(&model.Album{
			ID:       "al-2",
			Name:     "2",
			ArtistID: "ar-1",
		})

		_ = ds.Artist(ctx).Put(&model.Artist{
			Annotations: model.Annotations{
				PlayCount: 11,
				PlayDate:  &time.Time{},
				Rating:    2,
				Starred:   true,
				StarredAt: &time.Time{},
			},
			ID:   "ar-1",
			Name: "artist",
		})

		conf.Server.Podcast.Enabled = true
	})

	Describe("GetMusicDirectory", func() {
		created := time.Time{}.Add(1 * time.Minute)

		DescribeTable("GetMusicDirectory", func(id string, expected *responses.Directory, expectedErr error) {
			r = newGetRequest("id=" + id)
			resp, err := router.GetMusicDirectory(r)

			if expectedErr != nil {
				Expect(err).To(Equal(expectedErr))
			} else {
				Expect(expectedErr).To(BeNil())
			}

			if expected != nil {
				Expect(resp).ToNot(BeNil())
				Expect(resp.Directory).To(Equal(expected))
			} else {
				Expect(resp).To(BeNil())
			}
		},
			Entry("Artist with data", "ar-1", &responses.Directory{
				Child: []responses.Child{
					{
						Id:            "al-1",
						Parent:        "ar-1",
						IsDir:         true,
						Title:         "full-album",
						Name:          "full-album",
						Album:         "full-album",
						Artist:        "artist",
						Year:          2024,
						Genre:         "genre",
						CoverArt:      "al-al-1_0",
						Starred:       &time.Time{},
						Duration:      1234,
						PlayCount:     5,
						Created:       &created,
						ArtistId:      "ar-1",
						UserRating:    3,
						Played:        &time.Time{},
						SortName:      "1",
						MediaType:     "album",
						MusicBrainzId: "mbz",
						Genres:        responses.ItemGenres{{Name: "1"}, {Name: "2"}},
					},
					{
						Id:        "al-2",
						IsDir:     true,
						Title:     "2",
						Name:      "2",
						Album:     "2",
						CoverArt:  "al-al-2_0",
						Created:   &time.Time{},
						MediaType: "album",
						Genres:    responses.ItemGenres{},
					},
				},
				Id:         "ar-1",
				Name:       "artist",
				Starred:    &time.Time{},
				PlayCount:  11,
				Played:     &time.Time{},
				UserRating: 2,
			}, nil),
			Entry("album", "al-1", &responses.Directory{
				Child: []responses.Child{
					{
						Id:               "mf-1",
						Parent:           "al-1",
						Title:            "Title",
						Album:            "full-album",
						AlbumId:          "al-1",
						Artist:           "artist",
						ArtistId:         "ar-1",
						Track:            3,
						Year:             2024,
						Genre:            "Genre",
						CoverArt:         "al-al-1_0",
						Size:             1234,
						ContentType:      "audio/mpeg",
						Suffix:           "mp3",
						Starred:          &time.Time{},
						Duration:         33,
						BitRate:          128,
						Path:             "/full-album/03 - Title.mp3",
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
					{
						Id:        "mf-2",
						Path:      "//.",
						Created:   &time.Time{},
						Type:      "music",
						MediaType: "song",
						Genres:    responses.ItemGenres{},
					},
				},
				Id:         "al-1",
				Parent:     "ar-1",
				Name:       "full-album",
				CoverArt:   "al-al-1_0",
				Starred:    &time.Time{},
				PlayCount:  5,
				UserRating: 3,
				Played:     &time.Time{},
			}, nil),
			Entry("podcast", "pd-4444", &responses.Directory{
				Child: []responses.Child{
					{
						Id:               "pe-3333",
						Parent:           "pd-4444",
						IsDir:            false,
						Title:            "Podcast title",
						CoverArt:         "pe-3333_0",
						Size:             1000,
						ContentType:      "audio/mpeg",
						Suffix:           "mp3",
						Starred:          &time.Time{},
						Duration:         100,
						BitRate:          320,
						Path:             "4444/3333.mp3",
						PlayCount:        1,
						Created:          &time.Time{},
						AlbumId:          "",
						ArtistId:         "",
						Type:             "podcast",
						UserRating:       1,
						SongCount:        0,
						IsVideo:          false,
						BookmarkPosition: 0,
						Played:           &time.Time{},
						Bpm:              0,
						Year:             1,
					},
					{
						Id:       "pe-empty",
						Parent:   "pd-4444",
						Title:    "Podcast title",
						CoverArt: "pe-empty_0",
						Created:  &time.Time{},
						Type:     "podcast",
					},
				},
				Id:         "4444",
				Name:       "podcast",
				Starred:    &time.Time{},
				PlayCount:  10,
				Played:     &time.Time{},
				UserRating: 2,
				CoverArt:   "pd-4444_0",
				Genre:      "",
			}, nil),
			Entry("nonexistent item", "fake", nil, nil))
	})
})
