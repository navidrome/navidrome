package core_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
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
		ca   *mockCoverArt
	)

	BeforeEach(func() {
		ms = &mockMediaStreamer{}
		sh = &mockShare{}
		ds = &mockDataStore{}
		ca = &mockCoverArt{images: map[string][]byte{}}
		arch = core.NewArchiver(ms, ds, sh, ca)
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
	Context("cover art", func() {
		var (
			jpegData = []byte("\xff\xd8\xff\xe0 fake jpeg")
			pngData  = []byte("\x89PNG\x0d\x0a\x1a\x0a fake png")
		)

		mockAlbumTracks := func(filter squirrel.Sqlizer, mfs model.MediaFiles) {
			mfRepo := &mockMediaFileRepository{}
			mfRepo.On("GetAll", []model.QueryOptions{{Filters: filter, Sort: "album"}}).Return(mfs, nil)
			ds.On("MediaFile", mock.Anything).Return(mfRepo)
			ms.On("NewStream", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader("test")), nil)
		}

		It("adds the album cover to the album folder", func() {
			ca.images["al-1"] = jpegData
			mockAlbumTracks(squirrel.Eq{"album_id": "1"}, model.MediaFiles{
				{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album/Promo", DiscNumber: 1},
			})

			out := new(bytes.Buffer)
			Expect(arch.ZipAlbum(context.Background(), "1", "mp3", 128, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveLen(2))
			Expect(files).To(HaveKeyWithValue("Album_Promo/folder.jpg", jpegData))
			Expect(ca.requests).To(ConsistOf(coverRequest{id: "al-1", size: 500, square: false}))
		})

		It("adds the artist image to the root and each album cover to its folder", func() {
			ca.images["ar-1"] = pngData
			ca.images["al-1"] = jpegData
			ca.images["al-2"] = jpegData
			mockAlbumTracks(squirrel.And{
				persistence.ParticipantIDFilter("media_file", "1", model.RoleAlbumArtist),
				squirrel.Eq{"missing": false},
			}, model.MediaFiles{
				{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1},
				{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "2", Album: "Album 2", DiscNumber: 1},
			})

			out := new(bytes.Buffer)
			Expect(arch.ZipArtist(context.Background(), "1", "mp3", 128, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveLen(5))
			Expect(files).To(HaveKeyWithValue("folder.png", pngData))
			Expect(files).To(HaveKeyWithValue("Album 1/folder.jpg", jpegData))
			Expect(files).To(HaveKeyWithValue("Album 2/folder.jpg", jpegData))
		})

		It("adds one cover when different albums share a folder name", func() {
			ca.images["al-1"] = jpegData
			ca.images["al-2"] = pngData
			mockAlbumTracks(squirrel.And{
				persistence.ParticipantIDFilter("media_file", "1", model.RoleAlbumArtist),
				squirrel.Eq{"missing": false},
			}, model.MediaFiles{
				{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", DiscNumber: 1},
				{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", DiscNumber: 1},
			})

			out := new(bytes.Buffer)
			Expect(arch.ZipArtist(context.Background(), "1", "mp3", 128, out)).To(Succeed())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).ToNot(HaveOccurred())
			var covers []string
			for _, f := range zr.File {
				if strings.HasPrefix(f.Name, "Greatest Hits/folder.") {
					covers = append(covers, f.Name)
				}
			}
			Expect(covers).To(HaveLen(1))
		})

		It("adds the playlist cover to the root", func() {
			ca.images["pl-1"] = jpegData
			plRepo := &mockPlaylistRepository{}
			plRepo.On("GetWithTracks", "1", true, false).Return(&model.Playlist{
				ID:   "1",
				Name: "Test Playlist",
				Tracks: []model.PlaylistTrack{
					{MediaFile: model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Artist: "Artist 1", Title: "track1"}},
				},
			}, nil)
			ds.On("Playlist", mock.Anything).Return(plRepo)
			ms.On("NewStream", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader("test")), nil)

			out := new(bytes.Buffer)
			Expect(arch.ZipPlaylist(context.Background(), "1", "mp3", 128, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveLen(3))
			Expect(files).To(HaveKeyWithValue("folder.jpg", jpegData))
			Expect(files).To(HaveKey("Test Playlist.m3u"))
		})

		It("adds the shared item's cover to the root, even for a private playlist", func() {
			ca.images["pl-10"] = jpegData
			ms.On("NewStream", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader("test")), nil)
			share := &model.Share{
				ID:           "1",
				Downloadable: true,
				Format:       "mp3",
				MaxBitRate:   128,
				ResourceType: "playlist",
				ResourceIDs:  "10",
				Tracks: model.MediaFiles{
					{ID: "1", Path: "test_data/01 - track1.mp3", Suffix: "mp3", Artist: "Artist 1", Title: "track1"},
				},
			}

			out := new(bytes.Buffer)
			Expect(arch.ZipShare(context.Background(), share, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveLen(2))
			Expect(files).To(HaveKeyWithValue("folder.jpg", jpegData))
			Expect(ca.requests).To(ConsistOf(coverRequest{id: "pl-10", size: 500, square: false, admin: true}))
		})

		It("still builds the archive when the cover cannot be read", func() {
			ca.err = errors.New("boom")
			mockAlbumTracks(squirrel.Eq{"album_id": "1"}, model.MediaFiles{
				{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album", DiscNumber: 1},
			})

			out := new(bytes.Buffer)
			Expect(arch.ZipAlbum(context.Background(), "1", "mp3", 128, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveLen(1))
			Expect(files).To(HaveKey("Album/01 - track1.mp3"))
		})
	})
})

func readZip(out *bytes.Buffer) map[string][]byte {
	zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
	Expect(err).ToNot(HaveOccurred())
	files := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		r, err := f.Open()
		Expect(err).ToNot(HaveOccurred())
		data, err := io.ReadAll(r)
		Expect(err).ToNot(HaveOccurred())
		_ = r.Close()
		files[f.Name] = data
	}
	return files
}

type coverRequest struct {
	id     string
	size   int
	square bool
	admin  bool
}

type mockCoverArt struct {
	artwork.Artwork
	images   map[string][]byte
	err      error
	requests []coverRequest
}

func (m *mockCoverArt) Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (*artwork.Image, error) {
	user, _ := request.UserFrom(ctx)
	m.requests = append(m.requests, coverRequest{id: artID.String(), size: size, square: square, admin: user.IsAdmin})
	if m.err != nil {
		return nil, m.err
	}
	data, ok := m.images[artID.String()]
	if !ok {
		return nil, artwork.ErrUnavailable
	}
	return &artwork.Image{ReadCloser: io.NopCloser(bytes.NewReader(data))}, nil
}

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
