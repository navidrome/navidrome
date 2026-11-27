package core_test

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/stream"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
)

var _ = Describe("Archiver", func() {
	var (
		ctx  context.Context
		arch core.Archiver
		ms   *mockMediaStreamer
		dc   *fakeDecider
		ds   model.DataStore
		sh   *mockShare
		ca   *mockCoverArt
	)

	putTracks := func(mfs ...model.MediaFile) []string {
		GinkgoHelper()
		return slice.Map(persistencetest.PutMediaFiles(ctx, ds, mfs...), func(mf model.MediaFile) string { return mf.ID })
	}

	byArtist := func(id string, mfs ...model.MediaFile) []model.MediaFile {
		for i := range mfs {
			mfs[i].Participants = model.Participants{model.RoleAlbumArtist: {{ID: id, Name: "Artist " + id}}}
		}
		return mfs
	}

	putPlaylist := func(id, name string, tracks ...model.MediaFile) {
		GinkgoHelper()
		ids := putTracks(tracks...)
		Expect(ds.Playlist().Put(ctx, &model.Playlist{ID: id, Name: name, OwnerID: "user-1"})).To(Succeed())
		_, err := ds.Playlist().Tracks(ctx, id, false).Add(ctx, ids)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		user := model.User{ID: "user-1", UserName: "user-1"}
		Expect(ds.User().Put(ctx, &user)).To(Succeed())
		ctx = request.WithUser(ctx, user)
		for _, id := range []string{"1", "2"} {
			Expect(ds.Artist().Put(ctx, &model.Artist{ID: id, Name: "Artist " + id})).To(Succeed())
		}
		ms = &mockMediaStreamer{}
		dc = &fakeDecider{}
		sh = &mockShare{}
		ca = &mockCoverArt{images: map[string][]byte{}}
		arch = core.NewArchiver(ms, dc, ds, sh, ca)
	})

	Context("ZipAlbum", func() {
		It("zips an album correctly", func() {
			putTracks(
				model.MediaFile{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album/Promo", DiscNumber: 1, TrackNumber: 2},
				model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album/Promo", DiscNumber: 1, TrackNumber: 1},
				model.MediaFile{Path: "other/01 - other.mp3", Suffix: "mp3", AlbumID: "2", Album: "Other", DiscNumber: 1},
			)
			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).Return(io.NopCloser(strings.NewReader("test")), nil).Times(3)

			out := new(bytes.Buffer)
			err := arch.ZipAlbum(ctx, "1", "mp3", 128, out)
			Expect(err).ToNot(HaveOccurred())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).ToNot(HaveOccurred())

			Expect(zr.File).To(HaveLen(2))
			Expect(zr.File[0].Name).To(Equal("Album_Promo/01 - track1.mp3"))
			Expect(zr.File[1].Name).To(Equal("Album_Promo/02 - track2.mp3"))
		})

		It("streams the request resolved by the transcode decider and names the entry after its format", func() {
			putTracks(model.MediaFile{Path: "test_data/01 - track1.flac", Suffix: "flac", AlbumID: "1"})
			resolved := stream.Request{Format: "opus", BitRate: 128, SampleRate: 48000, Channels: 2}
			dc.resolved = &resolved
			ms.On("NewStream", mock.Anything, mock.Anything, resolved).Return(io.NopCloser(strings.NewReader("test")), nil).Once()

			out := new(bytes.Buffer)
			Expect(arch.ZipAlbum(ctx, "1", "mp3", 128, out)).To(Succeed())
			ms.AssertExpectations(GinkgoT())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).ToNot(HaveOccurred())
			Expect(zr.File[0].Name).To(HaveSuffix("01 - track1.opus"))
		})
	})

	Context("ZipArtist", func() {
		It("zips an artist's albums correctly", func() {
			putTracks(byArtist("1",
				model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1, TrackNumber: 1},
				model.MediaFile{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1, TrackNumber: 2},
				model.MediaFile{Path: "test_data/03 - missing.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1, TrackNumber: 3, Missing: true},
			)...)
			putTracks(byArtist("2",
				model.MediaFile{Path: "other/01 - other.mp3", Suffix: "mp3", AlbumID: "2", Album: "Other", DiscNumber: 1},
			)...)
			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).Return(io.NopCloser(strings.NewReader("test")), nil).Times(2)

			out := new(bytes.Buffer)
			err := arch.ZipArtist(ctx, "1", "mp3", 128, out)
			Expect(err).ToNot(HaveOccurred())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).ToNot(HaveOccurred())

			Expect(zr.File).To(HaveLen(2))
			Expect(zr.File[0].Name).To(Equal("Album 1/01 - track1.mp3"))
			Expect(zr.File[1].Name).To(Equal("Album 1/02 - track2.mp3"))
		})

		When("albums that share a name", func() {

			// zipArtistEntries zips the given tracks as artist "1" and returns the entry names in zip order.
			zipArtistEntries := func(mfs model.MediaFiles) []string {
				putTracks(byArtist("1", mfs...)...)
				ms.On("NewStream", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader("test")), nil)

				out := new(bytes.Buffer)
				Expect(arch.ZipArtist(ctx, "1", "mp3", 128, out)).To(Succeed())
				zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
				Expect(err).ToNot(HaveOccurred())
				names := make([]string, len(zr.File))
				for i, f := range zr.File {
					names[i] = f.Name
				}
				return names
			}

			It("keeps the albums in query order", func() {
				names := zipArtistEntries(model.MediaFiles{
					{Path: "c/01.mp3", Suffix: "mp3", AlbumID: "3", Album: "Album C", OrderAlbumName: "1"},
					{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album A", OrderAlbumName: "2", TrackNumber: 1},
					{Path: "a/02.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album A", OrderAlbumName: "2", TrackNumber: 2},
					{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Album B", OrderAlbumName: "3"},
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
				conf.Update(func(c *conf.Options) { c.Subsonic.AppendAlbumVersion = false })
				names := zipArtistEntries(model.MediaFiles{
					{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001,
						Tags: model.Tags{model.TagAlbumVersion: {"Original"}}},
					{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 2005,
						Tags: model.Tags{model.TagAlbumVersion: {"Deluxe Edition"}}},
				})
				Expect(names).To(Equal([]string{"Greatest Hits [Original]/01.mp3", "Greatest Hits [Deluxe Edition]/01.mp3"}))
			})

			It("leaves the one album without the field unsuffixed", func() {
				conf.Update(func(c *conf.Options) { c.Subsonic.AppendAlbumVersion = false })
				names := zipArtistEntries(model.MediaFiles{
					{Path: "a/01.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001},
					{Path: "b/01.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 2005,
						Tags: model.Tags{model.TagAlbumVersion: {"Deluxe Edition"}}},
				})
				Expect(names).To(Equal([]string{"Greatest Hits/01.mp3", "Greatest Hits [Deluxe Edition]/01.mp3"}))
			})

			It("skips a field that is empty on more than one album", func() {
				conf.Update(func(c *conf.Options) { c.Subsonic.AppendAlbumVersion = false })
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
				conf.Update(func(c *conf.Options) { c.Subsonic.AppendAlbumVersion = false })
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
	})

	Context("when the transcode limiter rejects a file", func() {
		It("aborts the archive instead of continuing with empty entries", func() {
			putTracks(
				model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album", DiscNumber: 1, TrackNumber: 1},
				model.MediaFile{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album", DiscNumber: 1, TrackNumber: 2},
			)

			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).
				Return(nil, stream.ErrTooManyTranscodes).Once()

			out := new(bytes.Buffer)
			err := arch.ZipAlbum(ctx, "1", "mp3", 128, out)
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
			err := arch.ZipShare(ctx, share, out)
			Expect(err).ToNot(HaveOccurred())

			// Share.Load records a visit; re-loading here would double-count
			// every download.
			sh.AssertNotCalled(GinkgoT(), "Load", mock.Anything, mock.Anything)

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).ToNot(HaveOccurred())

			Expect(zr.File).To(HaveLen(2))
			Expect(zr.File[0].Name).To(Equal("01 - Artist 1 - track1.mp3"))
			Expect(zr.File[1].Name).To(Equal("02 - Artist 2 - track2.mp3"))

		})
	})

	Context("ZipPlaylist", func() {
		It("names the entries and the M3U lines after the resolved format", func() {
			putPlaylist("1", "Test Playlist",
				model.MediaFile{Path: "test_data/01 - track1.flac", Suffix: "flac", Artist: "Artist 1", Title: "track1"},
			)
			dc.resolved = &stream.Request{Format: "opus", BitRate: 128}
			ms.On("NewStream", mock.Anything, mock.Anything, *dc.resolved).Return(io.NopCloser(strings.NewReader("test")), nil)

			out := new(bytes.Buffer)
			Expect(arch.ZipPlaylist(ctx, "1", "mp3", 128, out)).To(Succeed())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).ToNot(HaveOccurred())
			Expect(zr.File[0].Name).To(Equal("01 - Artist 1 - track1.opus"))
			m3u, err := zr.File[1].Open()
			Expect(err).ToNot(HaveOccurred())
			defer m3u.Close()
			content, err := io.ReadAll(m3u)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("01 - Artist 1 - track1.opus"))
		})

		It("zips a playlist correctly", func() {
			putPlaylist("1", "Test Playlist",
				model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1, Artist: "AC/DC", Title: "track1"},
				model.MediaFile{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1, Artist: "Artist 2", Title: "track2"},
			)
			ms.On("NewStream", mock.Anything, mock.Anything, stream.Request{Format: "mp3", BitRate: 128}).Return(io.NopCloser(strings.NewReader("test")), nil).Times(2)

			out := new(bytes.Buffer)
			err := arch.ZipPlaylist(ctx, "1", "mp3", 128, out)
			Expect(err).ToNot(HaveOccurred())

			zr, err := zip.NewReader(bytes.NewReader(out.Bytes()), int64(out.Len()))
			Expect(err).ToNot(HaveOccurred())

			Expect(zr.File).To(HaveLen(3))
			Expect(zr.File[0].Name).To(Equal("01 - AC_DC - track1.mp3"))
			Expect(zr.File[1].Name).To(Equal("02 - Artist 2 - track2.mp3"))
			Expect(zr.File[2].Name).To(Equal("Test Playlist.m3u"))

			// Verify M3U content
			m3uFile, err := zr.File[2].Open()
			Expect(err).ToNot(HaveOccurred())
			defer m3uFile.Close()

			m3uContent, err := io.ReadAll(m3uFile)
			Expect(err).ToNot(HaveOccurred())

			expectedM3U := "#EXTM3U\n#PLAYLIST:Test Playlist\n#EXTINF:0,AC/DC - track1\n01 - AC_DC - track1.mp3\n#EXTINF:0,Artist 2 - track2\n02 - Artist 2 - track2.mp3\n"
			Expect(string(m3uContent)).To(Equal(expectedM3U))
		})
	})
	Context("cover art", func() {
		var (
			jpegData = []byte("\xff\xd8\xff\xe0 fake jpeg")
			pngData  = []byte("\x89PNG\x0d\x0a\x1a\x0a fake png")
		)

		putAlbumTracks := func(mfs ...model.MediaFile) {
			GinkgoHelper()
			putTracks(mfs...)
			ms.On("NewStream", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader("test")), nil)
		}

		It("adds the album cover to the album folder", func() {
			ca.images["al-1"] = jpegData
			putAlbumTracks(model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album/Promo", DiscNumber: 1})

			out := new(bytes.Buffer)
			Expect(arch.ZipAlbum(ctx, "1", "mp3", 128, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveLen(2))
			Expect(files).To(HaveKeyWithValue("Album_Promo/folder.jpg", jpegData))
			Expect(ca.requests).To(ConsistOf(coverRequest{id: "al-1", size: 500, square: false}))
		})

		It("adds the artist image to the root and each album cover to its folder", func() {
			ca.images["ar-1"] = pngData
			ca.images["al-1"] = jpegData
			ca.images["al-2"] = jpegData
			putAlbumTracks(byArtist("1",
				model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album 1", DiscNumber: 1},
				model.MediaFile{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "2", Album: "Album 2", DiscNumber: 1},
			)...)

			out := new(bytes.Buffer)
			Expect(arch.ZipArtist(ctx, "1", "mp3", 128, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveLen(5))
			Expect(files).To(HaveKeyWithValue("folder.png", pngData))
			Expect(files).To(HaveKeyWithValue("Album 1/folder.jpg", jpegData))
			Expect(files).To(HaveKeyWithValue("Album 2/folder.jpg", jpegData))
		})

		It("puts each same-named album's cover in that album's own folder", func() {
			ca.images["al-1"] = jpegData
			ca.images["al-2"] = pngData
			putAlbumTracks(byArtist("1",
				model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Greatest Hits", Year: 2001, DiscNumber: 1},
				model.MediaFile{Path: "test_data/02 - track2.mp3", Suffix: "mp3", AlbumID: "2", Album: "Greatest Hits", Year: 2005, DiscNumber: 1},
			)...)

			out := new(bytes.Buffer)
			Expect(arch.ZipArtist(ctx, "1", "mp3", 128, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveKeyWithValue("Greatest Hits [2001]/folder.jpg", jpegData))
			Expect(files).To(HaveKeyWithValue("Greatest Hits [2005]/folder.png", pngData))
		})

		It("adds the playlist cover to the root", func() {
			ca.images["pl-1"] = jpegData
			putPlaylist("1", "Test Playlist",
				model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Artist: "Artist 1", Title: "track1"},
			)
			ms.On("NewStream", mock.Anything, mock.Anything, mock.Anything).Return(io.NopCloser(strings.NewReader("test")), nil)

			out := new(bytes.Buffer)
			Expect(arch.ZipPlaylist(ctx, "1", "mp3", 128, out)).To(Succeed())

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
			Expect(arch.ZipShare(ctx, share, out)).To(Succeed())

			files := readZip(out)
			Expect(files).To(HaveLen(2))
			Expect(files).To(HaveKeyWithValue("folder.jpg", jpegData))
			Expect(ca.requests).To(ConsistOf(coverRequest{id: "pl-10", size: 500, square: false, admin: true}))
		})

		It("still builds the archive when the cover cannot be read", func() {
			ca.err = errors.New("boom")
			putAlbumTracks(model.MediaFile{Path: "test_data/01 - track1.mp3", Suffix: "mp3", AlbumID: "1", Album: "Album", DiscNumber: 1})

			out := new(bytes.Buffer)
			Expect(arch.ZipAlbum(ctx, "1", "mp3", 128, out)).To(Succeed())

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

// fakeDecider echoes the legacy format/bitrate unless a resolved request is set.
type fakeDecider struct {
	stream.TranscodeDecider
	resolved *stream.Request
}

func (f *fakeDecider) ResolveRequest(_ context.Context, _ *model.MediaFile, format string, bitRate int, offset int) stream.Request {
	if f.resolved != nil {
		return *f.resolved
	}
	return stream.Request{Format: format, BitRate: bitRate, Offset: offset}
}

type mockShare struct {
	mock.Mock
	core.Share
}

func (m *mockShare) Load(ctx context.Context, id string) (*model.Share, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*model.Share), args.Error(1)
}
