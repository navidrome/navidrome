//go:build !windows

package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MatcherService", Ordered, func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	Describe("toTrack", func() {
		It("projects a MediaFile into a public Track", func() {
			bitDepth := 24
			bpm := 128
			rgGain := -7.5
			created := time.Unix(1700000000, 0)
			updated := time.Unix(1700000500, 0)
			birth := time.Unix(1699999000, 0)

			mf := &model.MediaFile{
				ID:             "mf-1",
				LibraryID:      3,
				LibraryName:    "Main",
				Path:           "/music/song.flac",
				Title:          "My Song",
				Album:          "My Album",
				Artist:         "My Artist",
				AlbumArtist:    "My Artist",
				AlbumID:        "al-1",
				SortTitle:      "my song",
				TrackNumber:    4,
				DiscNumber:     1,
				Year:           2020,
				Size:           1234,
				Suffix:         "flac",
				Duration:       210.5,
				BitRate:        1000,
				SampleRate:     44100,
				BitDepth:       &bitDepth,
				Channels:       2,
				Codec:          "flac",
				Genre:          "Rock",
				BPM:            &bpm,
				ExplicitStatus: "c",
				Compilation:    true,
				HasCoverArt:    true,
				MbzRecordingID: "rec-1",
				RGTrackGain:    &rgGain,
				CreatedAt:      created,
				UpdatedAt:      updated,
				BirthTime:      birth,
				Genres:         model.Genres{{Name: "Rock"}, {Name: "Pop"}},
				Tags:           model.Tags{model.TagName("isrc"): []string{"US-XXX-00"}},
			}
			mf.Participants = model.Participants{}
			mf.Participants.Add(model.RoleArtist, model.Artist{
				ID: "ar-1", Name: "My Artist", SortArtistName: "artist, my", MbzArtistID: "mbz-ar-1",
			})

			track := toTrack(mf)

			Expect(track.ID).To(Equal("mf-1"))
			Expect(track.LibraryID).To(Equal(int32(3)))
			Expect(track.LibraryName).To(Equal("Main"))
			Expect(track.Title).To(Equal("My Song"))
			Expect(track.Duration).To(Equal(210.5))
			Expect(track.BitDepth).To(Equal(int32(24)))
			Expect(track.BPM).To(Equal(int32(128)))
			Expect(track.RGTrackGain).To(Equal(-7.5))
			Expect(track.Compilation).To(BeTrue())
			Expect(track.MbzRecordingID).To(Equal("rec-1"))
			Expect(track.Genres).To(Equal([]string{"Rock", "Pop"}))
			Expect(track.CreatedAt).To(Equal(int64(1700000000)))
			Expect(track.UpdatedAt).To(Equal(int64(1700000500)))
			Expect(track.BirthTime).To(Equal(int64(1699999000)))
			Expect(track.Tags).To(HaveKeyWithValue("isrc", []string{"US-XXX-00"}))
			Expect(track.Participants).To(HaveKey("artist"))
			Expect(track.Participants["artist"]).To(HaveLen(1))
			Expect(track.Participants["artist"][0].ID).To(Equal("ar-1"))
			Expect(track.Participants["artist"][0].Name).To(Equal("My Artist"))
			Expect(track.Participants["artist"][0].SortName).To(Equal("artist, my"))
			Expect(track.Participants["artist"][0].MbzArtistID).To(Equal("mbz-ar-1"))
		})

		It("omits nil-able numeric fields when absent", func() {
			mf := &model.MediaFile{ID: "mf-2", Title: "No Optionals"}
			track := toTrack(mf)
			Expect(track.BitDepth).To(Equal(int32(0)))
			Expect(track.BPM).To(Equal(int32(0)))
			Expect(track.RGTrackGain).To(Equal(float64(0)))
		})
	})

	Describe("MatchSongs", func() {
		It("returns one entry per input song in order, with nil for no-match", func() {
			mediaFileRepo := tests.CreateMockMediaFileRepo()
			// First (ID) phase returns the match for input song 0 only.
			mediaFileRepo.SetData(model.MediaFiles{
				{ID: "mf-100", Title: "Hit", Artist: "Band"},
			})
			ds := &tests.MockDataStore{MockedMediaFile: mediaFileRepo}

			svc := newMatcherService(ds)
			results, err := svc.MatchSongs(context.Background(), []host.MatchSong{
				{ID: "mf-100", Name: "Hit", Artist: "Band"},
				{ID: "missing-id", Name: "Ghost", Artist: "Nobody"},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0]).ToNot(BeNil())
			Expect(results[0].ID).To(Equal("mf-100"))
			Expect(results[1]).To(BeNil())
		})

		It("returns an empty slice for empty input", func() {
			ds := &tests.MockDataStore{MockedMediaFile: tests.CreateMockMediaFileRepo()}
			svc := newMatcherService(ds)
			results, err := svc.MatchSongs(context.Background(), nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(BeEmpty())
		})
	})
})
