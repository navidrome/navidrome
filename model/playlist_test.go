package model_test

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlist", func() {
	Describe("ImageFilename", func() {
		It("returns ID_cleanname.ext for a normal name", func() {
			pls := model.Playlist{ID: "abc123", Name: "My Cool Playlist"}
			Expect(pls.ImageFilename(".jpg")).To(Equal("abc123_my_cool_playlist.jpg"))
		})

		It("falls back to ID.ext when name cleans to empty", func() {
			pls := model.Playlist{ID: "abc123", Name: "!!!"}
			Expect(pls.ImageFilename(".png")).To(Equal("abc123.png"))
		})

		It("falls back to ID.ext for empty name", func() {
			pls := model.Playlist{ID: "abc123", Name: ""}
			Expect(pls.ImageFilename(".jpg")).To(Equal("abc123.jpg"))
		})

		It("handles names with special characters", func() {
			pls := model.Playlist{ID: "x1", Name: "Rock & Roll! (2024)"}
			Expect(pls.ImageFilename(".webp")).To(Equal("x1_rock__roll_2024.webp"))
		})
	})

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
})
