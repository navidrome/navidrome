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
	"golang.org/x/text/unicode/norm"
)

var _ = Describe("Playlists", func() {
	var ds *tests.MockDataStore
	var ps Playlists
	var mockPlsRepo mockedPlaylistRepo
	var mockLibRepo *tests.MockLibraryRepo
	ctx := context.Background()

	BeforeEach(func() {
		mockPlsRepo = mockedPlaylistRepo{}
		mockLibRepo = &tests.MockLibraryRepo{}
		ds = &tests.MockDataStore{
			MockedPlaylist: &mockPlsRepo,
			MockedLibrary:  mockLibRepo,
		}
		ctx = request.WithUser(ctx, model.User{ID: "123"})
		// Path should be libPath, but we want to match the root folder referenced in the m3u, which is `/`
		mockLibRepo.SetData([]model.Library{{ID: 1, Path: "/"}})
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
				pls, err := ps.ImportFile(ctx, folder, "pls1.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Tracks).To(HaveLen(2))
				Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/playlists/test.mp3"))
				Expect(pls.Tracks[1].Path).To(Equal("tests/fixtures/playlists/test.ogg"))
				Expect(mockPlsRepo.last).To(Equal(pls))
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
				Expect(mockPlsRepo.last).To(Equal(pls))
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
			mockLibRepo.SetData([]model.Library{{ID: 1, Path: "/music"}, {ID: 2, Path: "/new"}})
			ctx = request.WithUser(ctx, model.User{ID: "123"})
		})

		It("parses well-formed playlists", func() {
			repo.data = []string{
				"tests/test.mp3",
				"tests/test.ogg",
				"tests/01 Invisible (RED) Edit Version.mp3",
				"downloads/newfile.flac",
			}
			m3u := strings.Join([]string{
				"#PLAYLIST:playlist 1",
				"/music/tests/test.mp3",
				"/music/tests/test.ogg",
				"/new/downloads/newfile.flac",
				"file:///music/tests/01%20Invisible%20(RED)%20Edit%20Version.mp3",
			}, "\n")
			f := strings.NewReader(m3u)

			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.OwnerID).To(Equal("123"))
			Expect(pls.Name).To(Equal("playlist 1"))
			Expect(pls.Sync).To(BeFalse())
			Expect(pls.Tracks).To(HaveLen(4))
			Expect(pls.Tracks[0].Path).To(Equal("tests/test.mp3"))
			Expect(pls.Tracks[1].Path).To(Equal("tests/test.ogg"))
			Expect(pls.Tracks[2].Path).To(Equal("downloads/newfile.flac"))
			Expect(pls.Tracks[3].Path).To(Equal("tests/01 Invisible (RED) Edit Version.mp3"))
			Expect(mockPlsRepo.last).To(Equal(pls))
		})

		It("sets the playlist name as a timestamp if the #PLAYLIST directive is not present", func() {
			repo.data = []string{
				"tests/test.mp3",
				"tests/test.ogg",
				"/tests/01 Invisible (RED) Edit Version.mp3",
			}
			m3u := strings.Join([]string{
				"/music/tests/test.mp3",
				"/music/tests/test.ogg",
			}, "\n")
			f := strings.NewReader(m3u)
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			_, err = time.Parse(time.RFC3339, pls.Name)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(2))
		})

		It("returns only tracks that exist in the database and in the same other as the m3u", func() {
			repo.data = []string{
				"album1/test1.mp3",
				"album2/test2.mp3",
				"album3/test3.mp3",
			}
			m3u := strings.Join([]string{
				"/music/album3/test3.mp3",
				"/music/album1/test1.mp3",
				"/music/album4/test4.mp3",
				"/music/album2/test2.mp3",
			}, "\n")
			f := strings.NewReader(m3u)
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(3))
			Expect(pls.Tracks[0].Path).To(Equal("album3/test3.mp3"))
			Expect(pls.Tracks[1].Path).To(Equal("album1/test1.mp3"))
			Expect(pls.Tracks[2].Path).To(Equal("album2/test2.mp3"))
		})

		It("is case-insensitive when comparing paths", func() {
			repo.data = []string{
				"abc/tEsT1.Mp3",
			}
			m3u := strings.Join([]string{
				"/music/ABC/TeSt1.mP3",
			}, "\n")
			f := strings.NewReader(m3u)
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(1))
			Expect(pls.Tracks[0].Path).To(Equal("abc/tEsT1.Mp3"))
		})

		It("handles Unicode normalization when comparing paths", func() {
			// Test case for Apple Music playlists that use NFC encoding vs macOS filesystem NFD
			// The character "è" can be represented as NFC (single codepoint) or NFD (e + combining accent)

			const pathWithAccents = "artist/Michèle Desrosiers/album/Noël.m4a"

			// Simulate a database entry with NFD encoding (as stored by macOS filesystem)
			nfdPath := norm.NFD.String(pathWithAccents)
			repo.data = []string{nfdPath}

			// Simulate an Apple Music M3U playlist entry with NFC encoding
			nfcPath := norm.NFC.String("/music/" + pathWithAccents)
			m3u := strings.Join([]string{
				nfcPath,
			}, "\n")
			f := strings.NewReader(m3u)

			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(1), "Should find the track despite Unicode normalization differences")
			Expect(pls.Tracks[0].Path).To(Equal(nfdPath))
		})
	})

	Describe("normalizePathForComparison", func() {
		It("normalizes Unicode characters to NFC form and converts to lowercase", func() {
			// Test with NFD (decomposed) input - as would come from macOS filesystem
			nfdPath := norm.NFD.String("Michèle") // Explicitly convert to NFD form
			normalized := normalizePathForComparison(nfdPath)
			Expect(normalized).To(Equal("michèle"))

			// Test with NFC (composed) input - as would come from Apple Music M3U
			nfcPath := "Michèle" // This might be in NFC form
			normalizedNfc := normalizePathForComparison(nfcPath)

			// Ensure the two paths are not equal in their original forms
			Expect(nfdPath).ToNot(Equal(nfcPath))

			// Both should normalize to the same result
			Expect(normalized).To(Equal(normalizedNfc))
		})

		It("handles paths with mixed case and Unicode characters", func() {
			path := "Artist/Noël Coward/Album/Song.mp3"
			normalized := normalizePathForComparison(path)
			Expect(normalized).To(Equal("artist/noël coward/album/song.mp3"))
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

type mockedPlaylistRepo struct {
	last *model.Playlist
	model.PlaylistRepository
}

func (r *mockedPlaylistRepo) FindByPath(string) (*model.Playlist, error) {
	return nil, model.ErrNotFound
}

func (r *mockedPlaylistRepo) Put(pls *model.Playlist) error {
	r.last = pls
	return nil
}
