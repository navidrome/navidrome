package core_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Archiver", func() {
	var (
		arch core.Archiver
		ms   *mockMediaStreamer
		ds   *mockDataStore
		sh   *mockShare
	)

	BeforeEach(func() {
		ms = &mockMediaStreamer{}
		sh = &mockShare{}
		ds = &mockDataStore{}
		arch = core.NewArchiver(ms, ds, sh)
	})

	Context("ZipAlbum", func() {
		It("zips an album correctly", func() {
			mfs := model.MediaFiles{
				{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album/Promo", DiscNumber: 1},
				{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album/Promo", DiscNumber: 1},
			}

			mfRepo := &mockMediaFileRepository{}
			mfRepo.On("GetAll", []model.QueryOptions{{
				Filters: squirrel.Eq{"album_id": "1"},
				Sort:    "album",
			}}).Return(mfs, nil)

			ds.On("MediaFile", mock.Anything).Return(mfRepo)
			ms.On("DoStream", mock.Anything, mock.Anything, "mp3", 128, 0).Return(io.NopCloser(strings.NewReader("test")), nil).Times(3)

			out := new(bytes.Buffer)
			err := arch.ZipAlbum(context.Background(), "1", "mp3", 128, out)
			Expect(err).To(BeNil())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).To(BeNil())

			Expect(len(zr.File)).To(Equal(2))
			Expect(zr.File[0].Name).To(Equal("Album_Promo/01 - track1.mp3"))
			Expect(zr.File[1].Name).To(Equal("Album_Promo/02 - track2.mp3"))
		})
	})

	Context("ZipArtist", func() {
		It("zips an artist's albums correctly", func() {
			mfs := model.MediaFiles{
				{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumArtistID: "1", AlbumID: "1", Album: "Album 1", DiscNumber: 1},
				{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumArtistID: "1", AlbumID: "1", Album: "Album 1", DiscNumber: 1},
			}

			mfRepo := &mockMediaFileRepository{}
			mfRepo.On("GetAll", []model.QueryOptions{{
				Filters: squirrel.Eq{"album_artist_id": "1"},
				Sort:    "album",
			}}).Return(mfs, nil)

			ds.On("MediaFile", mock.Anything).Return(mfRepo)
			ms.On("DoStream", mock.Anything, mock.Anything, "mp3", 128, 0).Return(io.NopCloser(strings.NewReader("test")), nil).Times(2)

			out := new(bytes.Buffer)
			err := arch.ZipArtist(context.Background(), "1", "mp3", 128, out)
			Expect(err).To(BeNil())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).To(BeNil())

			Expect(len(zr.File)).To(Equal(2))
			Expect(zr.File[0].Name).To(Equal("Album 1/01 - track1.mp3"))
			Expect(zr.File[1].Name).To(Equal("Album 1/02 - track2.mp3"))
		})
	})

	Context("ZipShare", func() {
		It("zips a share correctly", func() {
			mfs := model.MediaFiles{
				{ID: "1", Path: "test_data/01 - track1.mp3", Suffix: "mp3", Artist: "Artist 1", Title: "track1"},
				{ID: "2", Path: "test_data/02 - track2.mp3", Suffix: "mp3", Artist: "Artist 2", Title: "track2"},
			}

			share := &model.Share{
				ID:           "1",
				Downloadable: true,
				Format:       "mp3",
				MaxBitRate:   128,
				Tracks:       mfs,
			}

			sh.On("Load", mock.Anything, "1").Return(share, nil)
			ms.On("DoStream", mock.Anything, mock.Anything, "mp3", 128, 0).Return(io.NopCloser(strings.NewReader("test")), nil).Times(2)

			out := new(bytes.Buffer)
			err := arch.ZipShare(context.Background(), "1", out)
			Expect(err).To(BeNil())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).To(BeNil())

			Expect(len(zr.File)).To(Equal(2))
			Expect(zr.File[0].Name).To(Equal("01 - Artist 1 - track1.mp3"))
			Expect(zr.File[1].Name).To(Equal("02 - Artist 2 - track2.mp3"))

		})
	})

	Context("ZipPlaylist", func() {
		It("zips a playlist correctly", func() {
			tracks := []model.PlaylistTrack{
				{MediaFile: model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1, Artist: "AC/DC", Title: "track1"}},
				{MediaFile: model.MediaFile{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1, Artist: "Artist 2", Title: "track2"}},
			}

			pls := &model.Playlist{
				ID:     "1",
				Name:   "Test Playlist",
				Tracks: tracks,
			}

			plRepo := &mockPlaylistRepository{}
			plRepo.On("GetWithTracks", "1", true, false).Return(pls, nil)
			ds.On("Playlist", mock.Anything).Return(plRepo)
			ms.On("DoStream", mock.Anything, mock.Anything, "mp3", 128, 0).Return(io.NopCloser(strings.NewReader("test")), nil).Times(2)

			out := new(bytes.Buffer)
			err := arch.ZipPlaylist(context.Background(), "1", "mp3", 128, out)
			Expect(err).To(BeNil())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).To(BeNil())

			Expect(len(zr.File)).To(Equal(3))
			Expect(zr.File[0].Name).To(Equal("01 - AC_DC - track1.mp3"))
			Expect(zr.File[1].Name).To(Equal("02 - Artist 2 - track2.mp3"))
			Expect(zr.File[2].Name).To(Equal("Test Playlist.m3u"))

			// Verify M3U content
			m3uFile, err := zr.File[2].Open()
			Expect(err).To(BeNil())
			defer m3uFile.Close()

			m3uContent, err := io.ReadAll(m3uFile)
			Expect(err).To(BeNil())

			expectedM3U := "#EXTM3U\n#PLAYLIST:Test Playlist\n#EXTINF:0,AC/DC - track1\n01 - AC_DC - track1.mp3\n#EXTINF:0,Artist 2 - track2\n02 - Artist 2 - track2.mp3\n"
			Expect(string(m3uContent)).To(Equal(expectedM3U))
		})
	})
})

