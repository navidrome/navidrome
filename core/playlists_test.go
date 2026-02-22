package core_test

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
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
	var ps core.Playlists
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
	})

	Describe("ImportFile", func() {
		var folder *model.Folder
		BeforeEach(func() {
			ps = core.NewPlaylists(ds)
			ds.MockedMediaFile = &mockedMediaFileRepo{}
			libPath, _ := os.Getwd()
			// Set up library with the actual library path that matches the folder
			mockLibRepo.SetData([]model.Library{{ID: 1, Path: libPath}})
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

			It("parses playlists with UTF-8 BOM marker", func() {
				pls, err := ps.ImportFile(ctx, folder, "bom-test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Name).To(Equal("Test Playlist"))
				Expect(pls.Tracks).To(HaveLen(1))
				Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/playlists/test.mp3"))
			})

			It("parses UTF-16 LE encoded playlists with BOM and converts to UTF-8", func() {
				pls, err := ps.ImportFile(ctx, folder, "bom-test-utf16.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Name).To(Equal("UTF-16 Test Playlist"))
				Expect(pls.Tracks).To(HaveLen(1))
				Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/playlists/test.mp3"))
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
			It("parses NSP with public: true and creates public playlist", func() {
				pls, err := ps.ImportFile(ctx, folder, "public_playlist.nsp")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Name).To(Equal("Public Playlist"))
				Expect(pls.Public).To(BeTrue())
			})
			It("parses NSP with public: false and creates private playlist", func() {
				pls, err := ps.ImportFile(ctx, folder, "private_playlist.nsp")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Name).To(Equal("Private Playlist"))
				Expect(pls.Public).To(BeFalse())
			})
			It("uses server default when public field is absent", func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.DefaultPlaylistPublicVisibility = true

				pls, err := ps.ImportFile(ctx, folder, "recently_played.nsp")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Name).To(Equal("Recently Played"))
				Expect(pls.Public).To(BeTrue()) // Should be true since server default is true
			})
		})

		DescribeTable("Playlist filename Unicode normalization (regression fix-playlist-filename-normalization)",
			func(storedForm, filesystemForm string) {
				// Use Polish characters that decompose: ó (U+00F3) -> o + combining acute (U+006F + U+0301)
				plsNameNFC := "Piosenki_Polskie_zółć" // NFC form (composed)
				plsNameNFD := norm.NFD.String(plsNameNFC)
				Expect(plsNameNFD).ToNot(Equal(plsNameNFC)) // Verify they differ

				nameByForm := map[string]string{"NFC": plsNameNFC, "NFD": plsNameNFD}
				storedName := nameByForm[storedForm]
				filesystemName := nameByForm[filesystemForm]

				tmpDir := GinkgoT().TempDir()
				mockLibRepo.SetData([]model.Library{{ID: 1, Path: tmpDir}})
				ds.MockedMediaFile = &mockedMediaFileFromListRepo{data: []string{}}
				ps = core.NewPlaylists(ds)

				// Create the playlist file on disk with the filesystem's normalization form
				plsFile := tmpDir + "/" + filesystemName + ".m3u"
				Expect(os.WriteFile(plsFile, []byte("#PLAYLIST:Test\n"), 0600)).To(Succeed())

				// Pre-populate mock repo with the stored normalization form
				storedPath := tmpDir + "/" + storedName + ".m3u"
				existingPls := &model.Playlist{
					ID:   "existing-id",
					Name: "Existing Playlist",
					Path: storedPath,
					Sync: true,
				}
				mockPlsRepo.data = map[string]*model.Playlist{storedPath: existingPls}

				// Import using the filesystem's normalization form
				plsFolder := &model.Folder{
					ID:          "1",
					LibraryID:   1,
					LibraryPath: tmpDir,
					Path:        "",
					Name:        "",
				}
				pls, err := ps.ImportFile(ctx, plsFolder, filesystemName+".m3u")
				Expect(err).ToNot(HaveOccurred())

				// Should update existing playlist, not create new one
				Expect(pls.ID).To(Equal("existing-id"))
				Expect(pls.Name).To(Equal("Existing Playlist"))
			},
			Entry("finds NFD-stored playlist when filesystem provides NFC path", "NFD", "NFC"),
			Entry("finds NFC-stored playlist when filesystem provides NFD path", "NFC", "NFD"),
		)

		Describe("Cross-library relative paths", func() {
			var tmpDir, plsDir, songsDir string

			BeforeEach(func() {
				// Create temp directory structure
				tmpDir = GinkgoT().TempDir()
				plsDir = tmpDir + "/playlists"
				songsDir = tmpDir + "/songs"
				Expect(os.Mkdir(plsDir, 0755)).To(Succeed())
				Expect(os.Mkdir(songsDir, 0755)).To(Succeed())

				// Setup two different libraries with paths matching our temp structure
				mockLibRepo.SetData([]model.Library{
					{ID: 1, Path: songsDir},
					{ID: 2, Path: plsDir},
				})

				// Create a mock media file repository that returns files for both libraries
				// Note: The paths are relative to their respective library roots
				ds.MockedMediaFile = &mockedMediaFileFromListRepo{
					data: []string{
						"abc.mp3", // This is songs/abc.mp3 relative to songsDir
						"def.mp3", // This is playlists/def.mp3 relative to plsDir
					},
				}
				ps = core.NewPlaylists(ds)
			})

			It("handles relative paths that reference files in other libraries", func() {
				// Create a temporary playlist file with relative path
				plsContent := "#PLAYLIST:Cross Library Test\n../songs/abc.mp3\ndef.mp3"
				plsFile := plsDir + "/test.m3u"
				Expect(os.WriteFile(plsFile, []byte(plsContent), 0600)).To(Succeed())

				// Playlist is in the Playlists library folder
				// Important: Path should be relative to LibraryPath, and Name is the folder name
				plsFolder := &model.Folder{
					ID:          "2",
					LibraryID:   2,
					LibraryPath: plsDir,
					Path:        "",
					Name:        "",
				}

				pls, err := ps.ImportFile(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(2))
				Expect(pls.Tracks[0].Path).To(Equal("abc.mp3")) // From songsDir library
				Expect(pls.Tracks[1].Path).To(Equal("def.mp3")) // From plsDir library
			})

			It("ignores paths that point outside all libraries", func() {
				// Create a temporary playlist file with path outside libraries
				plsContent := "#PLAYLIST:Outside Test\n../../outside.mp3\nabc.mp3"
				plsFile := plsDir + "/test.m3u"
				Expect(os.WriteFile(plsFile, []byte(plsContent), 0600)).To(Succeed())

				plsFolder := &model.Folder{
					ID:          "2",
					LibraryID:   2,
					LibraryPath: plsDir,
					Path:        "",
					Name:        "",
				}

				pls, err := ps.ImportFile(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				// Should only find abc.mp3, not outside.mp3
				Expect(pls.Tracks).To(HaveLen(1))
				Expect(pls.Tracks[0].Path).To(Equal("abc.mp3"))
			})

			It("handles relative paths with multiple '../' components", func() {
				// Create a nested structure: tmpDir/playlists/subfolder/test.m3u
				subFolder := plsDir + "/subfolder"
				Expect(os.Mkdir(subFolder, 0755)).To(Succeed())

				// Create the media file in the subfolder directory
				// The mock will return it as "def.mp3" relative to plsDir
				ds.MockedMediaFile = &mockedMediaFileFromListRepo{
					data: []string{
						"abc.mp3", // From songsDir library
						"def.mp3", // From plsDir library root
					},
				}

				// From subfolder, ../../songs/abc.mp3 should resolve to songs library
				// ../def.mp3 should resolve to plsDir/def.mp3
				plsContent := "#PLAYLIST:Nested Test\n../../songs/abc.mp3\n../def.mp3"
				plsFile := subFolder + "/test.m3u"
				Expect(os.WriteFile(plsFile, []byte(plsContent), 0600)).To(Succeed())

				// The folder: AbsolutePath = LibraryPath + Path + Name
				// So for /playlists/subfolder: LibraryPath=/playlists, Path="", Name="subfolder"
				plsFolder := &model.Folder{
					ID:          "2",
					LibraryID:   2,
					LibraryPath: plsDir,
					Path:        "",          // Empty because subfolder is directly under library root
					Name:        "subfolder", // The folder name
				}

				pls, err := ps.ImportFile(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(2))
				Expect(pls.Tracks[0].Path).To(Equal("abc.mp3")) // From songsDir library
				Expect(pls.Tracks[1].Path).To(Equal("def.mp3")) // From plsDir library root
			})

			It("correctly resolves libraries when one path is a prefix of another", func() {
				// This tests the bug where /music would match before /music-classical
				// Create temp directory structure with prefix conflict
				tmpDir := GinkgoT().TempDir()
				musicDir := tmpDir + "/music"
				musicClassicalDir := tmpDir + "/music-classical"
				Expect(os.Mkdir(musicDir, 0755)).To(Succeed())
				Expect(os.Mkdir(musicClassicalDir, 0755)).To(Succeed())

				// Setup two libraries where one is a prefix of the other
				mockLibRepo.SetData([]model.Library{
					{ID: 1, Path: musicDir},          // /tmp/xxx/music
					{ID: 2, Path: musicClassicalDir}, // /tmp/xxx/music-classical
				})

				// Mock will return tracks from both libraries
				ds.MockedMediaFile = &mockedMediaFileFromListRepo{
					data: []string{
						"rock.mp3", // From music library
						"bach.mp3", // From music-classical library
					},
				}

				// Create playlist in music library that references music-classical
				plsContent := "#PLAYLIST:Cross Prefix Test\nrock.mp3\n../music-classical/bach.mp3"
				plsFile := musicDir + "/test.m3u"
				Expect(os.WriteFile(plsFile, []byte(plsContent), 0600)).To(Succeed())

				plsFolder := &model.Folder{
					ID:          "1",
					LibraryID:   1,
					LibraryPath: musicDir,
					Path:        "",
					Name:        "",
				}

				pls, err := ps.ImportFile(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(2))
				Expect(pls.Tracks[0].Path).To(Equal("rock.mp3")) // From music library
				Expect(pls.Tracks[1].Path).To(Equal("bach.mp3")) // From music-classical library (not music!)
			})

			It("correctly handles identical relative paths from different libraries", func() {
				// This tests the bug where two libraries have files at the same relative path
				// and only one appears in the playlist
				tmpDir := GinkgoT().TempDir()
				musicDir := tmpDir + "/music"
				classicalDir := tmpDir + "/classical"
				Expect(os.Mkdir(musicDir, 0755)).To(Succeed())
				Expect(os.Mkdir(classicalDir, 0755)).To(Succeed())
				Expect(os.MkdirAll(musicDir+"/album", 0755)).To(Succeed())
				Expect(os.MkdirAll(classicalDir+"/album", 0755)).To(Succeed())
				// Create placeholder files so paths resolve correctly
				Expect(os.WriteFile(musicDir+"/album/track.mp3", []byte{}, 0600)).To(Succeed())
				Expect(os.WriteFile(classicalDir+"/album/track.mp3", []byte{}, 0600)).To(Succeed())

				// Both libraries have a file at "album/track.mp3"
				mockLibRepo.SetData([]model.Library{
					{ID: 1, Path: musicDir},
					{ID: 2, Path: classicalDir},
				})

				// Mock returns files with same relative path but different IDs and library IDs
				// Keys use the library-qualified format: "libraryID:path"
				ds.MockedMediaFile = &mockedMediaFileRepo{
					data: map[string]model.MediaFile{
						"1:album/track.mp3": {ID: "music-track", Path: "album/track.mp3", LibraryID: 1, Title: "Rock Song"},
						"2:album/track.mp3": {ID: "classical-track", Path: "album/track.mp3", LibraryID: 2, Title: "Classical Piece"},
					},
				}
				// Recreate playlists service to pick up new mock
				ps = core.NewPlaylists(ds)

				// Create playlist in music library that references both tracks
				plsContent := "#PLAYLIST:Same Path Test\nalbum/track.mp3\n../classical/album/track.mp3"
				plsFile := musicDir + "/test.m3u"
				Expect(os.WriteFile(plsFile, []byte(plsContent), 0600)).To(Succeed())

				plsFolder := &model.Folder{
					ID:          "1",
					LibraryID:   1,
					LibraryPath: musicDir,
					Path:        "",
					Name:        "",
				}

				pls, err := ps.ImportFile(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())

				// Should have BOTH tracks, not just one
				Expect(pls.Tracks).To(HaveLen(2), "Playlist should contain both tracks with same relative path")

				// Verify we got tracks from DIFFERENT libraries (the key fix!)
				// Collect the library IDs
				libIDs := make(map[int]bool)
				for _, track := range pls.Tracks {
					libIDs[track.LibraryID] = true
				}
				Expect(libIDs).To(HaveLen(2), "Tracks should come from two different libraries")
				Expect(libIDs[1]).To(BeTrue(), "Should have track from library 1")
				Expect(libIDs[2]).To(BeTrue(), "Should have track from library 2")

				// Both tracks should have the same relative path
				Expect(pls.Tracks[0].Path).To(Equal("album/track.mp3"))
				Expect(pls.Tracks[1].Path).To(Equal("album/track.mp3"))
			})
		})
	})

	Describe("ImportM3U", func() {
		var repo *mockedMediaFileFromListRepo
		BeforeEach(func() {
			repo = &mockedMediaFileFromListRepo{}
			ds.MockedMediaFile = repo
			ps = core.NewPlaylists(ds)
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

		// Fullwidth characters (e.g., ＡＢＣＤ) are not handled by SQLite's NOCASE collation,
		// so we need exact matching for non-ASCII characters.
		It("matches fullwidth characters exactly (SQLite NOCASE limitation)", func() {
			// Fullwidth uppercase ＡＣＲＯＳＳ (U+FF21, U+FF23, U+FF32, U+FF2F, U+FF33, U+FF33)
			repo.data = []string{
				"plex/02 - ＡＣＲＯＳＳ.flac",
			}
			m3u := "/music/plex/02 - ＡＣＲＯＳＳ.flac\n"
			f := strings.NewReader(m3u)
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(1))
			Expect(pls.Tracks[0].Path).To(Equal("plex/02 - ＡＣＲＯＳＳ.flac"))
		})

		// Unicode normalization tests: NFC (composed) vs NFD (decomposed) forms
		// macOS stores paths in NFD, Linux/Windows use NFC. Playlists may use either form.
		DescribeTable("matches paths across Unicode NFC/NFD normalization",
			func(description, pathNFC string, dbForm, playlistForm norm.Form) {
				pathNFD := norm.NFD.String(pathNFC)
				Expect(pathNFD).ToNot(Equal(pathNFC), "test path should have decomposable characters")

				// Set up DB with specified normalization form
				var dbPath string
				if dbForm == norm.NFC {
					dbPath = pathNFC
				} else {
					dbPath = pathNFD
				}
				repo.data = []string{dbPath}

				// Set up playlist with specified normalization form
				var playlistPath string
				if playlistForm == norm.NFC {
					playlistPath = pathNFC
				} else {
					playlistPath = pathNFD
				}
				m3u := "/music/" + playlistPath + "\n"
				f := strings.NewReader(m3u)

				pls, err := ps.ImportM3U(ctx, f)
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(1))
				Expect(pls.Tracks[0].Path).To(Equal(dbPath))
			},
			// French: è (U+00E8) decomposes to e + combining grave (U+0065 + U+0300)
			Entry("French diacritics - DB:NFD, playlist:NFC",
				"macOS DB with Apple Music playlist",
				"artist/Michèle/song.mp3", norm.NFD, norm.NFC),

			// Japanese Katakana: ド (U+30C9) decomposes to ト (U+30C8) + combining dakuten (U+3099)
			Entry("Japanese Katakana with dakuten - DB:NFC, playlist:NFC (#4884)",
				"Linux/Windows DB with NFC playlist",
				"artist/\u30a2\u30a4\u30c9\u30eb/\u30c9\u30ea\u30fc\u30e0\u30bd\u30f3\u30b0.mp3", norm.NFC, norm.NFC),
			Entry("Japanese Katakana with dakuten - DB:NFD, playlist:NFC (#4884)",
				"macOS DB with NFC playlist",
				"artist/\u30a2\u30a4\u30c9\u30eb/\u30c9\u30ea\u30fc\u30e0\u30bd\u30f3\u30b0.mp3", norm.NFD, norm.NFC),

			// Cyrillic: й (U+0439) decomposes to и (U+0438) + combining breve (U+0306)
			Entry("Cyrillic characters - DB:NFD, playlist:NFC (#4791)",
				"macOS DB with NFC playlist",
				"Жуки/Батарейка/01 - Разлюбила.mp3", norm.NFD, norm.NFC),

			// Polish: ó (U+00F3) decomposes to o + combining acute (U+0301)
			Entry("Polish diacritics - DB:NFD, playlist:NFC (#4663)",
				"macOS DB with NFC playlist",
				"Zespół/Człowiek/Piosenka o miłości.mp3", norm.NFD, norm.NFC),
			Entry("Polish diacritics - DB:NFC, playlist:NFD",
				"Linux/Windows DB with macOS-exported playlist",
				"Zespół/Człowiek/Piosenka o miłości.mp3", norm.NFC, norm.NFD),
		)

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
			Expect(core.InPlaylistsPath(folder)).To(BeTrue())
		})

		It("returns true if PlaylistsPath is any (**/**)", func() {
			conf.Server.PlaylistsPath = "**/**"
			Expect(core.InPlaylistsPath(folder)).To(BeTrue())
		})

		It("returns true if folder is in PlaylistsPath", func() {
			conf.Server.PlaylistsPath = "other/**:playlists/**"
			Expect(core.InPlaylistsPath(folder)).To(BeTrue())
		})

		It("returns false if folder is not in PlaylistsPath", func() {
			conf.Server.PlaylistsPath = "other"
			Expect(core.InPlaylistsPath(folder)).To(BeFalse())
		})

		It("returns true if for a playlist in root of MusicFolder if PlaylistsPath is '.'", func() {
			conf.Server.PlaylistsPath = "."
			Expect(core.InPlaylistsPath(folder)).To(BeFalse())

			folder2 := model.Folder{
				LibraryPath: "/music",
				Path:        "",
				Name:        ".",
			}

			Expect(core.InPlaylistsPath(folder2)).To(BeTrue())
		})
	})
})

