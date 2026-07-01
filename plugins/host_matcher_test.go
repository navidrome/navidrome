//go:build !windows

package plugins

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/plugins/host"
	"github.com/navidrome/navidrome/plugins/types"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MatcherService", Ordered, func() {
	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
	})

	// newConverter returns the service as its concrete type so converter unit
	// tests can call toTrack directly with a chosen filesystem-permission flag.
	newConverter := func(hasFilesystemPerm bool) *matcherServiceImpl {
		return newMatcherService(nil, hasFilesystemPerm, newUserAccess(nil, true), newLibraryAccess(nil, true)).(*matcherServiceImpl)
	}

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
			mf.AverageRating = 4.2
			mf.Participants = model.Participants{}
			mf.Participants.Add(model.RoleArtist, model.Artist{
				ID: "ar-1", Name: "My Artist", SortArtistName: "artist, my", MbzArtistID: "mbz-ar-1",
			})
			mf.Participants.AddWithSubRole(model.RolePerformer, "violin", model.Artist{
				ID: "ar-2", Name: "A Fiddler",
			})

			track := newConverter(true).toTrack(mf, false)

			Expect(track.ID).To(Equal("mf-1"))
			Expect(track.LibraryID).To(Equal(int32(3)))
			Expect(track.LibraryName).To(Equal("Main"))
			Expect(track.Path).To(Equal("/music/song.flac"))
			Expect(track.Title).To(Equal("My Song"))
			Expect(track.Duration).To(Equal(210.5))
			Expect(track.BitDepth).To(HaveValue(Equal(int32(24))))
			Expect(track.BPM).To(HaveValue(Equal(int32(128))))
			Expect(track.RGTrackGain).To(HaveValue(Equal(-7.5)))
			Expect(track.Compilation).To(BeTrue())
			Expect(track.MbzRecordingID).To(Equal("rec-1"))
			Expect(track.Genres).To(Equal([]string{"Rock", "Pop"}))
			Expect(track.CreatedAt).To(Equal(int64(1700000000)))
			Expect(track.UpdatedAt).To(Equal(int64(1700000500)))
			Expect(track.BirthTime).To(Equal(int64(1699999000)))
			Expect(track.Tags).To(HaveKeyWithValue("isrc", []string{"US-XXX-00"}))
			// Flat, role-tagged, role-sorted.
			Expect(track.Participants).To(HaveLen(2))
			Expect(track.Participants[0]).To(Equal(types.ArtistRef{
				ID: "ar-1", Name: "My Artist", SortName: "artist, my", MBID: "mbz-ar-1", Role: "artist",
			}))
			Expect(track.Participants[1]).To(Equal(types.ArtistRef{
				ID: "ar-2", Name: "A Fiddler", Role: "performer", SubRole: "violin",
			}))
			// AverageRating is an aggregate, exposed even though the match is unscoped.
			Expect(track.AverageRating).To(Equal(4.2))
		})

		It("leaves nil-able numeric fields nil when absent", func() {
			mf := &model.MediaFile{ID: "mf-2", Title: "No Optionals"}
			track := newConverter(true).toTrack(mf, false)
			Expect(track.BitDepth).To(BeNil())
			Expect(track.BPM).To(BeNil())
			Expect(track.RGAlbumGain).To(BeNil())
			Expect(track.RGAlbumPeak).To(BeNil())
			Expect(track.RGTrackGain).To(BeNil())
			Expect(track.RGTrackPeak).To(BeNil())
		})

		It("preserves a real 0 ReplayGain value as non-nil", func() {
			zero := 0.0
			mf := &model.MediaFile{ID: "mf-3", Title: "Zero RG", RGTrackGain: &zero}
			track := newConverter(true).toTrack(mf, false)
			Expect(track.RGTrackGain).To(HaveValue(Equal(0.0)))
			Expect(track.RGAlbumGain).To(BeNil())
		})

		It("exposes Path only when the plugin has filesystem permission", func() {
			mf := &model.MediaFile{ID: "mf-4", Title: "With Path", Path: "/music/x.flac"}
			Expect(newConverter(true).toTrack(mf, false).Path).To(Equal("/music/x.flac"))
			Expect(newConverter(false).toTrack(mf, false).Path).To(BeEmpty())
		})
	})

	Describe("MatchSongs", func() {
		// The mock MediaFileRepo returns stored files (with annotations) verbatim,
		// ignoring QueryOptions and the context user. These tests therefore cover the
		// adapter's gating/access logic, not the SQL per-user join (a persistence-layer
		// concern).

		// allowAll returns a service permitted to match as any user across all
		// libraries.
		allowAll := func(ds model.DataStore) host.MatcherService {
			return newMatcherService(ds, false, newUserAccess(nil, true), newLibraryAccess(nil, true))
		}

		It("returns one entry per input song in order, with nil for no-match", func() {
			mediaFileRepo := tests.CreateMockMediaFileRepo()
			// First (ID) phase returns the match for input song 0 only.
			mediaFileRepo.SetData(model.MediaFiles{
				{ID: "mf-100", Title: "Hit", Artist: "Band"},
			})
			ds := &tests.MockDataStore{MockedMediaFile: mediaFileRepo}

			results, err := allowAll(ds).MatchSongs(GinkgoT().Context(), []types.SongRef{
				{ID: "mf-100", Name: "Hit", Artist: "Band"},
				{ID: "missing-id", Name: "Ghost", Artist: "Nobody"},
			}, host.MatchOptions{})

			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(HaveLen(2))
			Expect(results[0]).ToNot(BeNil())
			Expect(results[0].ID).To(Equal("mf-100"))
			Expect(results[1]).To(BeNil())
		})

		It("returns an empty slice for empty input", func() {
			ds := &tests.MockDataStore{MockedMediaFile: tests.CreateMockMediaFileRepo()}
			results, err := allowAll(ds).MatchSongs(GinkgoT().Context(), nil, host.MatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(results).To(BeEmpty())
		})

		Context("with a scoped user", func() {
			var ds *tests.MockDataStore
			var userRepo *tests.MockedUserRepo

			BeforeEach(func() {
				mediaFileRepo := tests.CreateMockMediaFileRepo()
				mf := model.MediaFile{ID: "mf-1", Title: "Hit", Artist: "Band", LibraryID: 1}
				mf.Starred = true
				mf.Rating = 5
				mediaFileRepo.SetData(model.MediaFiles{mf})

				userRepo = tests.CreateMockUserRepo()
				Expect(userRepo.Put(&model.User{ID: "u-alice", UserName: "alice"})).To(Succeed())

				ds = &tests.MockDataStore{MockedMediaFile: mediaFileRepo, MockedUser: userRepo}
			})

			input := []types.SongRef{{ID: "mf-1", Name: "Hit", Artist: "Band"}}

			It("does not expose annotations when no username is given", func() {
				svc := newMatcherService(ds, false, newUserAccess(nil, true), newLibraryAccess(nil, true))
				results, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0]).ToNot(BeNil())
				Expect(results[0].Starred).To(BeFalse())
				Expect(results[0].Rating).To(BeZero())
			})

			It("exposes the user's annotations when an allowed username is given", func() {
				svc := newMatcherService(ds, false, newUserAccess([]string{"u-alice"}, false), newLibraryAccess(nil, true))
				results, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{Username: "alice"})
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0]).ToNot(BeNil())
				Expect(results[0].Starred).To(BeTrue())
				Expect(results[0].Rating).To(Equal(int32(5)))
			})

			It("allows any username when allUsers is set", func() {
				svc := newMatcherService(ds, false, newUserAccess(nil, true), newLibraryAccess(nil, true))
				results, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{Username: "alice"})
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0].Starred).To(BeTrue())
			})

			It("returns an error for an unknown username", func() {
				svc := newMatcherService(ds, false, newUserAccess(nil, true), newLibraryAccess(nil, true))
				_, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{Username: "ghost"})
				Expect(err).To(MatchError(ContainSubstring("not found")))
			})

			It("surfaces a backend error rather than masking it as not-found", func() {
				userRepo.Error = errors.New("db is locked")
				svc := newMatcherService(ds, false, newUserAccess(nil, true), newLibraryAccess(nil, true))
				_, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{Username: "alice"})
				Expect(err).To(MatchError(ContainSubstring("db is locked")))
				Expect(err.Error()).ToNot(ContainSubstring("not found"))
			})

			It("returns an error for a username the plugin is not allowed to use", func() {
				svc := newMatcherService(ds, false, newUserAccess([]string{"u-bob"}, false), newLibraryAccess(nil, true))
				_, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{Username: "alice"})
				Expect(err).To(MatchError(ContainSubstring("not allowed")))
			})

			It("rejects a username with the same error whether it exists, when the plugin has no user scope", func() {
				// A plugin with no user scope (the only state a matcher-only plugin can
				// be in) must not leak whether a username exists via the error text.
				svc := newMatcherService(ds, false, newUserAccess(nil, false), newLibraryAccess(nil, true))

				_, errExisting := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{Username: "alice"})
				_, errMissing := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{Username: "ghost"})

				Expect(errExisting).To(HaveOccurred())
				Expect(errExisting.Error()).To(Equal(errMissing.Error()))
				Expect(errExisting.Error()).ToNot(ContainSubstring("not found"))
				Expect(errExisting.Error()).To(ContainSubstring("not authorized to scope by user"))
			})

			It("does not inherit the caller's request user for an unscoped match", func() {
				// The plugin may be invoked while handling another user's request; an
				// unscoped match must run as admin, not as that inherited user.
				capturing := &ctxCapturingDataStore{MockDataStore: ds}
				svc := newMatcherService(capturing, false, newUserAccess(nil, true), newLibraryAccess(nil, true))

				callerCtx := request.WithUser(GinkgoT().Context(), model.User{ID: "u-caller", UserName: "caller"})
				_, err := svc.MatchSongs(callerCtx, input, host.MatchOptions{})
				Expect(err).ToNot(HaveOccurred())

				usr, ok := request.UserFrom(capturing.lastMediaFileCtx)
				Expect(ok).To(BeTrue())
				Expect(usr.IsAdmin).To(BeTrue())
				Expect(usr.ID).ToNot(Equal("u-caller"))
			})

			It("uses the requested user, overriding an inherited caller user", func() {
				capturing := &ctxCapturingDataStore{MockDataStore: ds}
				svc := newMatcherService(capturing, false, newUserAccess(nil, true), newLibraryAccess(nil, true))

				callerCtx := request.WithUser(GinkgoT().Context(), model.User{ID: "u-caller", UserName: "caller"})
				_, err := svc.MatchSongs(callerCtx, input, host.MatchOptions{Username: "alice"})
				Expect(err).ToNot(HaveOccurred())

				usr, ok := request.UserFrom(capturing.lastMediaFileCtx)
				Expect(ok).To(BeTrue())
				Expect(usr.ID).To(Equal("u-alice"))
			})
		})

		Context("with plugin library access", func() {
			var ds *tests.MockDataStore

			BeforeEach(func() {
				mediaFileRepo := tests.CreateMockMediaFileRepo()
				mediaFileRepo.SetData(model.MediaFiles{
					{ID: "mf-lib1", Title: "A", Artist: "Band", LibraryID: 1},
					{ID: "mf-lib2", Title: "B", Artist: "Band", LibraryID: 2},
				})
				ds = &tests.MockDataStore{MockedMediaFile: mediaFileRepo}
			})

			input := []types.SongRef{
				{ID: "mf-lib1", Name: "A", Artist: "Band"},
				{ID: "mf-lib2", Name: "B", Artist: "Band"},
			}

			It("drops matches from libraries the plugin cannot access", func() {
				svc := newMatcherService(ds, false, newUserAccess(nil, true), newLibraryAccess([]int{1}, false))
				results, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0]).ToNot(BeNil())
				Expect(results[0].ID).To(Equal("mf-lib1"))
				Expect(results[1]).To(BeNil()) // library 2 not permitted
			})

			It("keeps all matches when allLibraries is set", func() {
				svc := newMatcherService(ds, false, newUserAccess(nil, true), newLibraryAccess(nil, true))
				results, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(results[0]).ToNot(BeNil())
				Expect(results[1]).ToNot(BeNil())
			})

			It("errors when the plugin has no library scope configured", func() {
				svc := newMatcherService(ds, false, newUserAccess(nil, true), newLibraryAccess(nil, false))
				_, err := svc.MatchSongs(GinkgoT().Context(), input, host.MatchOptions{})
				Expect(err).To(MatchError(ContainSubstring("no libraries configured")))
			})
		})
	})

	// Artist precedence for songRefToAgentSong is covered in metadata_agent_test.go;
	// here we cover the duration normalization the matcher path relies on.
	Describe("songRefToAgentSong duration", func() {
		It("prefers DurationMs over the deprecated seconds field", func() {
			song := songRefToAgentSong(types.SongRef{DurationMs: 247333, Duration: 99})
			Expect(song.Duration).To(Equal(uint32(247333)))
		})

		It("falls back to the seconds field when DurationMs is zero", func() {
			song := songRefToAgentSong(types.SongRef{Duration: 210.5})
			Expect(song.Duration).To(Equal(uint32(210500)))
		})

		It("clamps a negative seconds duration to zero instead of overflowing", func() {
			song := songRefToAgentSong(types.SongRef{Duration: -1})
			Expect(song.Duration).To(BeZero())
		})
	})
})

