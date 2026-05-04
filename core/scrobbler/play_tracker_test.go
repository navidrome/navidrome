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

type mockPluginLoader struct {
	mu         sync.RWMutex
	names      []string
	scrobblers map[string]Scrobbler
}

func (m *mockPluginLoader) PluginNames(service string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.names
}

func (m *mockPluginLoader) SetNames(names []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.names = names
}

func (m *mockPluginLoader) LoadScrobbler(name string) (Scrobbler, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.scrobblers[name]
	return s, ok
}

var _ = Describe("PlayTracker", func() {
	var ctx context.Context
	var ds model.DataStore
	var tracker *playTracker
	var eventBroker *fakeEventBroker
	var track model.MediaFile
	var album model.Album
	var artist1 model.Artist
	var artist2 model.Artist
	var fake *fakeScrobbler

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = GinkgoT().Context()
		ctx = request.WithUser(ctx, model.User{ID: "u-1"})
		ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: true})
		ds = &tests.MockDataStore{}
		fake = &fakeScrobbler{Authorized: true}
		Register("fake", func(model.DataStore) Scrobbler {
			return fake
		})
		Register("disabled", func(model.DataStore) Scrobbler {
			return nil
		})
		eventBroker = &fakeEventBroker{}
		tracker = newPlayTracker(ds, eventBroker, nil)
		tracker.builtinScrobblers["fake"] = fake // Bypass buffering for tests

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

	AfterEach(func() {
		// Stop the worker goroutine to prevent data races between tests
		tracker.stopBackgroundWorkers()
	})

	It("does not register disabled scrobblers", func() {
		Expect(tracker.builtinScrobblers).To(HaveKey("fake"))
		Expect(tracker.builtinScrobblers).ToNot(HaveKey("disabled"))
	})

	Describe("GetNowPlaying", func() {
		It("returns current playing music", func() {
			track2 := track
			track2.ID = "456"
			_ = ds.MediaFile(ctx).Put(&track2)
			ctx1 := request.WithUser(GinkgoT().Context(), model.User{UserName: "user-1"})
			ctx1 = request.WithPlayer(ctx1, model.Player{ScrobbleEnabled: true})
			_ = tracker.ReportPlayback(ctx1, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: StatePlaying, PlaybackRate: 1.0, ClientId: "player-1", ClientName: "player-one",
			})
			ctx2 := request.WithUser(GinkgoT().Context(), model.User{UserName: "user-2"})
			ctx2 = request.WithPlayer(ctx2, model.Player{ScrobbleEnabled: true})
			_ = tracker.ReportPlayback(ctx2, ReportPlaybackParams{
				MediaId: "456", PositionMs: 0, State: StatePlaying, PlaybackRate: 1.0, ClientId: "player-2", ClientName: "player-two",
			})

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
			info := PlaybackSession{MediaFile: track, Start: time.Now(), Username: "user"}
			_ = tracker.playMap.AddWithTTL("player-1", info, 10*time.Millisecond)
			Eventually(func() int { return len(eventBroker.getEvents()) }).Should(BeNumerically(">", 0))
			eventList := eventBroker.getEvents()
			evt, ok := eventList[len(eventList)-1].(*events.NowPlayingCount)
			Expect(ok).To(BeTrue())
			Expect(evt.Count).To(Equal(0))
		})

		It("does not send event when disabled", func() {
			conf.Server.EnableNowPlaying = false
			tracker = newPlayTracker(ds, eventBroker, nil)
			info := PlaybackSession{MediaFile: track, Start: time.Now(), Username: "user"}
			_ = tracker.playMap.AddWithTTL("player-2", info, 10*time.Millisecond)
			Consistently(func() int { return len(eventBroker.getEvents()) }).Should(Equal(0))
		})

		It("sends expired playback report when session expires", func() {
			info := PlaybackSession{
				MediaFile:  track,
				Start:      time.Now(),
				UserId:     "u-1",
				Username:   "user",
				PlayerId:   "player-3",
				PlayerName: "test-player",
				State:      StatePlaying,
				PositionMs: 5000,
			}
			_ = tracker.playMap.AddWithTTL("player-3", info, 10*time.Millisecond)
			Eventually(func() *PlaybackSession {
				return fake.LastPlaybackReport.Load()
			}).ShouldNot(BeNil())
			report := fake.LastPlaybackReport.Load()
			Expect(report.State).To(Equal(StateExpired))
			Expect(report.MediaFile.ID).To(Equal("123"))
			Expect(report.PlayerId).To(Equal("player-3"))
		})

		It("does not send expired report when session was already stopped", func() {
			info := PlaybackSession{
				MediaFile:  track,
				Start:      time.Now(),
				UserId:     "u-1",
				Username:   "user",
				PlayerId:   "player-4",
				PlayerName: "test-player",
				State:      StateStopped,
				PositionMs: 180000,
			}
			_ = tracker.playMap.AddWithTTL("player-4", info, 10*time.Millisecond)
			Consistently(func() *PlaybackSession {
				return fake.LastPlaybackReport.Load()
			}).Should(BeNil())
		})
	})

	Describe("Submit", func() {
		It("sends track to agent", func() {
			ctx = request.WithUser(ctx, model.User{ID: "u-1", UserName: "user-1"})
			ts := time.Now()

			err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: ts}})

			Expect(err).ToNot(HaveOccurred())
			Expect(fake.ScrobbleCalled.Load()).To(BeTrue())
			Expect(fake.GetUserID()).To(Equal("u-1"))
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

		Context("Scrobble History", func() {
			It("records scrobble in repository", func() {
				conf.Server.EnableScrobbleHistory = true
				ctx = request.WithUser(ctx, model.User{ID: "u-1", UserName: "user-1"})
				ts := time.Now()

				err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: ts}})

				Expect(err).ToNot(HaveOccurred())

				mockDS := ds.(*tests.MockDataStore)
				mockScrobble := mockDS.Scrobble(ctx).(*tests.MockScrobbleRepo)
				Expect(mockScrobble.RecordedScrobbles).To(HaveLen(1))
				Expect(mockScrobble.RecordedScrobbles[0].MediaFileID).To(Equal("123"))
				Expect(mockScrobble.RecordedScrobbles[0].UserID).To(Equal("u-1"))
				Expect(mockScrobble.RecordedScrobbles[0].SubmissionTime).To(Equal(ts))
			})

			It("does not record scrobble when history is disabled", func() {
				conf.Server.EnableScrobbleHistory = false
				ctx = request.WithUser(ctx, model.User{ID: "u-1", UserName: "user-1"})
				ts := time.Now()

				err := tracker.Submit(ctx, []Submission{{TrackID: "123", Timestamp: ts}})

				Expect(err).ToNot(HaveOccurred())
				mockDS := ds.(*tests.MockDataStore)
				mockScrobble := mockDS.Scrobble(ctx).(*tests.MockScrobbleRepo)
				Expect(mockScrobble.RecordedScrobbles).To(HaveLen(0))
			})
		})
	})

	Describe("ReportPlayback", func() {
		const defaultClientId = "client-1"

		BeforeEach(func() {
			ctx = request.WithPlayer(ctx, model.Player{ID: "p1", ScrobbleEnabled: true})
		})

		It("creates entry on starting and removes on stopped", func() {
			err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
			})
			Expect(err).ToNot(HaveOccurred())

			playing, err := tracker.GetNowPlaying(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(playing).To(HaveLen(1))
			Expect(playing[0].State).To(Equal("starting"))
			Expect(playing[0].MediaFile.ID).To(Equal("123"))

			err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				IgnoreScrobble: true,
			})
			Expect(err).ToNot(HaveOccurred())

			playing, err = tracker.GetNowPlaying(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(playing).To(BeEmpty())
		})

		It("full lifecycle: starting -> playing -> paused -> playing -> stopped", func() {
			err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
			})
			Expect(err).ToNot(HaveOccurred())
			err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 10000, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
			})
			Expect(err).ToNot(HaveOccurred())
			playing, err := tracker.GetNowPlaying(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(playing).To(HaveLen(1))
			Expect(playing[0].State).To(Equal("playing"))
			Expect(playing[0].PositionMs).To(BeNumerically(">=", int64(10000)))

			err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 30000, State: "paused", PlaybackRate: 1.0, ClientId: defaultClientId,
			})
			Expect(err).ToNot(HaveOccurred())
			playing, err = tracker.GetNowPlaying(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(playing[0].State).To(Equal("paused"))
			Expect(playing[0].PositionMs).To(Equal(int64(30000)))

			err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 30000, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
			})
			Expect(err).ToNot(HaveOccurred())

			err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 100000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
			})
			Expect(err).ToNot(HaveOccurred())
			playing, err = tracker.GetNowPlaying(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(playing).To(BeEmpty())
		})

		It("starting replaces existing entry for same player", func() {
			err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 50000, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
			})
			Expect(err).ToNot(HaveOccurred())
			err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
			})
			Expect(err).ToNot(HaveOccurred())
			playing, err := tracker.GetNowPlaying(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(playing).To(HaveLen(1))
			Expect(playing[0].State).To(Equal("starting"))
			Expect(playing[0].PositionMs).To(Equal(int64(0)))
		})

		It("multiple players have independent sessions", func() {
			ctx1 := request.WithUser(ctx, model.User{ID: "u-1", UserName: "user1"})
			ctx1 = request.WithPlayer(ctx1, model.Player{ID: "p1", ScrobbleEnabled: true})

			ctx2 := request.WithUser(ctx, model.User{ID: "u-1", UserName: "user1"})
			ctx2 = request.WithPlayer(ctx2, model.Player{ID: "p2", ScrobbleEnabled: true})

			track2 := track
			track2.ID = "456"
			_ = ds.MediaFile(ctx).Put(&track2)

			err := tracker.ReportPlayback(ctx1, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: "playing", PlaybackRate: 1.0, ClientId: "client-1",
			})
			Expect(err).ToNot(HaveOccurred())
			err = tracker.ReportPlayback(ctx2, ReportPlaybackParams{
				MediaId: "456", PositionMs: 0, State: "playing", PlaybackRate: 1.0, ClientId: "client-2",
			})
			Expect(err).ToNot(HaveOccurred())
			playing, err := tracker.GetNowPlaying(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(playing).To(HaveLen(2))
		})

		Describe("SSE broadcast on state change", func() {
			BeforeEach(func() {
				eventBroker = &fakeEventBroker{}
				tracker = newPlayTracker(ds, eventBroker, nil)
				tracker.builtinScrobblers["fake"] = fake
			})

			It("broadcasts NowPlayingCount on every state change", func() {
				// starting -> count should be 1
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				evts := eventBroker.getEvents()
				Expect(evts).To(HaveLen(1))
				Expect(evts[0].(*events.NowPlayingCount).Count).To(Equal(1))

				// playing -> count should be 1
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				evts = eventBroker.getEvents()
				Expect(evts).To(HaveLen(2))
				Expect(evts[1].(*events.NowPlayingCount).Count).To(Equal(1))

				// paused -> count should be 1
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 30000, State: "paused", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				evts = eventBroker.getEvents()
				Expect(evts).To(HaveLen(3))
				Expect(evts[2].(*events.NowPlayingCount).Count).To(Equal(1))

				// stopped -> count should be 0
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 30000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
					IgnoreScrobble: true,
				})
				Expect(err).ToNot(HaveOccurred())
				evts = eventBroker.getEvents()
				Expect(evts).To(HaveLen(4))
				Expect(evts[3].(*events.NowPlayingCount).Count).To(Equal(0))
			})

			It("does NOT broadcast when EnableNowPlaying is false", func() {
				conf.Server.EnableNowPlaying = false
				tracker = newPlayTracker(ds, eventBroker, nil)
				tracker.builtinScrobblers["fake"] = fake

				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(eventBroker.getEvents()).To(BeEmpty())
			})
		})

		Describe("auto-scrobble", func() {
			It("scrobbles on stopped when positionMs >= 50% of track", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 90000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(track.PlayCount).To(Equal(int64(1)))
				Expect(album.PlayCount).To(Equal(int64(1)))
				Expect(artist1.PlayCount).To(Equal(int64(1)))
			})

			It("scrobbles on stopped when positionMs >= 4 min for long tracks", func() {
				longTrack := model.MediaFile{
					ID: "long", Title: "Long Song", Album: "Album", AlbumID: "al-1",
					Duration: 600,
					Participants: map[model.Role]model.ParticipantList{
						model.RoleArtist: []model.Participant{_p("ar-1", "Artist 1")},
					},
				}
				_ = ds.MediaFile(ctx).Put(&longTrack)

				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "long", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "long", PositionMs: 240000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(longTrack.PlayCount).To(Equal(int64(1)))
			})

			It("does NOT scrobble when positionMs below threshold", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(track.PlayCount).To(Equal(int64(0)))
			})

			It("does NOT scrobble when ignoreScrobble=true even if threshold met", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 90000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
					IgnoreScrobble: true,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(track.PlayCount).To(Equal(int64(0)))
			})

			It("does NOT scrobble when player ScrobbleEnabled=false even if threshold met", func() {
				ctx = request.WithPlayer(ctx, model.Player{ID: "p1", ScrobbleEnabled: false})

				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 90000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(track.PlayCount).To(Equal(int64(0)))
			})

			It("scrobbles twice for two separate sessions of same song", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 90000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 90000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(track.PlayCount).To(Equal(int64(2)))
			})

			It("dispatches to external scrobblers on auto-scrobble", func() {
				fake.ScrobbleCalled.Store(false)
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 90000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(fake.ScrobbleCalled.Load()).To(BeTrue())
			})
		})

		Describe("position estimation", func() {
			It("estimates position for playing state based on elapsed time", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(50 * time.Millisecond)
				playing, err := tracker.GetNowPlaying(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(playing).To(HaveLen(1))
				Expect(playing[0].PositionMs).To(BeNumerically(">", int64(10000)))
			})

			It("does NOT estimate for paused", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "paused", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(50 * time.Millisecond)
				playing, err := tracker.GetNowPlaying(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(playing).To(HaveLen(1))
				Expect(playing[0].PositionMs).To(Equal(int64(10000)))
			})

			It("does NOT estimate for starting", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(50 * time.Millisecond)
				playing, err := tracker.GetNowPlaying(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(playing).To(HaveLen(1))
				Expect(playing[0].PositionMs).To(Equal(int64(0)))
			})

			It("respects playbackRate", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "playing", PlaybackRate: 2.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(100 * time.Millisecond)
				playing, err := tracker.GetNowPlaying(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(playing).To(HaveLen(1))
				// At 2x speed, 100ms real time = ~200ms playback time
				Expect(playing[0].PositionMs).To(BeNumerically(">", int64(10100)))
			})

			It("caps estimated position at track duration", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 179990, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				time.Sleep(50 * time.Millisecond)
				playing, err := tracker.GetNowPlaying(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(playing).To(HaveLen(1))
				Expect(playing[0].PositionMs).To(Equal(int64(180000))) // track.Duration * 1000
			})

		})

		Describe("resilience (no prior starting)", func() {
			It("playing without prior starting creates entry with Start approx now - positionMs", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 30000, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				playing, err := tracker.GetNowPlaying(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(playing).To(HaveLen(1))
				Expect(playing[0].State).To(Equal("playing"))
				expectedStart := time.Now().Add(-30 * time.Second)
				Expect(playing[0].Start).To(BeTemporally("~", expectedStart, 2*time.Second))
			})

			It("paused without prior starting creates entry", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 30000, State: "paused", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				playing, err := tracker.GetNowPlaying(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(playing).To(HaveLen(1))
				Expect(playing[0].State).To(Equal("paused"))
			})

			It("stopped without prior starting auto-scrobbles if threshold met", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 90000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(track.PlayCount).To(Equal(int64(1)))
			})

			It("stopped without prior starting does NOT scrobble if below threshold", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "stopped", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Expect(track.PlayCount).To(Equal(int64(0)))
			})
		})

		Describe("external scrobbler dispatch", func() {
			It("dispatches NowPlaying on starting", func() {
				fake.nowPlayingCalled.Store(false)
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool { return fake.GetNowPlayingCalled() }).Should(BeTrue())
			})

			It("dispatches NowPlaying on playing", func() {
				fake.nowPlayingCalled.Store(false)
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "playing", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool { return fake.GetNowPlayingCalled() }).Should(BeTrue())
			})

			It("does NOT dispatch on paused", func() {
				fake.nowPlayingCalled.Store(false)
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 10000, State: "paused", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Consistently(func() bool { return fake.GetNowPlayingCalled() }).Should(BeFalse())
			})

			It("does NOT dispatch when ignoreScrobble=true", func() {
				fake.nowPlayingCalled.Store(false)
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
					IgnoreScrobble: true,
				})
				Expect(err).ToNot(HaveOccurred())
				Consistently(func() bool { return fake.GetNowPlayingCalled() }).Should(BeFalse())
			})

			It("does NOT dispatch when ScrobbleEnabled=false", func() {
				fake.nowPlayingCalled.Store(false)
				ctx = request.WithPlayer(ctx, model.Player{ID: "p1", ScrobbleEnabled: false})
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: "starting", PlaybackRate: 1.0, ClientId: defaultClientId,
				})
				Expect(err).ToNot(HaveOccurred())
				Consistently(func() bool { return fake.GetNowPlayingCalled() }).Should(BeFalse())
			})
		})

		Describe("PlaybackReport dispatch", func() {
			It("dispatches PlaybackReport for starting state", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: StateStarting, PlaybackRate: 1.0,
					ClientId: "client-1", ClientName: "Test Player",
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool {
					return fake.PlaybackReportCalled.Load()
				}).Should(BeTrue())

				info := fake.LastPlaybackReport.Load()
				Expect(info).ToNot(BeNil())
				Expect(info.MediaFile.ID).To(Equal("123"))
				Expect(info.State).To(Equal(StateStarting))
				Expect(info.PositionMs).To(Equal(int64(0)))
				Expect(info.PlaybackRate).To(Equal(1.0))
				Expect(info.PlayerId).To(Equal("client-1"))
				Expect(info.PlayerName).To(Equal("Test Player"))
			})

			It("dispatches PlaybackReport for playing state", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: StateStarting, PlaybackRate: 1.0,
					ClientId: "client-1", ClientName: "Test Player",
				})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool { return fake.PlaybackReportCalled.Load() }).Should(BeTrue())
				fake.PlaybackReportCalled.Store(false)
				fake.LastPlaybackReport.Store(nil)

				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 30000, State: StatePlaying, PlaybackRate: 1.5,
					ClientId: "client-1", ClientName: "Test Player",
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool { return fake.PlaybackReportCalled.Load() }).Should(BeTrue())
				info := fake.LastPlaybackReport.Load()
				Expect(info.State).To(Equal(StatePlaying))
				Expect(info.PositionMs).To(Equal(int64(30000)))
				Expect(info.PlaybackRate).To(Equal(1.5))
			})

			It("dispatches PlaybackReport for paused state", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: StateStarting, PlaybackRate: 1.0,
					ClientId: "client-1", ClientName: "Test Player",
				})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool { return fake.PlaybackReportCalled.Load() }).Should(BeTrue())
				fake.PlaybackReportCalled.Store(false)
				fake.LastPlaybackReport.Store(nil)

				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 45000, State: StatePaused, PlaybackRate: 1.0,
					ClientId: "client-1", ClientName: "Test Player",
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool { return fake.PlaybackReportCalled.Load() }).Should(BeTrue())
				info := fake.LastPlaybackReport.Load()
				Expect(info.State).To(Equal(StatePaused))
				Expect(info.PositionMs).To(Equal(int64(45000)))
			})

			It("dispatches PlaybackReport for stopped state", func() {
				err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 0, State: StateStarting, PlaybackRate: 1.0,
					ClientId: "client-1", ClientName: "Test Player",
				})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() bool { return fake.PlaybackReportCalled.Load() }).Should(BeTrue())
				fake.PlaybackReportCalled.Store(false)
				fake.LastPlaybackReport.Store(nil)

				err = tracker.ReportPlayback(ctx, ReportPlaybackParams{
					MediaId: "123", PositionMs: 100000, State: StateStopped, PlaybackRate: 1.0,
					ClientId: "client-1", ClientName: "Test Player",
				})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() bool { return fake.PlaybackReportCalled.Load() }).Should(BeTrue())
				info := fake.LastPlaybackReport.Load()
				Expect(info.State).To(Equal(StateStopped))
				Expect(info.PositionMs).To(Equal(int64(100000)))
			})
		})
	})

	Describe("Plugin scrobbler logic", func() {
		var pluginLoader *mockPluginLoader
		var pluginFake *fakeScrobbler

		BeforeEach(func() {
			pluginFake = &fakeScrobbler{Authorized: true}
			pluginLoader = &mockPluginLoader{
				names:      []string{"plugin1"},
				scrobblers: map[string]Scrobbler{"plugin1": pluginFake},
			}
			tracker = newPlayTracker(ds, events.GetBroker(), pluginLoader)

			// Bypass buffering for both built-in and plugin scrobblers
			tracker.builtinScrobblers["fake"] = fake
			tracker.pluginScrobblers["plugin1"] = pluginFake
		})

		It("registers and uses plugin scrobbler for NowPlaying", func() {
			err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: StatePlaying, PlaybackRate: 1.0, ClientId: "player-1",
			})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool { return pluginFake.GetNowPlayingCalled() }).Should(BeTrue())
		})

		It("removes plugin scrobbler if not present anymore", func() {
			_ = tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: StatePlaying, PlaybackRate: 1.0, ClientId: "player-1",
			})
			Eventually(func() bool { return pluginFake.GetNowPlayingCalled() }).Should(BeTrue())
			pluginFake.nowPlayingCalled.Store(false)
			pluginLoader.SetNames([]string{})
			_ = tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: StatePlaying, PlaybackRate: 1.0, ClientId: "player-1",
			})
			Consistently(func() bool { return pluginFake.GetNowPlayingCalled() }).Should(BeFalse())
		})

		It("calls both builtin and plugin scrobblers for NowPlaying", func() {
			fake.nowPlayingCalled.Store(false)
			pluginFake.nowPlayingCalled.Store(false)
			err := tracker.ReportPlayback(ctx, ReportPlaybackParams{
				MediaId: "123", PositionMs: 0, State: StatePlaying, PlaybackRate: 1.0, ClientId: "player-1",
			})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool { return fake.GetNowPlayingCalled() }).Should(BeTrue())
			Eventually(func() bool { return pluginFake.GetNowPlayingCalled() }).Should(BeTrue())
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
			ctx = GinkgoT().Context()
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
			mockPlugin.SetNames([]string{})

			// Call refreshPluginScrobblers which should detect the removed plugin
			pTracker.refreshPluginScrobblers()

			// Verify the Stop method was called
			Expect(mockedBS.stopCalled).To(BeTrue())

			// Verify the scrobbler was removed from the map
			Expect(pTracker.pluginScrobblers).NotTo(HaveKey("plugin1"))
		})
	})

	Describe("Plugin reload (config update) behavior", func() {
		var mockPlugin *mockPluginLoader
		var pTracker *playTracker
		var originalScrobbler *fakeScrobbler
		var reloadedScrobbler *fakeScrobbler

		BeforeEach(func() {
			ctx = GinkgoT().Context()
			ctx = request.WithUser(ctx, model.User{ID: "u-1"})
			ctx = request.WithPlayer(ctx, model.Player{ScrobbleEnabled: true})
			ds = &tests.MockDataStore{}

			// Setup initial plugin scrobbler
			originalScrobbler = &fakeScrobbler{Authorized: true}
			reloadedScrobbler = &fakeScrobbler{Authorized: true}

			mockPlugin = &mockPluginLoader{
				names:      []string{"plugin1"},
				scrobblers: map[string]Scrobbler{"plugin1": originalScrobbler},
			}

			// Create tracker - this will create buffered scrobblers with loaders
			pTracker = newPlayTracker(ds, events.GetBroker(), mockPlugin)

			// Trigger initial plugin registration
			pTracker.refreshPluginScrobblers()
		})

		AfterEach(func() {
			pTracker.stopBackgroundWorkers()
		})

		It("uses the new plugin instance after reload (simulating config update)", func() {
			// First call should use the original scrobbler
			scrobblers := pTracker.getActiveScrobblers()
			pluginScr := scrobblers["plugin1"]
			Expect(pluginScr).ToNot(BeNil())

			err := pluginScr.NowPlaying(ctx, "u-1", &track, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(originalScrobbler.GetNowPlayingCalled()).To(BeTrue())
			Expect(reloadedScrobbler.GetNowPlayingCalled()).To(BeFalse())

			// Simulate plugin reload (config update): replace the scrobbler in the loader
			// This is what happens when UpdatePluginConfig is called - the plugin manager
			// unloads the old plugin and loads a new instance
			mockPlugin.mu.Lock()
			mockPlugin.scrobblers["plugin1"] = reloadedScrobbler
			mockPlugin.mu.Unlock()

			// Reset call tracking
			originalScrobbler.nowPlayingCalled.Store(false)

			// Get scrobblers again - should still return the same buffered scrobbler
			// but subsequent calls should use the new plugin instance via the loader
			scrobblers = pTracker.getActiveScrobblers()
			pluginScr = scrobblers["plugin1"]

			err = pluginScr.NowPlaying(ctx, "u-1", &track, 0)
			Expect(err).ToNot(HaveOccurred())

			// The new scrobbler should be called, not the old one
			Expect(reloadedScrobbler.GetNowPlayingCalled()).To(BeTrue())
			Expect(originalScrobbler.GetNowPlayingCalled()).To(BeFalse())
		})

		It("handles plugin becoming unavailable temporarily", func() {
			// First verify plugin works
			scrobblers := pTracker.getActiveScrobblers()
			pluginScr := scrobblers["plugin1"]

			err := pluginScr.NowPlaying(ctx, "u-1", &track, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(originalScrobbler.GetNowPlayingCalled()).To(BeTrue())

			// Simulate plugin becoming unavailable (e.g., during reload)
			mockPlugin.mu.Lock()
			delete(mockPlugin.scrobblers, "plugin1")
			mockPlugin.mu.Unlock()

			originalScrobbler.nowPlayingCalled.Store(false)

			// NowPlaying should return error when plugin unavailable
			err = pluginScr.NowPlaying(ctx, "u-1", &track, 0)
			Expect(err).To(HaveOccurred())
			Expect(originalScrobbler.GetNowPlayingCalled()).To(BeFalse())

			// Simulate plugin becoming available again
			mockPlugin.mu.Lock()
			mockPlugin.scrobblers["plugin1"] = reloadedScrobbler
			mockPlugin.mu.Unlock()

			// Should work again with new instance
			err = pluginScr.NowPlaying(ctx, "u-1", &track, 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(reloadedScrobbler.GetNowPlayingCalled()).To(BeTrue())
		})

		It("IsAuthorized uses the current plugin instance", func() {
			scrobblers := pTracker.getActiveScrobblers()
			pluginScr := scrobblers["plugin1"]

			// Original is authorized
			Expect(pluginScr.IsAuthorized(ctx, "u-1")).To(BeTrue())

			// Replace with unauthorized scrobbler
			unauthorizedScrobbler := &fakeScrobbler{Authorized: false}
			mockPlugin.mu.Lock()
			mockPlugin.scrobblers["plugin1"] = unauthorizedScrobbler
			mockPlugin.mu.Unlock()

			// Should reflect the new scrobbler's authorization status
			Expect(pluginScr.IsAuthorized(ctx, "u-1")).To(BeFalse())
		})
	})
})

var _ = DescribeTable("remainingTTL",
	func(durationSec float32, positionMs int64, rate float64, expected time.Duration) {
		Expect(remainingTTL(durationSec, positionMs, rate)).To(Equal(expected))
	},
	Entry("full track at 1x", float32(300), int64(0), 1.0, 305*time.Second),
	Entry("halfway through at 1x", float32(300), int64(150000), 1.0, 155*time.Second),
	Entry("near end at 1x", float32(300), int64(298000), 1.0, 7*time.Second),
	Entry("at end of track", float32(300), int64(300000), 1.0, 5*time.Second),
	Entry("past end of track", float32(300), int64(310000), 1.0, 5*time.Second),
	Entry("2x speed halves remaining time", float32(300), int64(0), 2.0, 155*time.Second),
	Entry("2x speed halfway", float32(300), int64(150000), 2.0, 80*time.Second),
	Entry("0.5x speed doubles remaining time", float32(300), int64(0), 0.5, 605*time.Second),
	Entry("zero rate defaults to 1x", float32(300), int64(0), 0.0, 305*time.Second),
	Entry("negative rate defaults to 1x", float32(300), int64(0), -1.0, 305*time.Second),
	Entry("short track", float32(3.5), int64(0), 1.0, 8*time.Second),
	Entry("zero duration", float32(0), int64(0), 1.0, 5*time.Second),
)

type fakeScrobbler struct {
	Authorized           bool
	nowPlayingCalled     atomic.Bool
	ScrobbleCalled       atomic.Bool
	PlaybackReportCalled atomic.Bool
	userID               atomic.Pointer[string]
	username             atomic.Pointer[string]
	track                atomic.Pointer[model.MediaFile]
	position             atomic.Int32
	LastScrobble         atomic.Pointer[Scrobble]
	LastPlaybackReport   atomic.Pointer[PlaybackSession]
	Error                error
}

func (f *fakeScrobbler) GetNowPlayingCalled() bool {
	return f.nowPlayingCalled.Load()
}

func (f *fakeScrobbler) GetUserID() string {
	if p := f.userID.Load(); p != nil {
		return *p
	}
	return ""
}

func (f *fakeScrobbler) GetTrack() *model.MediaFile {
	return f.track.Load()
}

func (f *fakeScrobbler) IsAuthorized(ctx context.Context, userId string) bool {
	return f.Error == nil && f.Authorized
}

func (f *fakeScrobbler) NowPlaying(ctx context.Context, userId string, track *model.MediaFile, position int) error {
	f.nowPlayingCalled.Store(true)
	if f.Error != nil {
		return f.Error
	}
	f.userID.Store(&userId)
	// Capture username from context (this is what plugin scrobblers do)
	username, _ := request.UsernameFrom(ctx)
	if username == "" {
		if u, ok := request.UserFrom(ctx); ok {
			username = u.UserName
		}
	}
	if username != "" {
		f.username.Store(&username)
	}
	f.track.Store(track)
	f.position.Store(int32(position))
	return nil
}

func (f *fakeScrobbler) Scrobble(ctx context.Context, userId string, s Scrobble) error {
	f.userID.Store(&userId)
	f.LastScrobble.Store(&s)
	f.ScrobbleCalled.Store(true)
	if f.Error != nil {
		return f.Error
	}
	return nil
}

func (f *fakeScrobbler) PlaybackReport(ctx context.Context, info PlaybackSession) error {
	f.PlaybackReportCalled.Store(true)
	if f.Error != nil {
		return f.Error
	}
	uid := info.UserId
	f.userID.Store(&uid)
	f.LastPlaybackReport.Store(&info)
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

func (m *mockBufferedScrobbler) PlaybackReport(ctx context.Context, info PlaybackSession) error {
	return m.wrapped.PlaybackReport(ctx, info)
}
