//go:build !windows

package plugins

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/core/scrobbler"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ScrobblerPlugin", Ordered, func() {
	var (
		scrobblerManager *Manager
		s                scrobbler.Scrobbler
		ctx              context.Context
	)

	BeforeAll(func() {
		ctx = GinkgoT().Context()
		// Add user to context for username extraction
		ctx = request.WithUser(ctx, model.User{ID: "user-1", UserName: "testuser"})

		// Load the scrobbler via a new manager with the test-scrobbler plugin
		scrobblerManager, _ = createTestManagerWithPlugins(nil, "test-scrobbler.wasm")

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
			result := s.IsAuthorized(ctx, "user-1")
			Expect(result).To(BeTrue())
		})

		It("returns false when plugin is configured to not authorize", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-scrobbler": {"authorized": "false"},
			}, "test-scrobbler.wasm")

			sc, ok := manager.LoadScrobbler("test-scrobbler")
			Expect(ok).To(BeTrue())

			result := sc.IsAuthorized(ctx, "user-1")
			Expect(result).To(BeFalse())
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
			}

			err := s.NowPlaying(ctx, "user-1", track, 30)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error when plugin returns error", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-scrobbler": {"error": "service unavailable", "error_type": "retry_later"},
			}, "test-scrobbler.wasm")

			sc, ok := manager.LoadScrobbler("test-scrobbler")
			Expect(ok).To(BeTrue())

			track := &model.MediaFile{ID: "track-1", Title: "Test Song"}
			err := sc.NowPlaying(ctx, "user-1", track, 30)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("retry later")))
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
				},
				TimeStamp: time.Now(),
			}

			err := s.Scrobble(ctx, "user-1", sc)
			Expect(err).ToNot(HaveOccurred())
		})

		It("returns error when plugin returns not_authorized error", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-scrobbler": {"error": "user not linked", "error_type": "not_authorized"},
			}, "test-scrobbler.wasm")

			sc, ok := manager.LoadScrobbler("test-scrobbler")
			Expect(ok).To(BeTrue())

			scrobble := scrobbler.Scrobble{
				MediaFile: model.MediaFile{ID: "track-1", Title: "Test Song"},
				TimeStamp: time.Now(),
			}
			err := sc.Scrobble(ctx, "user-1", scrobble)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("not authorized")))
		})

		It("returns error when plugin returns unrecoverable error", func() {
			manager, _ := createTestManagerWithPlugins(map[string]map[string]string{
				"test-scrobbler": {"error": "track rejected", "error_type": "unrecoverable"},
			}, "test-scrobbler.wasm")

			sc, ok := manager.LoadScrobbler("test-scrobbler")
			Expect(ok).To(BeTrue())

			scrobble := scrobbler.Scrobble{
				MediaFile: model.MediaFile{ID: "track-1", Title: "Test Song"},
				TimeStamp: time.Now(),
			}
			err := sc.Scrobble(ctx, "user-1", scrobble)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("unrecoverable")))
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
	It("returns nil for empty error type", func() {
		output := scrobblerOutput{ErrorType: ""}
		Expect(mapScrobblerError(output)).ToNot(HaveOccurred())
	})

	It("returns nil for 'none' error type", func() {
		output := scrobblerOutput{ErrorType: "none"}
		Expect(mapScrobblerError(output)).ToNot(HaveOccurred())
	})

	It("returns ErrNotAuthorized for 'not_authorized' error type", func() {
		output := scrobblerOutput{ErrorType: "not_authorized"}
		err := mapScrobblerError(output)
		Expect(err).To(MatchError(scrobbler.ErrNotAuthorized))
	})

	It("returns ErrNotAuthorized with message", func() {
		output := scrobblerOutput{ErrorType: "not_authorized", Error: "user not linked"}
		err := mapScrobblerError(output)
		Expect(err).To(MatchError(ContainSubstring("not authorized")))
		Expect(err).To(MatchError(ContainSubstring("user not linked")))
	})

	It("returns ErrRetryLater for 'retry_later' error type", func() {
		output := scrobblerOutput{ErrorType: "retry_later"}
		err := mapScrobblerError(output)
		Expect(err).To(MatchError(scrobbler.ErrRetryLater))
	})

	It("returns ErrUnrecoverable for 'unrecoverable' error type", func() {
		output := scrobblerOutput{ErrorType: "unrecoverable"}
		err := mapScrobblerError(output)
		Expect(err).To(MatchError(scrobbler.ErrUnrecoverable))
	})

	It("returns error for unknown error type", func() {
		output := scrobblerOutput{ErrorType: "unknown"}
		err := mapScrobblerError(output)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("unknown error type"))
	})
})