// mockedMediaFileRepo's FindByPaths method returns MediaFiles for the given paths.
// If data map is provided, looks up files by key; otherwise creates them from paths.
type mockedMediaFileRepo struct {
	model.MediaFileRepository
	data map[string]model.MediaFile
}

func (r *mockedMediaFileRepo) FindByPaths(paths []string) (model.MediaFiles, error) {
	var mfs model.MediaFiles

	// If data map provided, look up files
	if r.data != nil {
		for _, path := range paths {
			if mf, ok := r.data[path]; ok {
				mfs = append(mfs, mf)
			}
		}
		return mfs, nil
	}

	// Otherwise, create MediaFiles from paths
	for idx, path := range paths {
		// Strip library qualifier if present (format: "libraryID:path")
		actualPath := path
		libraryID := 1
		if parts := strings.SplitN(path, ":", 2); len(parts) == 2 {
			if id, err := strconv.Atoi(parts[0]); err == nil {
				libraryID = id
				actualPath = parts[1]
			}
		}

		mfs = append(mfs, model.MediaFile{
			ID:        strconv.Itoa(idx),
			Path:      actualPath,
			LibraryID: libraryID,
		})
	}
	return mfs, nil
}

// mockedMediaFileFromListRepo's FindByPaths method returns a list of MediaFiles based on the data field
type mockedMediaFileFromListRepo struct {
	model.MediaFileRepository
	data []string
}

