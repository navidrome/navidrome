package core

import (
	"context"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlists", func() {
	var ds model.DataStore
	var ps Playlists
	ctx := context.Background()

	BeforeEach(func() {
		ds = &tests.MockDataStore{
			MockedMediaFile: &mockedMediaFile{},
			MockedPlaylist:  &mockedPlaylist{},
		}
	})

	Describe("ImportFile", func() {
		BeforeEach(func() {
			ps = NewPlaylists(ds)
		})

		It("parses well-formed playlists", func() {
			pls, err := ps.ImportFile(ctx, "tests/fixtures", "playlists/pls1.m3u")
			Expect(err).To(BeNil())
			Expect(pls.Tracks).To(HaveLen(3))
			Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/test.mp3"))
			Expect(pls.Tracks[1].Path).To(Equal("tests/fixtures/test.ogg"))
			Expect(pls.Tracks[2].Path).To(Equal("/tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
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

func (r *mockedPlaylist) FindByPath(string) (*model.Playlist, error) {
	return nil, model.ErrNotFound
}

func (r *mockedPlaylist) Put(*model.Playlist) error {
	return nil
}
