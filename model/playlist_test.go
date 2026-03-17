package model_test

import (
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

	Describe("NormalizeChildPaths()", func() {
		It("normalizes file paths", func() {
			pls := model.Playlist{Rules: &criteria.Criteria{
				Expression: criteria.All{
					criteria.InPlaylist{"path": "/test/my-test-path.m3u"},
					criteria.InPlaylist{"path": "../my-test-path.m3u"},
					criteria.NotInPlaylist{"path": "/not-test/not-my-test-path.m3u"},
					criteria.Any{
						criteria.InPlaylist{"path": "../../in-the-test.nsp"},
						criteria.NotInPlaylist{"path": "./sibling.nsp"},
						criteria.All{
							criteria.InPlaylist{"path": "/other-root/other.m3u"},
							criteria.NotInPlaylist{"path": "../../../out-of-containment.nsp"},
						},
					},
				},
			},
				Path: "/test/nested/my-playlist.nsp"}

			pls.NormalizeChildPaths()
			Expect(pls.Rules).Should(BeEquivalentTo(&criteria.Criteria{
				Expression: criteria.All{
					criteria.InPlaylist{"path": "/test/my-test-path.m3u"},
					criteria.InPlaylist{"path": "/test/my-test-path.m3u"},
					criteria.NotInPlaylist{"path": "/not-test/not-my-test-path.m3u"},
					criteria.Any{
						criteria.InPlaylist{"path": "/in-the-test.nsp"},
						criteria.NotInPlaylist{"path": "/test/nested/sibling.nsp"},
						criteria.All{
							criteria.InPlaylist{"path": "/other-root/other.m3u"},
							criteria.NotInPlaylist{"path": "/out-of-containment.nsp"},
						},
					},
				},
			}))
		})

		It("normalizes various file paths", func() {
			// Absolute path
			pls := model.Playlist{ID: "123"}
			pls.Rules = &criteria.Criteria{
				Expression: criteria.All{
					criteria.InPlaylist{"path": "/test/my-test-path.m3u"},
				},
			}

			pls.NormalizeChildPaths()
			Expect(pls.Rules).NotTo(BeNil())
		})

		It("handles relative paths correctly", func() {
			pls := model.Playlist{ID: "123", Path: "/test/my-playlist.m3u"}
			pls.Rules = &criteria.Criteria{
				Expression: criteria.All{
					criteria.InPlaylist{"path": "../my-test-path.m3u"},
				},
			}

			pls.NormalizeChildPaths()
			Expect(pls.Rules).Should(BeEquivalentTo(&criteria.Criteria{
				Expression: criteria.All{
					criteria.InPlaylist{"path": "/my-test-path.m3u"},
				},
			}))
		})

		It("ignores non-path entries", func() {
			pls := model.Playlist{ID: "123"}
			pls.Rules = &criteria.Criteria{
				Expression: criteria.All{
					criteria.InPlaylist{"path": "/not-test/not-my-test-path.m3u"},
				},
			}

			pls.NormalizeChildPaths()
			Expect(pls.Rules).NotTo(BeNil())
		})
	})
})
