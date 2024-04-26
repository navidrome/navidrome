package core

import (
	"context"
	"os"
	"time"

	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlists", func() {
	var ds model.DataStore
	var ps Playlists
	var mp mockedPlaylist
	ctx := context.Background()

	BeforeEach(func() {
		mp = mockedPlaylist{}
		ds = &tests.MockDataStore{
			MockedMediaFile: &mockedMediaFile{},
			MockedPlaylist:  &mp,
		}
		ctx = request.WithUser(ctx, model.User{ID: "123"})
	})

	Describe("ImportFile", func() {
		BeforeEach(func() {
			ps = NewPlaylists(ds)
		})

		Describe("M3U", func() {
			It("parses well-formed playlists", func() {
				pls, err := ps.ImportFile(ctx, "tests/fixtures", "playlists/pls1.m3u")
				Expect(err).To(BeNil())
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Tracks).To(HaveLen(3))
				Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/test.mp3"))
				Expect(pls.Tracks[1].Path).To(Equal("tests/fixtures/test.ogg"))
				Expect(pls.Tracks[2].Path).To(Equal("/tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
				Expect(mp.last).To(Equal(pls))
			})

			It("parses playlists using LF ending", func() {
				pls, err := ps.ImportFile(ctx, "tests/fixtures/playlists", "lf-ended.m3u")
				Expect(err).To(BeNil())
				Expect(pls.Tracks).To(HaveLen(2))
			})

			It("parses playlists using CR ending (old Mac format)", func() {
				pls, err := ps.ImportFile(ctx, "tests/fixtures/playlists", "cr-ended.m3u")
				Expect(err).To(BeNil())
				Expect(pls.Tracks).To(HaveLen(2))
			})
		})

		Describe("NSP", func() {
			It("parses well-formed playlists", func() {
				pls, err := ps.ImportFile(ctx, "tests/fixtures", "playlists/recently_played.nsp")
				Expect(err).To(BeNil())
				Expect(mp.last).To(Equal(pls))
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Name).To(Equal("Recently Played"))
				Expect(pls.Comment).To(Equal("Recently played tracks"))
				Expect(pls.Rules.Sort).To(Equal("lastPlayed"))
				Expect(pls.Rules.Order).To(Equal("desc"))
				Expect(pls.Rules.Limit).To(Equal(100))
				Expect(pls.Rules.Expression).To(BeAssignableToTypeOf(criteria.All{}))
			})
		})
	})

	Describe("ImportM3U", func() {
		BeforeEach(func() {
			ps = NewPlaylists(ds)
			ctx = request.WithUser(ctx, model.User{ID: "123"})
		})

		It("parses well-formed playlists", func() {
			f, _ := os.Open("tests/fixtures/playlists/pls-post-with-name.m3u")
			defer f.Close()
			pls, err := ps.ImportM3U(ctx, f)
			Expect(pls.OwnerID).To(Equal("123"))
			Expect(pls.Name).To(Equal("playlist 1"))
			Expect(err).To(BeNil())
			Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/test.mp3"))
			Expect(pls.Tracks[1].Path).To(Equal("tests/fixtures/test.ogg"))
			Expect(pls.Tracks[2].Path).To(Equal("/tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
			Expect(mp.last).To(Equal(pls))
			f.Close()

		})

		It("sets the playlist name as a timestamp if the #PLAYLIST directive is not present", func() {
			f, _ := os.Open("tests/fixtures/playlists/pls-post.m3u")
			defer f.Close()
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).To(BeNil())
			_, err = time.Parse(time.RFC3339, pls.Name)
			Expect(err).To(BeNil())
		})
	})
})

type mockedMediaFile struct {
	model.MediaFileRepository
}

func (r *mockedMediaFile) FindByPath(s string) (*model.MediaFile, error) {
	return &model.MediaFile{
		ID:   "123",
		Path: s,
	}, nil
}

type mockedPlaylist struct {
	last *model.Playlist
	model.PlaylistRepository
}

func (r *mockedPlaylist) FindByPath(string) (*model.Playlist, error) {
	return nil, model.ErrNotFound
}

func (r *mockedPlaylist) Put(pls *model.Playlist) error {
	r.last = pls
	return nil
}
