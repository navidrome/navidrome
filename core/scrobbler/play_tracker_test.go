package scrobbler

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("PlayTracker", func() {
	var ctx context.Context
	var ds model.DataStore
	var tracker PlayTracker
	var track model.MediaFile
	var album model.Album
	var artist model.Artist
	var fake fakeScrobbler

	BeforeEach(func() {
		// Remove buffering to simplify tests
		conf.Server.DevEnableBufferedScrobble = false

		ctx = context.Background()
		ctx = request.WithUser(ctx, model.User{ID: "u-1"})
		ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: true})
		ds = &tests.MockDataStore{}
		fake = fakeScrobbler{Authorized: true}
		Register("fake", func(ds model.DataStore) Scrobbler {
			return &fake
		})
		tracker = newPlayTracker(ds, events.GetBroker())

		track = model.MediaFile{
			ID:             "123",
			Title:          "Track Title",
			Album:          "Track Album",
			AlbumID:        "al-1",
			Artist:         "Track Artist",
			ArtistID:       "ar-1",
			AlbumArtist:    "Track AlbumArtist",
			TrackNumber:    1,
			Duration:       180,
			MbzRecordingID: "mbz-123",
		}
		_ = ds.MediaFile(ctx).Put(&track)
		artist = model.Artist{ID: "ar-1"}
		_ = ds.Artist(ctx).Put(&artist)
		album = model.Album{ID: "al-1"}
		_ = ds.Album(ctx).(*tests.MockAlbumRepo).Put(&album)
	})

	Describe("NowPlaying", func() {
		It("sends track to agent", func() {
			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123")
			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeTrue())
			Expect(fake.UserID).To(Equal("u-1"))
			Expect(fake.Track.ID).To(Equal("123"))
		})
		It("does not send track to agent if user has not authorized", func() {
			fake.Authorized = false

			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123")

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeFalse())
		})
		It("does not send track to agent if player is not enabled to send scrobbles", func() {
			ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: false})

			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123")

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeFalse())
		})
		It("does not send track to agent if artist is unknown", func() {
			track.Artist = consts.UnknownArtist

			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123")

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeFalse())
		})
	})

	Describe("GetNowPlaying", func() {
		It("returns current playing music", func() {
			track2 := track
			track2.ID = "456"
			_ = ds.MediaFile(ctx).Put(&track2)
			ctx = request.WithUser(context.Background(), model.User{UserName: "user-1"})
			_ = tracker.NowPlaying(ctx, "player-1", "player-one", "123")
			ctx = request.WithUser(context.Background(), model.User{UserName: "user-2"})
			_ = tracker.NowPlaying(ctx, "player-2", "player-two", "456")

			playing, err := tracker.GetNowPlaying(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(playing).To(HaveLen(2))
			Expect(playing[0].PlayerId).To(Equal("player-2"))
			Expect(playing[0].PlayerName).To(Equal("player-two"))
			Expect(playing[0].Username).To(Equal("user-2"))
			Expect(playing[0].MediaFile.ID).To(Equal("456"))

			Expect(playing[1].PlayerId).To(Equal("player-1"))
			Expect(playing[1].PlayerName).To(Equal("player-one"))
			Expect(playing[1].Username).To(Equal("user-1"))
			Expect(playing[1].MediaFile.ID).To(Equal("123"))
		})
	})

	Describe("Submit", func() {
		It("sends track to agent", func() {
			ctx = request.WithUser(ctx, model.User{ID: "u-1", UserName: "user-1"})
			ts := time.Now()

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: ts}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled).To(BeTrue())
			Expect(fake.UserID).To(Equal("u-1"))
			Expect(fake.LastScrobble.ID).To(Equal("123"))
		})

		It("increments play counts in the DB", func() {
			ctx = request.WithUser(ctx, model.User{ID: "u-1", UserName: "user-1"})
			ts := time.Now()

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: ts}})

			Expect(err).ToNot(HaveOccurred())
			Expect(track.PlayCount).To(Equal(int64(1)))
			Expect(album.PlayCount).To(Equal(int64(1)))
			Expect(artist.PlayCount).To(Equal(int64(1)))
		})

		It("does not send track to agent if user has not authorized", func() {
			fake.Authorized = false

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: time.Now()}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled).To(BeFalse())
		})

		It("does not send track to agent if player is not enabled to send scrobbles", func() {
			ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: false})

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: time.Now()}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled).To(BeFalse())
		})

		It("does not send track to agent if artist is unknown", func() {
			track.Artist = consts.UnknownArtist

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: time.Now()}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled).To(BeFalse())
		})

		It("increments play counts even if it cannot scrobble", func() {
			fake.Error = errors.New("error")

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: time.Now()}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled).To(BeFalse())

			Expect(track.PlayCount).To(Equal(int64(1)))
			Expect(album.PlayCount).To(Equal(int64(1)))
			Expect(artist.PlayCount).To(Equal(int64(1)))
		})

	})

})

type fakeScrobbler struct {
	Authorized       bool
	NowPlayingCalled bool
	ScrobbleCalled   bool
	UserID           string
	Track            *model.MediaFile
	LastScrobble     Scrobble
	Error            error
}

func (f *fakeScrobbler) IsAuthorized(ctx context.Context, userId string) bool {
	return f.Error == nil && f.Authorized
}

func (f *fakeScrobbler) NowPlaying(ctx context.Context, userId string, track *model.MediaFile) error {
	f.NowPlayingCalled = true
	if f.Error != nil {
		return f.Error
	}
	f.UserID = userId
	f.Track = track
	return nil
}

func (f *fakeScrobbler) Scrobble(ctx context.Context, userId string, s Scrobble) error {
	f.ScrobbleCalled = true
	if f.Error != nil {
		return f.Error
	}
	f.UserID = userId
	f.LastScrobble = s
	return nil
}
