package scrobbler

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/events"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockPluginLoader is a test implementation of PluginLoader for plugin scrobbler tests
// Moved to top-level scope to avoid linter issues

type mockPluginLoader struct {
	names      []string
	scrobblers map[string]Scrobbler
}

func (m *mockPluginLoader) PluginNames(service string) []string {
	return m.names
}

func (m *mockPluginLoader) LoadScrobbler(name string) (Scrobbler, bool) {
	s, ok := m.scrobblers[name]
	return s, ok
}

var _ = Describe("PlayTracker", func() {
	var ctx context.Context
	var ds model.DataStore
	var tracker PlayTracker
	var eventBroker *fakeEventBroker
	var track model.MediaFile
	var album model.Album
	var artist1 model.Artist
	var artist2 model.Artist
	var fake fakeScrobbler

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		ctx = request.WithUser(ctx, model.User{ID: "u-1"})
		ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: true})
		ds = &tests.MockDataStore{}
		fake = fakeScrobbler{Authorized: true}
		Register("fake", func(model.DataStore) Scrobbler {
			return &fake
		})
		Register("disabled", func(model.DataStore) Scrobbler {
			return nil
		})
		eventBroker = &fakeEventBroker{}
		tracker = newPlayTracker(ds, eventBroker, nil)
		tracker.(*playTracker).builtinScrobblers["fake"] = &fake // Bypass buffering for tests

		track = model.MediaFile{
			ID:             "123",
			Title:          "Track Title",
			Album:          "Track Album",
			AlbumID:        "al-1",
			TrackNumber:    1,
			Duration:       180,
			MbzRecordingID: "mbz-123",
			Participants: map[model.Role]model.ParticipantList{
				model.RoleArtist: []model.Participant{_p("ar-1", "Artist 1"), _p("ar-2", "Artist 2")},
			},
		}
		_ = ds.MediaFile(ctx).Put(&track)
		artist1 = model.Artist{ID: "ar-1"}
		_ = ds.Artist(ctx).Put(&artist1)
		artist2 = model.Artist{ID: "ar-2"}
		_ = ds.Artist(ctx).Put(&artist2)
		album = model.Album{ID: "al-1"}
		_ = ds.Album(ctx).(*tests.MockAlbumRepo).Put(&album)
	})

	It("does not register disabled scrobblers", func() {
		Expect(tracker.(*playTracker).builtinScrobblers).To(HaveKey("fake"))
		Expect(tracker.(*playTracker).builtinScrobblers).ToNot(HaveKey("disabled"))
	})

	Describe("NowPlaying", func() {
		It("sends track to agent", func() {
			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeTrue())
			Expect(fake.UserID).To(Equal("u-1"))
			Expect(fake.Track.ID).To(Equal("123"))
			Expect(fake.Track.Participants).To(Equal(track.Participants))
		})
		It("does not send track to agent if user has not authorized", func() {
			fake.Authorized = false

			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeFalse())
		})
		It("does not send track to agent if player is not enabled to send scrobbles", func() {
			ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: false})

			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeFalse())
		})
		It("does not send track to agent if artist is unknown", func() {
			track.Artist = consts.UnknownArtist

			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeFalse())
		})

		It("stores position when greater than zero", func() {
			pos := 42
			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", pos)
			Expect(err).ToNot(HaveOccurred())

			playing, err := tracker.GetNowPlaying(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(playing).To(HaveLen(1))
			Expect(playing[0].Position).To(Equal(pos))
			Expect(fake.Position).To(Equal(pos))
		})

		It("sends event with count", func() {
			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)
			Expect(err).ToNot(HaveOccurred())
			eventList := eventBroker.getEvents()
			Expect(eventList).ToNot(BeEmpty())
			evt, ok := eventList[0].(*events.NowPlayingCount)
			Expect(ok).To(BeTrue())
			Expect(evt.Count).To(Equal(1))
		})

		It("does not send event when disabled", func() {
			conf.Server.EnableNowPlaying = false
			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(eventBroker.getEvents()).To(BeEmpty())
		})
	})

	Describe("GetNowPlaying", func() {
		It("returns current playing music", func() {
			track2 := track
			track2.ID = "456"
			_ = ds.MediaFile(ctx).Put(&track2)
			ctx = request.WithUser(context.Background(), model.User{UserName: "user-1"})
			_ = tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)
			ctx = request.WithUser(context.Background(), model.User{UserName: "user-2"})
			_ = tracker.NowPlaying(ctx, "player-2", "player-two", "456", 0)

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

	Describe("Expiration events", func() {
		It("sends event when entry expires", func() {
			info := NowPlayingInfo{MediaFile: track, Start: time.Now(), Username: "user"}
			_ = tracker.(*playTracker).playMap.AddWithTTL("player-1", info, 10*time.Millisecond)
			Eventually(func() int { return len(eventBroker.getEvents()) }).Should(BeNumerically(">", 0))
			eventList := eventBroker.getEvents()
			evt, ok := eventList[len(eventList)-1].(*events.NowPlayingCount)
			Expect(ok).To(BeTrue())
			Expect(evt.Count).To(Equal(0))
		})

		It("does not send event when disabled", func() {
			conf.Server.EnableNowPlaying = false
			tracker = newPlayTracker(ds, eventBroker, nil)
			info := NowPlayingInfo{MediaFile: track, Start: time.Now(), Username: "user"}
			_ = tracker.(*playTracker).playMap.AddWithTTL("player-2", info, 10*time.Millisecond)
			Consistently(func() int { return len(eventBroker.getEvents()) }).Should(Equal(0))
		})
	})

	Describe("Submit", func() {
		It("sends track to agent", func() {
			ctx = request.WithUser(ctx, model.User{ID: "u-1", UserName: "user-1"})
			ts := time.Now()

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: ts}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled.Load()).To(BeTrue())
			Expect(fake.UserID).To(Equal("u-1"))
			lastScrobble := fake.LastScrobble.Load()
			Expect(lastScrobble.TimeStamp).To(BeTemporally("~", ts, 1*time.Second))
			Expect(lastScrobble.ID).To(Equal("123"))
			Expect(lastScrobble.Participants).To(Equal(track.Participants))
		})

		It("increments play counts in the DB", func() {
			ctx = request.WithUser(ctx, model.User{ID: "u-1", UserName: "user-1"})
			ts := time.Now()

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: ts}})

			Expect(err).ToNot(HaveOccurred())
			Expect(track.PlayCount).To(Equal(int64(1)))
			Expect(album.PlayCount).To(Equal(int64(1)))

			// It should increment play counts for all artists
			Expect(artist1.PlayCount).To(Equal(int64(1)))
			Expect(artist2.PlayCount).To(Equal(int64(1)))
		})

		It("does not send track to agent if user has not authorized", func() {
			fake.Authorized = false

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: time.Now()}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled.Load()).To(BeFalse())
		})

		It("does not send track to agent if player is not enabled to send scrobbles", func() {
			ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: false})

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: time.Now()}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled.Load()).To(BeFalse())
		})

		It("does not send track to agent if artist is unknown", func() {
			track.Artist = consts.UnknownArtist

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: time.Now()}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled.Load()).To(BeFalse())
		})

		It("increments play counts even if it cannot scrobble", func() {
			fake.Error = errors.New("error")

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: time.Now()}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled.Load()).To(BeFalse())

			Expect(track.PlayCount).To(Equal(int64(1)))
			Expect(album.PlayCount).To(Equal(int64(1)))

			// It should increment play counts for all artists
			Expect(artist1.PlayCount).To(Equal(int64(1)))
			Expect(artist2.PlayCount).To(Equal(int64(1)))
		})
	})

	Describe("Plugin scrobbler logic", func() {
		var pluginLoader *mockPluginLoader
		var pluginFake fakeScrobbler

		BeforeEach(func() {
			pluginFake = fakeScrobbler{Authorized: true}
			pluginLoader = &mockPluginLoader{
				names:      []string{"plugin1"},
				scrobblers: map[string]Scrobbler{"plugin1": &pluginFake},
			}
			tracker = newPlayTracker(ds, events.GetBroker(), pluginLoader)

			// Bypass buffering for both built-in and plugin scrobblers
			tracker.(*playTracker).builtinScrobblers["fake"] = &fake
			tracker.(*playTracker).pluginScrobblers["plugin1"] = &pluginFake
		})

		It("registers and uses plugin scrobbler for NowPlaying", func() {
			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(pluginFake.NowPlayingCalled).To(BeTrue())
		})

		It("removes plugin scrobbler if not present anymore", func() {
			// First call: plugin present
			_ = tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)
			Expect(pluginFake.NowPlayingCalled).To(BeTrue())
			pluginFake.NowPlayingCalled = false
			// Remove plugin
			pluginLoader.names = []string{}
			_ = tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)
			Expect(pluginFake.NowPlayingCalled).To(BeFalse())
		})

		It("calls both builtin and plugin scrobblers for NowPlaying", func() {
			fake.NowPlayingCalled = false
			pluginFake.NowPlayingCalled = false
			err := tracker.NowPlaying(ctx, "player-1", "player-one", "123", 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(fake.NowPlayingCalled).To(BeTrue())
			Expect(pluginFake.NowPlayingCalled).To(BeTrue())
		})

		It("calls plugin scrobbler for Submit", func() {
			ts := time.Now()
			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: ts}})
			Expect(err).ToNot(HaveOccurred())
			Expect(pluginFake.ScrobbleCalled.Load()).To(BeTrue())
		})
	})

	Describe("Plugin Scrobbler Management", func() {
		var pluginScr *fakeScrobbler
		var mockPlugin *mockPluginLoader
		var pTracker *playTracker
		var mockedBS *mockBufferedScrobbler

		BeforeEach(func() {
			ctx = context.Background()
			ctx = request.WithUser(ctx, model.User{ID: "u-1"})
			ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: true})
			ds = &tests.MockDataStore{}

			// Setup plugin scrobbler
			pluginScr = &fakeScrobbler{Authorized: true}
			mockPlugin = &mockPluginLoader{
				names:      []string{"plugin1"},
				scrobblers: map[string]Scrobbler{"plugin1": pluginScr},
			}

			// Create a tracker with the mock plugin loader
			pTracker = newPlayTracker(ds, events.GetBroker(), mockPlugin)

			// Create a mock buffered scrobbler and explicitly cast it to Scrobbler
			mockedBS = &mockBufferedScrobbler{
				wrapped: pluginScr,
			}
			// Make sure the instance is added with its concrete type preserved
			pTracker.pluginScrobblers["plugin1"] = mockedBS
		})

		It("calls Stop on scrobblers when removing them", func() {
			// Change the plugin names to simulate a plugin being removed
			mockPlugin.names = []string{}

			// Call refreshPluginScrobblers which should detect the removed plugin
			pTracker.refreshPluginScrobblers()

			// Verify the Stop method was called
			Expect(mockedBS.stopCalled).To(BeTrue())

			// Verify the scrobbler was removed from the map
			Expect(pTracker.pluginScrobblers).NotTo(HaveKey("plugin1"))
		})
	})
})

