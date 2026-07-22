package artwork

import (
	"context"
	"errors"
	"image"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// fakeExternalProvider is a minimal external.Provider stub for resolve_test.go;
// only AlbumImage/ArtistImage are exercised by the resolvers.
type fakeExternalProvider struct {
	external.Provider
	albumImage  func(ctx context.Context, id string) (*url.URL, error)
	artistImage func(ctx context.Context, id string) (*url.URL, error)
}

func (f *fakeExternalProvider) AlbumImage(ctx context.Context, id string) (*url.URL, error) {
	if f.albumImage != nil {
		return f.albumImage(ctx, id)
	}
	return nil, model.ErrNotFound
}

func (f *fakeExternalProvider) ArtistImage(ctx context.Context, id string) (*url.URL, error) {
	if f.artistImage != nil {
		return f.artistImage(ctx, id)
	}
	return nil, model.ErrNotFound
}

func (f *fakeExternalProvider) ArtistImageResult(ctx context.Context, id string) (*url.URL, error) {
	return f.ArtistImage(ctx, id)
}

var _ = Describe("resolveItem", func() {
	var (
		ctx        context.Context
		ds         *tests.MockDataStore
		folderRepo *fakeFolderRepo
		libRepo    *tests.MockLibraryRepo
		ffm        *tests.MockFFmpeg
		prov       *fakeExternalProvider
		repoRoot   string
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = context.Background()
		var err error
		repoRoot, err = os.Getwd()
		Expect(err).ToNot(HaveOccurred())

		folderRepo = &fakeFolderRepo{}
		libRepo = &tests.MockLibraryRepo{}
		libRepo.SetData(model.Libraries{{ID: 0, Path: testFileLibPath(repoRoot)}})
		ffm = tests.NewMockFFmpeg("")
		prov = &fakeExternalProvider{}
		ds = &tests.MockDataStore{
			MockedFolder:  folderRepo,
			MockedLibrary: libRepo,
		}
	})

	Describe("kind dispatch", func() {
		It("returns an error for kinds the worker never enqueues", func() {
			_, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "mf", ItemID: "x"}, nil)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("album", func() {
		BeforeEach(func() {
			conf.Server.CoverArtPriority = "cover.jpg, embedded"
			ds.MockedAlbum = tests.CreateMockAlbumRepo()
		})

		It("resolves folder art from the library FS", func() {
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "al1", Name: "Album", FolderIDs: []string{"f1"}},
			})

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al1"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).ToNot(BeNil())
			defer res.reader.Close()
			Expect(res.source).To(Equal("folder"))
			Expect(filepath.ToSlash(res.sourcePath)).To(HaveSuffix("tests/fixtures/artist/an-album/cover.jpg"))
			Expect(res.refMtime).To(BeNumerically(">", 0))
			Expect(res.extError).To(BeFalse())
		})

		It("falls back to embedded art when no folder image matches", func() {
			folderRepo.result = nil
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "al2", Name: "Album", EmbedArtPath: "tests/fixtures/artist/an-album/test.mp3", FolderIDs: []string{"f1"}},
			})

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al2"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).ToNot(BeNil())
			defer res.reader.Close()
			Expect(res.source).To(Equal("embedded"))
			Expect(filepath.ToSlash(res.sourcePath)).To(HaveSuffix("tests/fixtures/artist/an-album/test.mp3"))
			Expect(res.refMtime).To(BeNumerically(">", 0))
		})

		It("sets extError when the external source errors without being not-found", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "al3", Name: "Album"},
			})
			prov.albumImage = func(context.Context, string) (*url.URL, error) {
				return nil, errors.New("agent timed out")
			}

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al3"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).To(BeNil())
			Expect(res.extError).To(BeTrue())
		})

		It("does not set extError when the external source reports not-found", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "al4", Name: "Album"},
			})
			// prov.albumImage left nil -> fakeExternalProvider returns model.ErrNotFound

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al4"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).To(BeNil())
			Expect(res.extError).To(BeFalse())
		})

		It("routes the external step through a custom extGate", func() {
			conf.Server.CoverArtPriority = "external"
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "al5", Name: "Album"},
			})
			prov.albumImage = func(context.Context, string) (*url.URL, error) {
				return nil, errors.New("boom")
			}
			var extGateCalls int
			extGate := func(f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
				extGateCalls++
				return f()
			}

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "al", ItemID: "al5"}, extGate)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.extError).To(BeTrue())
			Expect(extGateCalls).To(Equal(1))
		})
	})

	Describe("artist", func() {
		It("resolves the uploaded image before any priority chain lookup", func() {
			tmpDir := GinkgoT().TempDir()
			conf.Server.DataFolder = conf.NewDir(tmpDir)
			Expect(os.MkdirAll(filepath.Join(tmpDir, "artwork", "artist"), 0755)).To(Succeed())
			imgPath := filepath.Join(tmpDir, "artwork", "artist", "ar1_test.jpg")
			Expect(os.WriteFile(imgPath, []byte("uploaded artist image"), 0600)).To(Succeed())

			artistRepo := tests.CreateMockArtistRepo()
			artistRepo.SetData(model.Artists{{ID: "ar1", Name: "Artist", UploadedImage: "ar1_test.jpg"}})
			ds.MockedArtist = artistRepo

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar1"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).ToNot(BeNil())
			defer res.reader.Close()
			Expect(res.source).To(Equal("upload"))
			Expect(res.sourcePath).To(Equal(imgPath))
		})

		It("falls through to the ArtistArtPriority chain when there is no upload", func() {
			conf.Server.ArtistArtPriority = "album/artist.*"
			folderRepo.result = []model.Folder{{
				LibraryPath: testFileLibPath(repoRoot),
				Path:        "tests/fixtures/artist/an-album",
				ImageFiles:  []string{"artist.png"},
			}}
			artistRepo := tests.CreateMockArtistRepo()
			artistRepo.SetData(model.Artists{{ID: "ar2", Name: "Artist"}})
			ds.MockedArtist = artistRepo
			ds.MockedAlbum = tests.CreateMockAlbumRepo()
			ds.MockedAlbum.(*tests.MockAlbumRepo).All = model.Albums{
				{ID: "al9", Name: "Album", LibraryID: 0, FolderIDs: []string{"f1"}},
			}

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar2"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).ToNot(BeNil())
			defer res.reader.Close()
			Expect(res.source).To(Equal("folder"))
			Expect(filepath.ToSlash(res.sourcePath)).To(HaveSuffix("tests/fixtures/artist/an-album/artist.png"))
		})

		It("sets extError when the external source errors without being not-found", func() {
			conf.Server.ArtistArtPriority = "external"
			artistRepo := tests.CreateMockArtistRepo()
			artistRepo.SetData(model.Artists{{ID: "ar3", Name: "Artist"}})
			ds.MockedArtist = artistRepo
			prov.artistImage = func(context.Context, string) (*url.URL, error) {
				return nil, errors.New("agent timed out")
			}

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar3"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).To(BeNil())
			Expect(res.extError).To(BeTrue())
		})

		It("does not set extError when the external source reports not-found", func() {
			conf.Server.ArtistArtPriority = "external"
			artistRepo := tests.CreateMockArtistRepo()
			artistRepo.SetData(model.Artists{{ID: "ar4", Name: "Artist"}})
			ds.MockedArtist = artistRepo
			// prov.artistImage left nil -> fakeExternalProvider returns model.ErrNotFound

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar4"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).To(BeNil())
			Expect(res.extError).To(BeFalse())
		})

		It("routes the external step through a custom extGate", func() {
			conf.Server.ArtistArtPriority = "external"
			artistRepo := tests.CreateMockArtistRepo()
			artistRepo.SetData(model.Artists{{ID: "ar5", Name: "Artist"}})
			ds.MockedArtist = artistRepo
			prov.artistImage = func(context.Context, string) (*url.URL, error) {
				return nil, errors.New("boom")
			}
			var extGateCalls int
			extGate := func(f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
				extGateCalls++
				return f()
			}

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "ar", ItemID: "ar5"}, extGate)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.extError).To(BeTrue())
			Expect(extGateCalls).To(Equal(1))
		})
	})

	Describe("radio", func() {
		It("yields an empty resolution when there is no uploaded image", func() {
			tmpDir := GinkgoT().TempDir()
			conf.Server.DataFolder = conf.NewDir(tmpDir)

			radioRepo := tests.CreateMockedRadioRepo()
			radioRepo.Data = map[string]*model.Radio{"ra1": {ID: "ra1", Name: "Radio"}}
			ds.MockedRadio = radioRepo

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "ra", ItemID: "ra1"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res).To(Equal(resolution{}))
		})

		It("resolves the uploaded image when set", func() {
			tmpDir := GinkgoT().TempDir()
			conf.Server.DataFolder = conf.NewDir(tmpDir)
			Expect(os.MkdirAll(filepath.Join(tmpDir, "artwork", "radio"), 0755)).To(Succeed())
			imgPath := filepath.Join(tmpDir, "artwork", "radio", "ra2_test.jpg")
			Expect(os.WriteFile(imgPath, []byte("uploaded radio image"), 0600)).To(Succeed())

			radioRepo := tests.CreateMockedRadioRepo()
			radioRepo.Data = map[string]*model.Radio{"ra2": {ID: "ra2", Name: "Radio", UploadedImage: "ra2_test.jpg"}}
			ds.MockedRadio = radioRepo

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "ra", ItemID: "ra2"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).ToNot(BeNil())
			defer res.reader.Close()
			Expect(res.source).To(Equal("upload"))
			Expect(res.sourcePath).To(Equal(imgPath))
		})
	})

	Describe("playlist", func() {
		BeforeEach(func() {
			conf.Server.CoverArtPriority = "cover.jpg"
			folderRepo.result = []model.Folder{{
				Path:       "tests/fixtures/artist/an-album",
				ImageFiles: []string{"cover.jpg"},
			}}
			ds.MockedAlbum = tests.CreateMockAlbumRepo()
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "t1", Name: "T1", FolderIDs: []string{"f1"}},
				{ID: "t2", Name: "T2", FolderIDs: []string{"f1"}},
				{ID: "t3", Name: "T3", FolderIDs: []string{"f1"}},
				{ID: "t4", Name: "T4", FolderIDs: []string{"f1"}},
			})
		})

		DescribeTable("yields a generated grid from up to 4 album tiles",
			func(albumIDs []string, expectedSize int) {
				plRepo := tests.CreateMockPlaylistRepo()
				plRepo.SetData(model.Playlists{{ID: "pl1", Name: "Playlist"}})
				plRepo.TracksRepo = &tests.MockPlaylistTrackRepo{AlbumIDs: albumIDs}
				ds.MockedPlaylist = plRepo

				res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "pl", ItemID: "pl1"}, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.reader).ToNot(BeNil())
				defer res.reader.Close()
				Expect(res.source).To(Equal("generated"))

				img, format, err := image.Decode(res.reader)
				Expect(err).ToNot(HaveOccurred())
				Expect(format).To(Equal("png"))
				Expect(img.Bounds().Dx()).To(Equal(expectedSize))
				Expect(img.Bounds().Dy()).To(Equal(expectedSize))
			},
			// tileSize-1: the 4-tile canvas is built as [0, tileSize-1], matching
			// reader_playlist.go's createTiledImage exactly.
			Entry("1 album -> single tile", []string{"t1"}, tileSize/2),
			Entry("2 albums -> duplicated to 4 tiles", []string{"t1", "t2"}, tileSize-1),
			Entry("3 albums -> duplicated to 4 tiles", []string{"t1", "t2", "t3"}, tileSize-1),
			Entry("4 albums -> full grid", []string{"t1", "t2", "t3", "t4"}, tileSize-1),
		)

		It("resolves the uploaded image before the generated grid", func() {
			tmpDir := GinkgoT().TempDir()
			conf.Server.DataFolder = conf.NewDir(tmpDir)
			Expect(os.MkdirAll(filepath.Join(tmpDir, "artwork", "playlist"), 0755)).To(Succeed())
			imgPath := filepath.Join(tmpDir, "artwork", "playlist", "plu_test.jpg")
			Expect(os.WriteFile(imgPath, []byte("uploaded playlist image"), 0600)).To(Succeed())

			plRepo := tests.CreateMockPlaylistRepo()
			plRepo.SetData(model.Playlists{{ID: "plu", Name: "Playlist", UploadedImage: "plu_test.jpg"}})
			plRepo.TracksRepo = &tests.MockPlaylistTrackRepo{AlbumIDs: []string{"t1", "t2"}}
			ds.MockedPlaylist = plRepo

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "pl", ItemID: "plu"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).ToNot(BeNil())
			defer res.reader.Close()
			Expect(res.source).To(Equal("upload"))
			Expect(res.sourcePath).To(Equal(imgPath))
		})

		It("resolves a sidecar image next to the playlist file before the grid", func() {
			plDir := GinkgoT().TempDir()
			Expect(os.WriteFile(filepath.Join(plDir, "list.m3u"), []byte("#EXTM3U"), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(plDir, "list.jpg"), []byte("sidecar image"), 0600)).To(Succeed())

			plRepo := tests.CreateMockPlaylistRepo()
			plRepo.SetData(model.Playlists{{ID: "pls", Name: "Playlist", Path: filepath.Join(plDir, "list.m3u")}})
			plRepo.TracksRepo = &tests.MockPlaylistTrackRepo{AlbumIDs: []string{"t1", "t2"}}
			ds.MockedPlaylist = plRepo

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "pl", ItemID: "pls"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).ToNot(BeNil())
			defer res.reader.Close()
			Expect(res.source).To(Equal("folder"))
			Expect(filepath.ToSlash(res.sourcePath)).To(HaveSuffix("list.jpg"))
		})

		It("routes ExternalImageURL through extGate and sets extError on transient failure", func() {
			conf.Server.EnableM3UExternalAlbumArt = true
			folderRepo.result = nil // no grid tiles, so the external failure is what surfaces

			plRepo := tests.CreateMockPlaylistRepo()
			plRepo.SetData(model.Playlists{{ID: "ple", Name: "Playlist", ExternalImageURL: "http://example.com/cover.jpg"}})
			plRepo.TracksRepo = &tests.MockPlaylistTrackRepo{AlbumIDs: []string{"t1"}}
			ds.MockedPlaylist = plRepo

			var extGateCalls int
			extGate := func(f func() (io.ReadCloser, string, error)) (io.ReadCloser, string, error) {
				extGateCalls++
				return nil, "", errors.New("network down")
			}

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "pl", ItemID: "ple"}, extGate)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).To(BeNil())
			Expect(res.extError).To(BeTrue())
			Expect(extGateCalls).To(Equal(1))
		})

		It("treats a missing local ExternalImageURL as a definitive miss, not extError", func() {
			folderRepo.result = nil // no grid tiles, so the local-file miss is what surfaces

			plRepo := tests.CreateMockPlaylistRepo()
			plRepo.SetData(model.Playlists{{ID: "plm", Name: "Playlist", ExternalImageURL: "/nonexistent/path/cover.jpg"}})
			plRepo.TracksRepo = &tests.MockPlaylistTrackRepo{AlbumIDs: []string{"t1"}}
			ds.MockedPlaylist = plRepo

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "pl", ItemID: "plm"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).To(BeNil())
			Expect(res.extError).To(BeFalse())
		})

		It("yields an empty resolution when no album has art", func() {
			ds.MockedAlbum.(*tests.MockAlbumRepo).SetData(model.Albums{
				{ID: "empty1", Name: "Empty"},
			})
			plRepo := tests.CreateMockPlaylistRepo()
			plRepo.SetData(model.Playlists{{ID: "pl2", Name: "Playlist"}})
			plRepo.TracksRepo = &tests.MockPlaylistTrackRepo{AlbumIDs: []string{"empty1"}}
			ds.MockedPlaylist = plRepo
			folderRepo.result = nil

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "pl", ItemID: "pl2"}, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(res.reader).To(BeNil())
			Expect(res.source).To(BeEmpty())
		})

		It("does not resolve as absent when every sampled album fails to resolve", func() {
			// "missing1"/"missing2" are not in MockAlbumRepo's data, so resolveAlbum
			// returns a genuine (non-external) error for every sampled tile.
			plRepo := tests.CreateMockPlaylistRepo()
			plRepo.SetData(model.Playlists{{ID: "pl3", Name: "Playlist"}})
			plRepo.TracksRepo = &tests.MockPlaylistTrackRepo{AlbumIDs: []string{"missing1", "missing2"}}
			ds.MockedPlaylist = plRepo

			res, err := resolveItem(ctx, ds, prov, ffm, model.ArtworkQueueItem{ItemKind: "pl", ItemID: "pl3"}, nil)
			Expect(err).To(HaveOccurred())
			Expect(res).To(Equal(resolution{}))
		})
	})
})
