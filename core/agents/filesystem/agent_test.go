package filesystem

import (
	"context"

	"github.com/navidrome/navidrome/core/agents"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/navidrome/navidrome/utils/gg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("localAgent", func() {
	var ds model.DataStore
	var ctx context.Context
	var agent *filesystemAgent

	BeforeEach(func() {
		ds = &tests.MockDataStore{}
		ctx = context.Background()
		agent = filesystemConstructor(ds)
	})

	Describe("GetArtistBiography", func() {
		BeforeEach(func() {
			ds.Artist(ctx).(*tests.MockArtistRepo).SetData(model.Artists{
				model.Artist{ID: "ar-1234",
					Name: "artist"},
			})
		})

		It("should fetch artist biography", func() {

			ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
				model.Album{ID: "al-1234",
					AlbumArtistID: "ar-1234",
					Name:          "album",
					Paths:         "tests/fixtures/artist/an-album",
				},
			})

			bio, err := agent.GetArtistBiography(ctx, "ar-1234", "album", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(bio).To(Equal("This is an artist biography"))
		})

		It("should fetch artist biography with slash", func() {
			ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
				model.Album{ID: "al-1234",
					AlbumArtistID: "ar-1234",
					Name:          "album",
					Paths:         "tests/fixtures/artist/",
				},
			})

			bio, err := agent.GetArtistBiography(ctx, "ar-1234", "album", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(bio).To(Equal("This is an artist biography"))
		})

		It("should error when file doesn't exist", func() {
			ds.Album(ctx).(*tests.MockAlbumRepo).SetData(model.Albums{
				model.Album{ID: "al-1234",
					AlbumArtistID: "ar-1234",
					Name:          "album",
					Paths:         "tests/fixtures/fake-artist/fake-album",
				},
			})

			bio, err := agent.GetArtistBiography(ctx, "ar-1234", "album", "")
			Expect(err).To(Equal(agents.ErrNotFound))
			Expect(bio).To(Equal(""))
		})
	})

	Describe("GetSongLyrics", func() {
		It("should parse LRC file", func() {
			mf := model.MediaFile{
				Path: "tests/fixtures/01 Invisible (RED) Edit Version.mp3",
			}

			lyrics, err := agent.GetSongLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(Equal(model.LyricList{
				{
					DisplayArtist: "",
					DisplayTitle:  "",
					Lang:          "xxx",
					Line: []model.Line{
						{Start: P(int64(0)), Value: "Line 1"},
						{Start: P(int64(5210)), Value: "Line 2"},
						{Start: P(int64(12450)), Value: "Line 3"},
					},
					Offset: nil,
					Synced: true,
				},
			}))
		})

		It("should parse both LRC and TXT", func() {
			mf := model.MediaFile{
				Path: "tests/fixtures/test.wav",
			}

			lyrics, err := agent.GetSongLyrics(ctx, &mf)
			Expect(err).ToNot(HaveOccurred())
			Expect(lyrics).To(Equal(model.LyricList{
				{
					DisplayArtist: "Artist",
					DisplayTitle:  "Title",
					Lang:          "xxx",
					Line: []model.Line{
						{Start: P(int64(0)), Value: "Line 1"},
						{Start: P(int64(5210)), Value: "Line 2"},
						{Start: P(int64(12450)), Value: "Line 5"},
					},
					Offset: P(int64(100)),
					Synced: true,
				},
				{

					DisplayArtist: "",
					DisplayTitle:  "",
					Lang:          "xxx",
					Line: []model.Line{
						{
							Start: nil,
							Value: "Unsynchronized lyric line 1",
						},
						{
							Start: nil,
							Value: "Unsynchronized lyric line 2",
						},
						{
							Start: nil,
							Value: "Unsynchronized lyric line 3",
						},
					},
					Offset: nil,
					Synced: false,
				},
			}))
		})
	})
})