func (r *mockedMediaFileFromListRepo) FindByPaths(paths []string) (model.MediaFiles, error) {
	var mfs model.MediaFiles

	for idx, dataPath := range r.data {
		for _, requestPath := range paths {
			// Strip library qualifier if present (format: "libraryID:path")
			actualPath := requestPath
			libraryID := 1
			if parts := strings.SplitN(requestPath, ":", 2); len(parts) == 2 {
				if id, err := strconv.Atoi(parts[0]); err == nil {
					libraryID = id
					actualPath = parts[1]
				}
			}

			// Case-insensitive comparison (like SQL's "collate nocase"), but with no
			// implicit Unicode normalization (SQLite does not normalize NFC/NFD).
			if strings.EqualFold(actualPath, dataPath) {
				mfs = append(mfs, model.MediaFile{
					ID:        strconv.Itoa(idx),
					Path:      dataPath, // Return original path from DB
					LibraryID: libraryID,
				})
				break
			}
		}
	}
	return mfs, nil
}

type mockedPlaylistRepo struct {
	last *model.Playlist
	data map[string]*model.Playlist // keyed by path
	model.PlaylistRepository
}

func (r *mockedPlaylistRepo) FindByPath(path string) (*model.Playlist, error) {
	if r.data != nil {
		if pls, ok := r.data[path]; ok {
			return pls, nil
		}
	}
	return nil, model.ErrNotFound
}

func (r *mockedPlaylistRepo) Put(pls *model.Playlist) error {
	r.last = pls
	return nil
}
