package core_test

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
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
			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).Return(io.NopCloser(strings.NewReader("test")), nil).Times(3)

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
				Filters: squirrel.And{
					persistence.ParticipantIDFilter("media_file", "1", model.RoleAlbumArtist),
					squirrel.Eq{"missing": false},
				},
				Sort: "album",
			}}).Return(mfs, nil)

			ds.On("MediaFile", mock.Anything).Return(mfRepo)
			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).Return(io.NopCloser(strings.NewReader("test")), nil).Times(2)

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

	Context("ZipArtist with albums that share a name", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
		})

		// zipArtistEntries zips the given tracks as artist "1" and returns the entry names in zip order.
		zipArtistEntries := func(mfs model.MediaFiles) []string {
			mfRepo := &mockMediaFileRepository{}
			mfRepo.On("GetAll", mock.Anything).Return(mfs, nil)
			ds.On("MediaFile", mock.Anything).Return(mfRepo)
			ms.On("NewStream", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader("test")), nil)

			out := new(bytes.Buffer)
			Expect(arch.ZipArtist(context.Background(), "1", "mp3", 128, out)).To(Succeed())
			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).To(BeNil())
			names := make([]string, len(zr.File))
			for i, f := range zr.File {
				names[i] = f.Name
			}
			return names
		}

		It("keeps the albums in query order", func() {
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "3", Album: "Album C"},
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album A"},
				{Path: "a/02.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album A"},
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Album B"},
			})
			Expect(names).To(Equal([]string{"Album C/01.mp3", "Album A/01.mp3", "Album A/02.mp3", "Album B/01.mp3"}))
		})

		It("suffixes the year when it tells the albums apart", func() {
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01 - Intro.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001},
				{Path: "b/01 - Intro.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 2005},
			})
			Expect(names).To(Equal([]string{"Greatest Hits [2001]/01 - Intro.mp3", "Greatest Hits [2005]/01 - Intro.mp3"}))
		})

		It("prefers the release year, so reissues of the same original are told apart", func() {
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 1996, ReleaseYear: 2001},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 1996, ReleaseYear: 2011},
				{Path: "c/01.mp3", Suffix: "mp3", AlbumID: "3", Album: "Greatest Hits", Year: 1996},
			})
			Expect(names).To(Equal([]string{"Greatest Hits [2001]/01.mp3", "Greatest Hits [2011]/01.mp3", "Greatest Hits [1996]/01.mp3"}))
		})

		It("names the folder after the full album name", func() {
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001,
					Tags: model.Tags{model.TagAlbumVersion: {"Original"}}},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 2005,
					Tags: model.Tags{model.TagAlbumVersion: {"CD/Digital"}}},
			})
			Expect(names).To(Equal([]string{"Greatest Hits (Original)/01.mp3", "Greatest Hits (CD_Digital)/01.mp3"}))
		})

		It("prefers the album version over the year when it is not part of the name", func() {
			conf.Server.Subsonic.AppendAlbumVersion = false
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001,
					Tags: model.Tags{model.TagAlbumVersion: {"Original"}}},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 2005,
					Tags: model.Tags{model.TagAlbumVersion: {"Deluxe Edition"}}},
			})
			Expect(names).To(Equal([]string{"Greatest Hits [Original]/01.mp3", "Greatest Hits [Deluxe Edition]/01.mp3"}))
		})

		It("leaves the one album without the field unsuffixed", func() {
			conf.Server.Subsonic.AppendAlbumVersion = false
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 2005,
					Tags: model.Tags{model.TagAlbumVersion: {"Deluxe Edition"}}},
			})
			Expect(names).To(Equal([]string{"Greatest Hits/01.mp3", "Greatest Hits [Deluxe Edition]/01.mp3"}))
		})

		It("skips a field that is empty on more than one album", func() {
			conf.Server.Subsonic.AppendAlbumVersion = false
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 2005},
				{Path: "c/01.mp3", Suffix: "mp3", AlbumID: "3", Album: "Greatest Hits", Year: 2010,
					Tags: model.Tags{model.TagAlbumVersion: {"Deluxe Edition"}}},
			})
			Expect(names).To(Equal([]string{"Greatest Hits [2001]/01.mp3", "Greatest Hits [2005]/01.mp3", "Greatest Hits [2010]/01.mp3"}))
		})

		It("skips a field that is the same on every album", func() {
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Live", Year: 2001, MbzAlbumType: "album", CatalogNum: "CAT-1"},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Live", Year: 2001, MbzAlbumType: "album", CatalogNum: "CAT-2"},
			})
			Expect(names).To(Equal([]string{"Live [CAT-1]/01.mp3", "Live [CAT-2]/01.mp3"}))
		})

		It("falls back to the album id when nothing differs", func() {
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "0123456789abcdef", Album: "Greatest Hits", Year: 2001},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "fedcba9876543210", Album: "Greatest Hits", Year: 2001},
			})
			Expect(names).To(Equal([]string{"Greatest Hits [012345]/01.mp3", "Greatest Hits [fedcba]/01.mp3"}))
		})

		It("treats names that sanitize to the same folder as a clash", func() {
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "A/B", Year: 2001},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: `A\B`, Year: 2005},
			})
			Expect(names).To(Equal([]string{"A_B [2001]/01.mp3", "A_B [2005]/01.mp3"}))
		})

		It("sanitizes the suffix", func() {
			conf.Server.Subsonic.AppendAlbumVersion = false
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Hits", Tags: model.Tags{model.TagAlbumVersion: {"Vinyl"}}},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Hits", Tags: model.Tags{model.TagAlbumVersion: {"CD/Digital"}}},
			})
			Expect(names).To(Equal([]string{"Hits [Vinyl]/01.mp3", "Hits [CD_Digital]/01.mp3"}))
		})

		It("leaves the folder name alone when only one album has it", func() {
			names := zipArtistEntries(model.MediaFiles{
				{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001},
				{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Other", Year: 2005},
			})
			Expect(names).To(Equal([]string{"Greatest Hits/01.mp3", "Other/01.mp3"}))
		})
	})

	Context("when the transcode limiter rejects a file", func() {
		It("aborts the archive instead of continuing with empty entries", func() {
			mfs := model.MediaFiles{
				{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album", DiscNumber: 1},
				{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album", DiscNumber: 1},
			}

			mfRepo := &mockMediaFileRepository{}
			mfRepo.On("GetAll", []model.QueryOptions{{
				Filters: squirrel.Eq{"album_id": "1"},
				Sort:    "album",
			}}).Return(mfs, nil)
			ds.On("MediaFile", mock.Anything).Return(mfRepo)

			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).
				Return(nil, stream.ErrTooManyTranscodes).Once()

			out := new(bytes.Buffer)
			err := arch.ZipAlbum(context.Background(), "1", "mp3", 128, out)
			Expect(err).To(MatchError(stream.ErrTooManyTranscodes))
			// NewStream should only have been called once: the loop must bail
			// out on the rejection instead of trying every remaining track.
			ms.AssertNumberOfCalls(GinkgoT(), "NewStream", 1)
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

			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).Return(io.NopCloser(strings.NewReader("test")), nil).Times(2)

			out := new(bytes.Buffer)
			err := arch.ZipShare(context.Background(), share, out)
			Expect(err).To(BeNil())

			// Share.Load records a visit; re-loading here would double-count
			// every download.
			sh.AssertNotCalled(GinkgoT(), "Load", mock.Anything, mock.Anything)

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
			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).Return(io.NopCloser(strings.NewReader("test")), nil).Times(2)

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
	stream.MediaStreamer
}

func (m *mockMediaStreamer) NewStream(ctx context.Context, mf *model.MediaFile, req stream.Request) (*stream.Stream, error) {
	args := m.Called(ctx, mf, req)
	if args.Error(1) != nil {
		return nil, args.Error(1)
	}
	return &stream.Stream{ReadCloser: args.Get(0).(io.ReadCloser)}, nil
}

type mockShare struct {
	mock.Mock
	core.Share
}

func (m *mockShare) Load(ctx context.Context, id string) (*model.Share, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Share), args.Error(1)
}