type fakeScrobbler struct {
	Authorized       bool
	NowPlayingCalled bool
	ScrobbleCalled   atomic.Bool
	UserID           string
	Track            *model.MediaFile
	Position         int
	LastScrobble     atomic.Pointer[Scrobble]
	Error            error
}

func (f *fakeScrobbler) IsAuthorized(ctx context.Context, userId string) bool {
	return f.Error == nil && f.Authorized
}

func (f *fakeScrobbler) NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error {
	f.NowPlayingCalled = true
	if f.Error != nil {
		return f.Error
	}
	f.UserID = userId
	f.Track = track
	f.Position = position
	return nil
}

func (f *fakeScrobbler) Scrobble(ctx context.Context, userId string, s Scrobble) error {
	f.UserID = userId
	f.LastScrobble.Store(&s)
	f.ScrobbleCalled.Store(true)
	if f.Error != nil {
		return f.Error
	}
	return nil
}

func _p(id, name string, sortName ...string) model.Participant {
	p := model.Participant{Artist: model.Artist{ID: id, Name: name}}
	if len(sortName) > 0 {
		p.Artist.SortArtistName = sortName[0]
	}
	return p
}

type fakeEventBroker struct {
	http.Handler
	events []events.Event
	mu     sync.Mutex
}

func (f *fakeEventBroker) SendMessage(_ context.Context, event events.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakeEventBroker) SendBroadcastMessage(_ context.Context, event events.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func (f *fakeEventBroker) getEvents() []events.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.events
}

var _ events.Broker = (*fakeEventBroker)(nil)

// mockBufferedScrobbler used to test that Stop is called
type mockBufferedScrobbler struct {
	wrapped    Scrobbler
	stopCalled bool
}

func (m *mockBufferedScrobbler) Stop() {
	m.stopCalled = true
}

func (m *mockBufferedScrobbler) IsAuthorized(ctx context.Context, userId string) bool {
	return m.wrapped.IsAuthorized(ctx, userId)
}

func (m *mockBufferedScrobbler) NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error {
	return m.wrapped.NowPlaying(ctx, userId, track, position)
}

func (m *mockBufferedScrobbler) Scrobble(ctx context.Context, userId string, s Scrobble) error {
	return m.wrapped.Scrobble(ctx, userId, s)
}
