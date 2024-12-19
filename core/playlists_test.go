package core

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Playlists", func() {
	var ds *tests.MockDataStore
	var ps Playlists
	var mp mockedPlaylist
	ctx := context.Background()

	BeforeEach(func() {
		mp = mockedPlaylist{}
		ds = &tests.MockDataStore{
			MockedPlaylist: &mp,
		}
		ctx = request.WithUser(ctx, model.User{ID: "123"})
	})

	Describe("ImportFile", func() {
		var folder *model.Folder
		BeforeEach(func() {
			ps = NewPlaylists(ds)
			ds.MockedMediaFile = &mockedMediaFileRepo{}
			libPath, _ := os.Getwd()
			folder = &model.Folder{
				ID:          "1",
				LibraryID:   1,
				LibraryPath: libPath,
				Path:        "tests/fixtures",
				Name:        "playlists",
			}
		})

		Describe("M3U", func() {
			It("parses well-formed playlists", func() {
				// get absolute path for "tests/fixtures" folder
				pls, err := ps.ImportFile(ctx, folder, "pls1.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Tracks).To(HaveLen(3))
				Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/playlists/test.mp3"))
				Expect(pls.Tracks[1].Path).To(Equal("tests/fixtures/playlists/test.ogg"))
				Expect(pls.Tracks[2].Path).To(Equal("/tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
				Expect(mp.last).To(Equal(pls))
			})

			It("parses playlists using LF ending", func() {
				pls, err := ps.ImportFile(ctx, folder, "lf-ended.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(2))
			})

			It("parses playlists using CR ending (old Mac format)", func() {
				pls, err := ps.ImportFile(ctx, folder, "cr-ended.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(2))
			})
		})

		Describe("NSP", func() {
			It("parses well-formed playlists", func() {
				pls, err := ps.ImportFile(ctx, folder, "recently_played.nsp")
				Expect(err).ToNot(HaveOccurred())
				Expect(mp.last).To(Equal(pls))
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Name).To(Equal("Recently Played"))
				Expect(pls.Comment).To(Equal("Recently played tracks"))
				Expect(pls.Rules.Sort).To(Equal("lastPlayed"))
				Expect(pls.Rules.Order).To(Equal("desc"))
				Expect(pls.Rules.Limit).To(Equal(100))
				Expect(pls.Rules.Expression).To(BeAssignableToTypeOf(criteria.All{}))
			})
			It("returns an error if the playlist is not well-formed", func() {
				_, err := ps.ImportFile(ctx, folder, "invalid_json.nsp")
				Expect(err.Error()).To(ContainSubstring("line 19, column 1: invalid character '\\n'"))
			})
		})
	})

	Describe("ImportM3U", func() {
		var repo *mockedMediaFileFromListRepo
		BeforeEach(func() {
			repo = &mockedMediaFileFromListRepo{}
			ds.MockedMediaFile = repo
			ps = NewPlaylists(ds)
			ctx = request.WithUser(ctx, model.User{ID: "123"})
		})

		It("parses well-formed playlists", func() {
			repo.data = []string{
				"tests/fixtures/test.mp3",
				"tests/fixtures/test.ogg",
				"/tests/fixtures/01 Invisible (RED) Edit Version.mp3",
			}
			f, _ := os.Open("tests/fixtures/playlists/pls-with-name.m3u")
			defer f.Close()
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.OwnerID).To(Equal("123"))
			Expect(pls.Name).To(Equal("playlist 1"))
			Expect(pls.Sync).To(BeFalse())
			Expect(pls.Tracks).To(HaveLen(3))
			Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/test.mp3"))
			Expect(pls.Tracks[1].Path).To(Equal("tests/fixtures/test.ogg"))
			Expect(pls.Tracks[2].Path).To(Equal("/tests/fixtures/01 Invisible (RED) Edit Version.mp3"))
			Expect(mp.last).To(Equal(pls))
			f.Close()

		})

		It("sets the playlist name as a timestamp if the #PLAYLIST directive is not present", func() {
			repo.data = []string{
				"tests/fixtures/test.mp3",
				"tests/fixtures/test.ogg",
				"/tests/fixtures/01 Invisible (RED) Edit Version.mp3",
			}
			f, _ := os.Open("tests/fixtures/playlists/pls-without-name.m3u")
			defer f.Close()
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			_, err = time.Parse(time.RFC3339, pls.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(3))
		})

		It("returns only tracks that exist in the database and in the same other as the m3u", func() {
			repo.data = []string{
				"test1.mp3",
				"test2.mp3",
				"test3.mp3",
			}
			m3u := strings.Join([]string{
				"test3.mp3",
				"test1.mp3",
				"test4.mp3",
				"test2.mp3",
			}, "\n")
			f := strings.NewReader(m3u)
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(3))
			Expect(pls.Tracks[0].Path).To(Equal("test3.mp3"))
			Expect(pls.Tracks[1].Path).To(Equal("test1.mp3"))
			Expect(pls.Tracks[2].Path).To(Equal("test2.mp3"))
		})

		It("is case-insensitive when comparing paths", func() {
			repo.data = []string{
				"tEsT1.Mp3",
			}
			m3u := strings.Join([]string{
				"TeSt1.mP3",
			}, "\n")
			f := strings.NewReader(m3u)
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(1))
			Expect(pls.Tracks[0].Path).To(Equal("tEsT1.Mp3"))
		})
	})

	Describe("InPlaylistsPath", func() {
		var folder model.Folder

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			folder = model.Folder{
				LibraryPath: "/music",
				Path:        "playlists/abc",
				Name:        "folder1",
			}
		})

		It("returns true if PlaylistsPath is empty", func() {
			conf.Server.PlaylistsPath = ""
			Expect(InPlaylistsPath(folder)).To(BeTrue())
		})

		It("returns true if PlaylistsPath is any (**/**)", func() {
			conf.Server.PlaylistsPath = "**/**"
			Expect(InPlaylistsPath(folder)).To(BeTrue())
		})

		It("returns true if folder is in PlaylistsPath", func() {
			conf.Server.PlaylistsPath = "other/**:playlists/**"
			Expect(InPlaylistsPath(folder)).To(BeTrue())
		})

		It("returns false if folder is not in PlaylistsPath", func() {
			conf.Server.PlaylistsPath = "other"
			Expect(InPlaylistsPath(folder)).To(BeFalse())
		})

		It("returns true if for a playlist in root of MusicFolder if PlaylistsPath is '.'", func() {
			conf.Server.PlaylistsPath = "."
			Expect(InPlaylistsPath(folder)).To(BeFalse())

			folder2 := model.Folder{
				LibraryPath: "/music",
				Path:        "",
				Name:        ".",
			}

			Expect(InPlaylistsPath(folder2)).To(BeTrue())
		})
	})
})

// mockedMediaFileRepo's FindByPaths method returns a list of MediaFiles with the same paths as the input
type mockedMediaFileRepo struct {
	model.MediaFileRepository
}

func (r *mockedMediaFileRepo) FindByPaths(paths []string) (model.MediaFiles, error) {
	var mfs model.MediaFiles
	for idx, path := range paths {
		mfs = append(mfs, model.MediaFile{
			ID:   strconv.Itoa(idx),
			Path: path,
		})
	}
	return mfs, nil
}

// mockedMediaFileFromListRepo's FindByPaths method returns a list of MediaFiles based on the data field
type mockedMediaFileFromListRepo struct {
	model.MediaFileRepository
	data []string
}

func (r *mockedMediaFileFromListRepo) FindByPaths([]string) (model.MediaFiles, error) {
	var mfs model.MediaFiles
	for idx, path := range r.data {
		mfs = append(mfs, model.MediaFile{
			ID:   strconv.Itoa(idx),
			Path: path,
		})
	}
	return mfs, nil
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
