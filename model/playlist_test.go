package model_test

import (
	"path/filepath"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlist", func() {
	Describe("ToM3U8()", func() {
		var pls model.Playlist
		BeforeEach(func() {
			pls = model.Playlist{Name: "Mellow sunset"}
			pls.Tracks = model.PlaylistTracks{
				{MediaFile: model.MediaFile{Artist: "Morcheeba feat. Kurt Wagner", Title: "What New York Couples Fight About",
					Duration:    377.84,
					LibraryPath: "/music/library", Path: "Morcheeba/Charango/01-06 What New York Couples Fight About.mp3"}},
				{MediaFile: model.MediaFile{Artist: "A Tribe Called Quest", Title: "Description of a Fool (Groove Armada's Acoustic mix)",
					Duration:    374.49,
					LibraryPath: "/music/library", Path: "Groove Armada/Back to Mine_ Groove Armada/01-01 Description of a Fool (Groove Armada's Acoustic mix).mp3"}},
				{MediaFile: model.MediaFile{Artist: "Lou Reed", Title: "Walk on the Wild Side",
					Duration:    253.1,
					LibraryPath: "/music/library", Path: "Lou Reed/Walk on the Wild Side_ The Best of Lou Reed/01-06 Walk on the Wild Side.m4a"}},
				{MediaFile: model.MediaFile{Artist: "Legião Urbana", Title: "On the Way Home",
					Duration:    163.89,
					LibraryPath: "/music/library", Path: "Legião Urbana/Música p_ acampamentos/02-05 On the Way Home.mp3"}},
			}
		})
		It("generates the correct M3U format", func() {
			tests.SkipOnWindows("path separator bug (#TBD-path-sep-model)")
			expected := `#EXTM3U
#PLAYLIST:Mellow sunset
#EXTINF:378,Morcheeba feat. Kurt Wagner - What New York Couples Fight About
/music/library/Morcheeba/Charango/01-06 What New York Couples Fight About.mp3
#EXTINF:374,A Tribe Called Quest - Description of a Fool (Groove Armada's Acoustic mix)
/music/library/Groove Armada/Back to Mine_ Groove Armada/01-01 Description of a Fool (Groove Armada's Acoustic mix).mp3
#EXTINF:253,Lou Reed - Walk on the Wild Side
/music/library/Lou Reed/Walk on the Wild Side_ The Best of Lou Reed/01-06 Walk on the Wild Side.m4a
#EXTINF:164,Legião Urbana - On the Way Home
/music/library/Legião Urbana/Música p_ acampamentos/02-05 On the Way Home.mp3
`
			Expect(pls.ToM3U8()).To(Equal(expected))
		})
	})

	Describe("RefreshDelay", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.SmartPlaylistRefreshDelay = 5 * time.Second
		})

		It("returns the global config value when rules have no refreshDelay", func() {
			pls := model.Playlist{Rules: &criteria.Criteria{Expression: criteria.All{criteria.Is{"loved": true}}}}
			Expect(pls.RefreshDelay()).To(Equal(5 * time.Second))
		})

		It("returns the per-playlist value when set", func() {
			pls := model.Playlist{Rules: &criteria.Criteria{
				Expression:   criteria.All{criteria.Is{"loved": true}},
				RefreshDelay: 24 * time.Hour,
			}}
			Expect(pls.RefreshDelay()).To(Equal(24 * time.Hour))
		})

		It("returns the global value for non-smart playlists", func() {
			pls := model.Playlist{}
			Expect(pls.RefreshDelay()).To(Equal(5 * time.Second))
		})
	})

	Describe("TracksEditable", func() {
		It("is true for a plain playlist", func() {
			Expect(model.Playlist{}.TracksEditable()).To(BeTrue())
		})

		It("is false for a smart playlist", func() {
			pls := model.Playlist{Rules: &criteria.Criteria{Expression: criteria.Is{"loved": true}}}
			Expect(pls.TracksEditable()).To(BeFalse())
		})

		It("is false for a synced playlist", func() {
			Expect(model.Playlist{Sync: true}.TracksEditable()).To(BeFalse())
		})
	})

	Describe("NormalizedRules()", func() {
		// absPath builds an OS-native absolute path so these specs also run on Windows.
		absPath := func(parts ...string) string {
			abs, err := filepath.Abs(filepath.Join(parts...))
			Expect(err).ToNot(HaveOccurred())
			return abs
		}
		normalize := func(pls model.Playlist) criteria.Expression {
			return pls.NormalizedRules().Expression
		}

		It("resolves relative references against the playlist folder", func() {
			pls := model.Playlist{
				Path: absPath("test", "nested", "my-playlist.nsp"),
				Rules: &criteria.Criteria{Expression: criteria.All{
					criteria.InPlaylist{"path": "../up.m3u"},
					criteria.NotInPlaylist{"path": "./sibling.nsp"},
					criteria.Any{criteria.InPlaylist{"path": "sub/deep.nsp"}},
				}},
			}
			Expect(normalize(pls)).To(BeEquivalentTo(criteria.All{
				criteria.InPlaylist{"path": absPath("test", "up.m3u")},
				criteria.NotInPlaylist{"path": absPath("test", "nested", "sibling.nsp")},
				criteria.Any{criteria.InPlaylist{"path": absPath("test", "nested", "sub", "deep.nsp")}},
			}))
		})

		It("cleans absolute references", func() {
			dirty := absPath("music") + string(filepath.Separator) + "." + string(filepath.Separator) + "child.nsp"
			pls := model.Playlist{
				Path:  absPath("test", "my-playlist.nsp"),
				Rules: &criteria.Criteria{Expression: criteria.All{criteria.NotInPlaylist{"path": dirty}}},
			}
			Expect(normalize(pls)).To(BeEquivalentTo(criteria.All{
				criteria.NotInPlaylist{"path": absPath("music", "child.nsp")},
			}))
		})

		It("treats a leading slash as absolute on every OS", func() {
			pls := model.Playlist{
				Path:  absPath("test", "my-playlist.nsp"),
				Rules: &criteria.Criteria{Expression: criteria.All{criteria.InPlaylist{"path": "/other/./root.m3u"}}},
			}
			Expect(normalize(pls)).To(BeEquivalentTo(criteria.All{
				criteria.InPlaylist{"path": filepath.FromSlash("/other/root.m3u")},
			}))
		})

		It("leaves empty paths and id references untouched", func() {
			pls := model.Playlist{
				Path: absPath("test", "my-playlist.nsp"),
				Rules: &criteria.Criteria{Expression: criteria.All{
					criteria.InPlaylist{"path": ""},
					criteria.InPlaylist{"id": "94d8ba52-7aca-40e2-af82-4cb09c43d710"},
					criteria.Eq{"artist": "Bob Dealin"},
				}},
			}
			Expect(normalize(pls)).To(BeEquivalentTo(criteria.All{
				criteria.InPlaylist{"path": ""},
				criteria.InPlaylist{"id": "94d8ba52-7aca-40e2-af82-4cb09c43d710"},
				criteria.Eq{"artist": "Bob Dealin"},
			}))
		})

		It("skips relative references when the playlist has no path", func() {
			pls := model.Playlist{
				Rules: &criteria.Criteria{Expression: criteria.All{criteria.InPlaylist{"path": "../up.m3u"}}},
			}
			Expect(normalize(pls)).To(BeEquivalentTo(criteria.All{
				criteria.InPlaylist{"path": filepath.FromSlash("../up.m3u")},
			}))
		})

		It("preserves every other criteria field", func() {
			rules := criteria.Criteria{
				Expression:   criteria.All{criteria.InPlaylist{"path": "child.nsp"}},
				Sort:         "title",
				Order:        "desc",
				Limit:        10,
				LimitPercent: 25,
				Offset:       5,
				RefreshDelay: 3 * time.Hour,
			}
			pls := model.Playlist{Path: absPath("test", "my-playlist.nsp"), Rules: &rules}

			normalized := *pls.NormalizedRules()
			normalized.Expression = rules.Expression
			Expect(normalized).To(Equal(rules))
		})

		It("does not mutate the original playlist rules", func() {
			original := criteria.All{criteria.InPlaylist{"path": "child.nsp"}}
			pls := model.Playlist{Path: absPath("test", "my-playlist.nsp"), Rules: &criteria.Criteria{Expression: original}}
			_ = pls.NormalizedRules()
			Expect(original[0]).To(BeEquivalentTo(criteria.InPlaylist{"path": "child.nsp"}))
		})
	})
})
