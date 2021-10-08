package scanner

import (
	"context"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("playlistSync", func() {
	var ds model.DataStore
	var ps *playlistSync
	ctx := context.Background()

	BeforeEach(func() {
		ds = &tests.MockDataStore{
			MockedMediaFile: &mockedMediaFile{},
			MockedPlaylist:  &mockedPlaylist{},
		}
	})

	Describe("parsePlaylist", func() {
		BeforeEach(func() {
			ps = newPlaylistSync(ds, "tests/")
		})

		It("parses well-formed playlists", func() {
			pls, err := ps.parsePlaylist(ctx, "playlists/pls1.m3u", "tests/fixtures")
			Expect(err).To(BeNil())
			Expect(pls.Tracks).To(HaveLen(3))
			Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/test.mp3"))
			Expect(pls.Tracks[1].Path).To(Equal("tests/fixtures/test.ogg"))
			Expect(pls.Tracks[2].Path).To(Equal("/tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
		})

		It("parses playlists using LF ending", func() {
			pls, err := ps.parsePlaylist(ctx, "lf-ended.m3u", "tests/fixtures/playlists")
			Expect(err).To(BeNil())
			Expect(pls.Tracks).To(HaveLen(2))
		})

		It("parses playlists using CR ending (old Mac format)", func() {
			pls, err := ps.parsePlaylist(ctx, "cr-ended.m3u", "tests/fixtures/playlists")
			Expect(err).To(BeNil())
			Expect(pls.Tracks).To(HaveLen(2))
		})

	})

	Describe("processPlaylists", func() {
		Context("Default PlaylistsPath", func() {
			BeforeEach(func() {
				conf.Server.PlaylistsPath = consts.DefaultPlaylistsPath
			})
			It("finds and import playlists at the top level", func() {
				ps = newPlaylistSync(ds, "tests/fixtures/playlists/subfolder1")
				Expect(ps.processPlaylists(ctx, "tests/fixtures/playlists/subfolder1")).To(Equal(int64(1)))
			})

			It("finds and import playlists at any subfolder level", func() {
				ps = newPlaylistSync(ds, "tests")
				Expect(ps.processPlaylists(ctx, "tests/fixtures/playlists/subfolder1")).To(Equal(int64(1)))
			})
		})

		It("ignores playlists not in the PlaylistsPath", func() {
			conf.Server.PlaylistsPath = "subfolder1"
			ps = newPlaylistSync(ds, "tests/fixtures/playlists")

			Expect(ps.processPlaylists(ctx, "tests/fixtures/playlists/subfolder1")).To(Equal(int64(1)))
			Expect(ps.processPlaylists(ctx, "tests/fixtures/playlists/subfolder2")).To(Equal(int64(0)))
		})

		It("only imports playlists from the root of MusicFolder if PlaylistsPath is '.'", func() {
			conf.Server.PlaylistsPath = "."
			ps = newPlaylistSync(ds, "tests/fixtures/playlists")

			Expect(ps.processPlaylists(ctx, "tests/fixtures/playlists")).To(Equal(int64(3)))
			Expect(ps.processPlaylists(ctx, "tests/fixtures/playlists/subfolder1")).To(Equal(int64(0)))
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
	model.PlaylistRepository
}

func (r *mockedPlaylist) FindByPath(path string) (*model.Playlist, error) {
	return nil, model.ErrNotFound
}

func (r *mockedPlaylist) Put(pls *model.Playlist) error {
	return nil
}