type mockDataStore struct {
	mock.Mock
	model.DataStore
}

func (m *mockDataStore) MediaFile(ctx context.Context) model.MediaFileRepository {
	args := m.Called(ctx)
	return args.Get(0).(model.MediaFileRepository)
}

func (m *mockDataStore) Playlist(ctx context.Context) model.PlaylistRepository {
	args := m.Called(ctx)
	return args.Get(0).(model.PlaylistRepository)
}

func (m *mockDataStore) Library(context.Context) model.LibraryRepository {
	return &mockLibraryRepository{}
}

type mockLibraryRepository struct {
	mock.Mock
	model.LibraryRepository
}

func (m *mockLibraryRepository) GetPath(id int) (string, error) {
	return "/music", nil
}

type mockMediaFileRepository struct {
	mock.Mock
	model.MediaFileRepository
}

func (m *mockMediaFileRepository) GetAll(options ...model.QueryOptions) (model.MediaFiles, error) {
	args := m.Called(options)
	return args.Get(0).(model.MediaFiles), args.Error(1)
}

type mockPlaylistRepository struct {
	mock.Mock
	model.PlaylistRepository
}

func (m *mockPlaylistRepository) GetWithTracks(id string, refreshSmartPlaylists, includeMissing bool) (*model.Playlist, error) {
	args := m.Called(id, refreshSmartPlaylists, includeMissing)
	return args.Get(0).(*model.Playlist), args.Error(1)
}

type mockMediaStreamer struct {
	mock.Mock
	core.MediaStreamer
}

func (m *mockMediaStreamer) DoStream(ctx context.Context, mf *model.MediaFile, reqFormat string, reqBitRate int, reqOffset int) (*core.Stream, error) {
	args := m.Called(ctx, mf, reqFormat, reqBitRate, reqOffset)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return &core.Stream{ReadCloser: args.Get(0).(io.ReadCloser)}, nil
}

type mockShare struct {
	mock.Mock
	core.Share
}

func (m *mockShare) Load(ctx context.Context, id string) (*model.Share, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Share), args.Error(1)
}