var _ = Describe("MatcherService Integration", Ordered, func() {
	var (
		manager *Manager
		tmpDir  string
	)

	BeforeAll(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "matcher-integration-test-*")
		Expect(err).ToNot(HaveOccurred())

		srcPath := filepath.Join(testdataDir, "test-matcher"+PackageExtension)
		destPath := filepath.Join(tmpDir, "test-matcher"+PackageExtension)
		data, err := os.ReadFile(srcPath)
		Expect(err).ToNot(HaveOccurred())
		err = os.WriteFile(destPath, data, 0600)
		Expect(err).ToNot(HaveOccurred())

		hash := sha256.Sum256(data)
		hashHex := hex.EncodeToString(hash[:])

		DeferCleanup(configtest.SetupConfig())
		conf.Server.Plugins.Enabled = true
		conf.Server.Plugins.Folder = conf.NewDir(tmpDir)
		conf.Server.Plugins.AutoReload = false

		mockPluginRepo := tests.CreateMockPluginRepo()
		mockPluginRepo.Permitted = true
		// AllLibraries: the matcher requires a library scope.
		mockPluginRepo.SetData(model.Plugins{{
			ID:           "test-matcher",
			Path:         destPath,
			SHA256:       hashHex,
			Enabled:      true,
			AllUsers:     true,
			AllLibraries: true,
		}})

		mediaFileRepo := tests.CreateMockMediaFileRepo()
		hit := model.MediaFile{ID: "mf-hit", Title: "Hit", Artist: "Band"}
		hit.Starred = true
		mediaFileRepo.SetData(model.MediaFiles{hit})

		userRepo := tests.CreateMockUserRepo()
		Expect(userRepo.Put(&model.User{ID: "u-alice", UserName: "alice"})).To(Succeed())

		dataStore := &tests.MockDataStore{
			MockedPlugin:    mockPluginRepo,
			MockedMediaFile: mediaFileRepo,
			MockedUser:      userRepo,
		}

		manager = &Manager{
			plugins:        make(map[string]*plugin),
			ds:             dataStore,
			subsonicRouter: http.NotFoundHandler(),
		}
		Expect(manager.Start(GinkgoT().Context())).To(Succeed())

		DeferCleanup(func() {
			_ = manager.Stop()
			_ = os.RemoveAll(tmpDir)
		})
	})

	It("loads the plugin with the matcher permission", func() {
		manager.mu.RLock()
		p, ok := manager.plugins["test-matcher"]
		manager.mu.RUnlock()
		Expect(ok).To(BeTrue())
		Expect(p.manifest.Permissions).ToNot(BeNil())
		Expect(p.manifest.Permissions.Matcher).ToNot(BeNil())
	})

	It("matches songs through the host boundary, preserving order and nils", func() {
		ctx := GinkgoT().Context()
		manager.mu.RLock()
		p := manager.plugins["test-matcher"]
		manager.mu.RUnlock()

		instance, err := p.instance(ctx)
		Expect(err).ToNot(HaveOccurred())
		defer instance.Close(ctx)

		type tIn struct {
			Songs    []types.SongRef `json:"songs"`
			Username string          `json:"username,omitempty"`
		}
		type tOut struct {
			MatchedIDs []string `json:"matched_ids"`
			Starred    []bool   `json:"starred"`
			Error      *string  `json:"error,omitempty"`
		}

		call := func(in tIn) tOut {
			inputBytes, err := json.Marshal(in)
			Expect(err).ToNot(HaveOccurred())
			_, outputBytes, err := instance.Call("nd_test_matcher", inputBytes)
			Expect(err).ToNot(HaveOccurred())
			var out tOut
			Expect(json.Unmarshal(outputBytes, &out)).To(Succeed())
			Expect(out.Error).To(BeNil())
			return out
		}

		songs := []types.SongRef{
			{ID: "mf-hit", Name: "Hit", Artist: "Band"},
			{ID: "nope", Name: "Ghost", Artist: "Nobody"},
		}

		By("matching without a user, preserving order and nils")
		out := call(tIn{Songs: songs})
		Expect(out.MatchedIDs).To(HaveLen(2))
		Expect(out.MatchedIDs[0]).To(Equal("mf-hit"))
		Expect(out.MatchedIDs[1]).To(BeEmpty())
		Expect(out.Starred[0]).To(BeFalse()) // no user scope → no annotations

		By("matching as a user, exposing that user's annotations across the boundary")
		scoped := call(tIn{Songs: songs, Username: "alice"})
		Expect(scoped.MatchedIDs[0]).To(Equal("mf-hit"))
		Expect(scoped.Starred[0]).To(BeTrue())
	})
})

// ctxCapturingDataStore records the context passed to MediaFile so tests can assert
// which user the matcher resolved before querying the library.
type ctxCapturingDataStore struct {
	*tests.MockDataStore
	lastMediaFileCtx context.Context
}

func (d *ctxCapturingDataStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	d.lastMediaFileCtx = ctx
	return d.MockDataStore.MediaFile(ctx)
}
