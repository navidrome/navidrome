//go:build !windows

package plugins

import (
	"context"
	"errors"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// ctxWithUser returns a fresh context with the test user.
// Must be called within each test, not in BeforeAll, because the context
// from BeforeAll gets cancelled before tests run.
func ctxWithUser() context.Context {
	return request.WithUser(GinkgoT().Context(), model.User{ID: "user-1", UserName: "testuser"})
}

var _ = Describe("ScrobblerPlugin", Ordered, func() {
	var (
		scrobblerManager *Manager
		s                scrobbler.Scrobbler
	)

	BeforeAll(func() {
		// Load the scrobbler via a new manager with the test-scrobbler plugin
		scrobblerManager, _ = createTestManagerWithPlugins(nil, "test-scrobbler"+PackageExtension)

		var ok bool
		s, ok = scrobblerManager.LoadScrobbler("test-scrobbler")
		Expect(ok).To(BeTrue())
	})

	Describe("LoadScrobbler", func() {
		It("returns a scrobbler for a plugin with Scrobbler capability", func() {
			Expect(s).ToNot(BeNil())
		})

		It("returns false for a plugin without Scrobbler capability", func() {
			_, ok := testManager.LoadScrobbler("test-metadata-agent")
			Expect(ok).To(BeFalse())
		})

		It("returns false for non-existent plugin", func() {
			_, ok := scrobblerManager.LoadScrobbler("non-existent")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("IsAuthorized", func() {
		It("returns true when plugin is configured to authorize", func() {
			result := s.IsAuthorized(ctxWithUser(), "user-1")
			Expect(result).To(BeTrue())
		})

		It("returns false when plugin is configured to not authorize", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-scrobbler": {"authorized": "false"},
			}, "test-scrobbler"+PackageExtension)

			sc, ok := manager.LoadScrobbler("test-scrobbler")
			Expect(ok).To(BeTrue())

			result := sc.IsAuthorized(ctxWithUser(), "user-1")
			Expect(result).To(BeFalse())
		})
	})

	Describe("isUserAllowed", func() {
		It("returns true when allUsers is true", func() {
			sp := &ScrobblerPlugin{allUsers: true}
			Expect(sp.isUserAllowed("any-user")).To(BeTrue())
		})

		It("returns false when allowedUserIDs is empty and allUsers is false", func() {
			sp := &ScrobblerPlugin{allUsers: false, allowedUserIDs: []string{}}
			Expect(sp.isUserAllowed("user-1")).To(BeFalse())
		})

		It("returns false when allowedUserIDs is nil and allUsers is false", func() {
			sp := &ScrobblerPlugin{allUsers: false}
			Expect(sp.isUserAllowed("user-1")).To(BeFalse())
		})

		It("returns true when user is in allowedUserIDs", func() {
			sp := &ScrobblerPlugin{
				allUsers:       false,
				allowedUserIDs: []string{"user-1", "user-2"},
				userIDMap:      map[string]struct{}{"user-1": {}, "user-2": {}},
			}
			Expect(sp.isUserAllowed("user-1")).To(BeTrue())
		})

		It("returns false when user is not in allowedUserIDs", func() {
			sp := &ScrobblerPlugin{
				allUsers:       false,
				allowedUserIDs: []string{"user-1", "user-2"},
				userIDMap:      map[string]struct{}{"user-1": {}, "user-2": {}},
			}
			Expect(sp.isUserAllowed("user-3")).To(BeFalse())
		})
	})

	Describe("NowPlaying", func() {
		It("successfully calls the plugin", func() {
			track := &model.MediaFile{
				ID:          "track-1",
				Title:       "Test Song",
				Album:       "Test Album",
				Artist:      "Test Artist",
				AlbumArtist: "Test Album Artist",
				Duration:    180,
				TrackNumber: 1,
				DiscNumber:  1,
				Participants: model.Participants{
					model.RoleArtist:      {{Artist: model.Artist{ID: "artist-1", Name: "Test Artist"}}},
					model.RoleAlbumArtist: {{Artist: model.Artist{ID: "album-artist-1", Name: "Test Album Artist"}}},
				},
			}

			err := s.NowPlaying(ctxWithUser(), "user-1", track, 30)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error when plugin returns error", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-scrobbler": {"error": "service unavailable", "error_type": "scrobbler(retry_later)"},
			}, "test-scrobbler"+PackageExtension)

			sc, ok := manager.LoadScrobbler("test-scrobbler")
			Expect(ok).To(BeTrue())

			track := &model.MediaFile{ID: "track-1", Title: "Test Song"}
			err := sc.NowPlaying(ctxWithUser(), "user-1", track, 30)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(scrobbler.ErrRetryLater))
		})
	})

	Describe("Scrobble", func() {
		It("successfully calls the plugin", func() {
			sc := scrobbler.Scrobble{
				MediaFile: model.MediaFile{
					ID:          "track-1",
					Title:       "Test Song",
					Album:       "Test Album",
					Artist:      "Test Artist",
					AlbumArtist: "Test Album Artist",
					Duration:    180,
					TrackNumber: 1,
					DiscNumber:  1,
					Participants: model.Participants{
						model.RoleArtist:      {{Artist: model.Artist{ID: "artist-1", Name: "Test Artist"}}},
						model.RoleAlbumArtist: {{Artist: model.Artist{ID: "album-artist-1", Name: "Test Album Artist"}}},
					},
				},
				TimeStamp: time.Now(),
			}

			err := s.Scrobble(ctxWithUser(), "user-1", sc)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error when plugin returns not_authorized error", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-scrobbler": {"error": "user not linked", "error_type": "scrobbler(not_authorized)"},
			}, "test-scrobbler"+PackageExtension)

			sc, ok := manager.LoadScrobbler("test-scrobbler")
			Expect(ok).To(BeTrue())

			scrobble := scrobbler.Scrobble{
				MediaFile: model.MediaFile{ID: "track-1", Title: "Test Song"},
				TimeStamp: time.Now(),
			}
			err := sc.Scrobble(ctxWithUser(), "user-1", scrobble)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
		})

		It("returns error when plugin returns unrecoverable error", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-scrobbler": {"error": "track rejected", "error_type": "scrobbler(unrecoverable)"},
			}, "test-scrobbler"+PackageExtension)

			sc, ok := manager.LoadScrobbler("test-scrobbler")
			Expect(ok).To(BeTrue())

			scrobble := scrobbler.Scrobble{
				MediaFile: model.MediaFile{ID: "track-1", Title: "Test Song"},
				TimeStamp: time.Now(),
			}
			err := sc.Scrobble(ctxWithUser(), "user-1", scrobble)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(scrobbler.ErrUnrecoverable))
		})
	})

	Describe("PluginNames", func() {
		It("returns plugin names with Scrobbler capability", func() {
			names := scrobblerManager.PluginNames("Scrobbler")
			Expect(names).To(ContainElement("test-scrobbler"))
		})

		It("does not return metadata agent plugins for Scrobbler capability", func() {
			names := testManager.PluginNames("Scrobbler")
			Expect(names).ToNot(ContainElement("test-metadata-agent"))
		})
	})
})

var _ = Describe("mapScrobblerError", func() {
	It("returns nil for nil error", func() {
		Expect(mapScrobblerError(nil)).ToNot(HaveOccurred())
	})

	It("returns ErrNotAuthorized for error containing 'not_authorized'", func() {
		err := mapScrobblerError(errors.New("plugin error: scrobbler(not_authorized)"))
		Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
	})

	It("returns ErrRetryLater for error containing 'retry_later'", func() {
		err := mapScrobblerError(errors.New("temporary failure: scrobbler(retry_later)"))
		Expect(err).To(MatchError(scrobbler.ErrRetryLater))
	})

	It("returns ErrUnrecoverable for error containing 'unrecoverable'", func() {
		err := mapScrobblerError(errors.New("fatal error: scrobbler(unrecoverable)"))
		Expect(err).To(MatchError(scrobbler.ErrUnrecoverable))
	})

	It("returns ErrUnrecoverable for unknown error", func() {
		err := mapScrobblerError(errors.New("some unknown error"))
		Expect(err).To(MatchError(scrobbler.ErrUnrecoverable))
	})
})
