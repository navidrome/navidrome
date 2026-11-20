package playlists_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing/iotest"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/playlists"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/criteria"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zeebo/xxh3"
	"golang.org/x/text/unicode/norm"
)

var _ = Describe("Playlists - Import", func() {
	var ds model.DataStore
	var ps playlists.Playlists
	var ctx context.Context

	useLibrary := func(dir string, tracks ...string) {
		GinkgoHelper()
		persistencetest.SetLibraries(ctx, ds, dir)
		putTracks(ctx, ds, model.DefaultLibraryID, tracks...)
	}
	savedPlaylists := func() model.Playlists {
		GinkgoHelper()
		all, err := ds.Playlist().GetAll(asAdmin(ctx))
		Expect(err).ToNot(HaveOccurred())
		return all
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		putUsers(ctx, ds, "123")
		ps = playlists.NewPlaylists(ds, artwork.NewUploader(ds))
		ctx = request.WithUser(ctx, model.User{ID: "123"})
	})

	Describe("ImportFromFolder", func() {
		var folder *model.Folder
		BeforeEach(func() {
			libPath, _ := os.Getwd()
			// Set up library with the actual library path that matches the folder
			useLibrary(libPath, "tests/fixtures/playlists/test.mp3", "tests/fixtures/playlists/test.ogg",
				"tests/fixtures/playlists/abc.mp3", "tests/fixtures/playlists/def.mp3")
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
				pls, err := ps.ImportFromFolder(ctx, folder, "pls1.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Tracks).To(HaveLen(2))
				Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/playlists/test.mp3"))
				Expect(pls.Tracks[1].Path).To(Equal("tests/fixtures/playlists/test.ogg"))
				stored := storedPlaylist(ctx, ds, pls.ID)
				Expect(stored.OwnerID).To(Equal("123"))
				Expect(trackIDs(stored)).To(Equal(trackIDs(pls)))
			})

			It("parses playlists using LF ending", func() {
				pls, err := ps.ImportFromFolder(ctx, folder, "lf-ended.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(2))
			})

			It("parses playlists using CR ending (old Mac format)", func() {
				pls, err := ps.ImportFromFolder(ctx, folder, "cr-ended.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(2))
			})

			It("parses playlists with UTF-8 BOM marker", func() {
				pls, err := ps.ImportFromFolder(ctx, folder, "bom-test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Name).To(Equal("Test Playlist"))
				Expect(pls.Tracks).To(HaveLen(1))
				Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/playlists/test.mp3"))
			})

			It("parses UTF-16 LE encoded playlists with BOM and converts to UTF-8", func() {
				pls, err := ps.ImportFromFolder(ctx, folder, "bom-test-utf16.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Name).To(Equal("UTF-16 Test Playlist"))
				Expect(pls.Tracks).To(HaveLen(1))
				Expect(pls.Tracks[0].Path).To(Equal("tests/fixtures/playlists/test.mp3"))
			})

			It("parses #EXTALBUMARTURL with HTTP URL", func() {
				conf.Update(func(c *conf.Options) { c.EnableM3UExternalAlbumArt = true })

				pls, err := ps.ImportFromFolder(ctx, folder, "pls-with-art-url.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(Equal("https://example.com/cover.jpg"))
				Expect(pls.Tracks).To(HaveLen(2))
			})

			It("parses #EXTALBUMARTURL with absolute local path", func() {
				tmpDir := GinkgoT().TempDir()
				imgPath := filepath.Join(tmpDir, "cover.jpg")
				Expect(os.WriteFile(imgPath, []byte("fake image"), 0600)).To(Succeed())

				m3u := fmt.Sprintf("#EXTALBUMARTURL:%s\ntest.mp3\ntest.ogg\n", imgPath)
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3", "test.ogg")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(Equal(imgPath))
			})

			It("parses #EXTALBUMARTURL with relative local path", func() {
				tmpDir := GinkgoT().TempDir()
				Expect(os.WriteFile(filepath.Join(tmpDir, "cover.jpg"), []byte("fake image"), 0600)).To(Succeed())

				m3u := "#EXTALBUMARTURL:cover.jpg\ntest.mp3\n"
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(Equal(filepath.Join(tmpDir, "cover.jpg")))
			})

			It("parses #EXTALBUMARTURL with file:// URL", func() {
				tmpDir := GinkgoT().TempDir()
				imgPath := filepath.Join(tmpDir, "my cover.jpg")
				Expect(os.WriteFile(imgPath, []byte("fake image"), 0600)).To(Succeed())

				m3u := fmt.Sprintf("#EXTALBUMARTURL:file://%s\ntest.mp3\n", strings.ReplaceAll(imgPath, " ", "%20"))
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(Equal(imgPath))
			})

			It("preserves + in file:// URLs (PathUnescape, not QueryUnescape)", func() {
				tmpDir := GinkgoT().TempDir()
				imgPath := filepath.Join(tmpDir, "A+B.jpg")
				Expect(os.WriteFile(imgPath, []byte("fake image"), 0600)).To(Succeed())

				m3u := fmt.Sprintf("#EXTALBUMARTURL:file://%s\ntest.mp3\n", imgPath)
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(Equal(imgPath))
			})

			It("rejects #EXTALBUMARTURL with absolute path outside library boundaries", func() {
				tests.SkipOnWindows("relies on Unix /etc filesystem")
				tmpDir := GinkgoT().TempDir()

				m3u := "#EXTALBUMARTURL:/etc/passwd\ntest.mp3\n"
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(BeEmpty())
			})

			It("rejects #EXTALBUMARTURL with file:// URL outside library boundaries", func() {
				tmpDir := GinkgoT().TempDir()

				m3u := "#EXTALBUMARTURL:file:///etc/passwd\ntest.mp3\n"
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(BeEmpty())
			})

			It("rejects #EXTALBUMARTURL with relative path escaping library", func() {
				tmpDir := GinkgoT().TempDir()

				m3u := "#EXTALBUMARTURL:../../etc/passwd\ntest.mp3\n"
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(BeEmpty())
			})

			It("rejects #EXTALBUMARTURL pointing at a non-image file inside the library", func() {
				tmpDir := GinkgoT().TempDir()
				Expect(os.WriteFile(filepath.Join(tmpDir, "config.ini"), []byte("password=secret"), 0600)).To(Succeed())

				m3u := "#EXTALBUMARTURL:config.ini\ntest.mp3\n"
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(BeEmpty())
			})

			It("ignores HTTP #EXTALBUMARTURL when EnableM3UExternalAlbumArt is false", func() {
				conf.Update(func(c *conf.Options) { c.EnableM3UExternalAlbumArt = false })

				tmpDir := GinkgoT().TempDir()
				m3u := "#EXTALBUMARTURL:https://example.com/cover.jpg\ntest.mp3\n"
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				useLibrary(tmpDir, "test.mp3")

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(BeEmpty())
			})

			It("updates ExternalImageURL on re-scan even when UploadedImage is set", func() {
				conf.Update(func(c *conf.Options) { c.EnableM3UExternalAlbumArt = true })

				tmpDir := GinkgoT().TempDir()
				useLibrary(tmpDir, "test.mp3")

				m3u := "#EXTALBUMARTURL:https://example.com/new-cover.jpg\ntest.mp3\n"
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				existingPls := &model.Playlist{
					ID:               "existing-id",
					Name:             "Existing Playlist",
					Path:             plsFile,
					Sync:             true,
					OwnerID:          "123",
					UploadedImage:    "existing-id.jpg",
					ExternalImageURL: "https://example.com/old-cover.jpg",
				}
				putPlaylists(ctx, ds, existingPls)

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.UploadedImage).To(Equal("existing-id.jpg"))
				Expect(pls.ExternalImageURL).To(Equal("https://example.com/new-cover.jpg"))
				stored := storedPlaylist(ctx, ds, "existing-id")
				Expect(stored.UploadedImage).To(Equal("existing-id.jpg"))
				Expect(stored.ExternalImageURL).To(Equal("https://example.com/new-cover.jpg"))
			})

			It("skips non-synced playlist on re-import (respects user's choice)", func() {
				tmpDir := GinkgoT().TempDir()
				useLibrary(tmpDir, "test.mp3")

				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte("test.mp3\n"), 0600)).To(Succeed())

				existingPls := &model.Playlist{
					ID:      "existing-id",
					Name:    "Existing Playlist",
					Path:    plsFile,
					Sync:    false,
					OwnerID: "123",
				}
				putPlaylists(ctx, ds, existingPls)

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				// updatePlaylist skips the non-synced playlist, so the returned
				// playlist has no ID (was never persisted/updated).
				Expect(pls.ID).To(BeEmpty())
				Expect(storedPlaylist(ctx, ds, "existing-id").Tracks).To(BeEmpty())
			})

			It("stores a content hash but re-imports unchanged M3U playlists", func() {
				tmpDir := GinkgoT().TempDir()
				useLibrary(tmpDir)

				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte("test.mp3\n"), 0600)).To(Succeed())
				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}

				first, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(first.ImportedHash).ToNot(BeEmpty())
				Expect(storedPlaylist(ctx, ds, first.ID).ImportedHash).To(Equal(first.ImportedHash))

				// Re-import with a matching stored hash: M3U must still be re-imported, so a track added since resolves.
				putTracks(ctx, ds, model.DefaultLibraryID, "test.mp3")
				_, err = ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(storedPlaylist(ctx, ds, first.ID).Tracks).To(HaveLen(1))
			})

			It("clears ExternalImageURL on re-scan when directive is removed", func() {
				tmpDir := GinkgoT().TempDir()
				useLibrary(tmpDir, "test.mp3")

				m3u := "test.mp3\n"
				plsFile := filepath.Join(tmpDir, "test.m3u")
				Expect(os.WriteFile(plsFile, []byte(m3u), 0600)).To(Succeed())

				existingPls := &model.Playlist{
					ID:               "existing-id",
					Name:             "Existing Playlist",
					Path:             plsFile,
					Sync:             true,
					OwnerID:          "123",
					ExternalImageURL: "https://example.com/old-cover.jpg",
				}
				putPlaylists(ctx, ds, existingPls)

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(BeEmpty())
				Expect(storedPlaylist(ctx, ds, "existing-id").ExternalImageURL).To(BeEmpty())
			})
		})

		Describe("NSP", func() {
			It("parses well-formed playlists", func() {
				pls, err := ps.ImportFromFolder(ctx, folder, "recently_played.nsp")
				Expect(err).ToNot(HaveOccurred())
				stored := storedPlaylist(ctx, ds, pls.ID)
				Expect(stored.Name).To(Equal(pls.Name))
				Expect(stored.Rules).To(Equal(pls.Rules))
				Expect(pls.OwnerID).To(Equal("123"))
				Expect(pls.Name).To(Equal("Recently Played"))
				Expect(pls.Comment).To(Equal("Recently played tracks"))
				Expect(pls.Rules.Sort).To(Equal("lastPlayed"))
				Expect(pls.Rules.Order).To(Equal("desc"))
				Expect(pls.Rules.Limit).To(Equal(100))
				Expect(pls.Rules.Expression).To(BeAssignableToTypeOf(criteria.All{}))
			})
			It("returns an error if the playlist is not well-formed", func() {
				tests.SkipOnWindows("line-ending differences affect JSON error offset")
				_, err := ps.ImportFromFolder(ctx, folder, "invalid_json.nsp")
				Expect(err.Error()).To(ContainSubstring("line 19, column 1: invalid character '\\n'"))
			})
			It("parses NSP with public: true and creates public playlist", func() {
				pls, err := ps.ImportFromFolder(ctx, folder, "public_playlist.nsp")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Name).To(Equal("Public Playlist"))
				Expect(pls.Public).To(BeTrue())
			})
			It("parses NSP with public: false and creates private playlist", func() {
				pls, err := ps.ImportFromFolder(ctx, folder, "private_playlist.nsp")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Name).To(Equal("Private Playlist"))
				Expect(pls.Public).To(BeFalse())
			})
			It("uses server default when public field is absent", func() {
				conf.Update(func(c *conf.Options) { c.DefaultPlaylistPublicVisibility = true })

				pls, err := ps.ImportFromFolder(ctx, folder, "recently_played.nsp")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Name).To(Equal("Recently Played"))
				Expect(pls.Public).To(BeTrue()) // Should be true since server default is true
			})

			It("preserves counters when re-importing an existing smart playlist", func() {
				tmpDir := GinkgoT().TempDir()
				useLibrary(tmpDir)

				nsp := `{"name":"My Smart","all":[{"is":{"loved":true}}],"sort":"title","order":"asc"}`
				plsFile := filepath.Join(tmpDir, "smart.nsp")
				Expect(os.WriteFile(plsFile, []byte(nsp), 0600)).To(Succeed())

				existingPls := &model.Playlist{
					ID:      "smart-id",
					Name:    "My Smart",
					Path:    plsFile,
					Sync:    true,
					OwnerID: "123",
					Rules:   &criteria.Criteria{Expression: criteria.All{criteria.Is{"loved": true}}},
				}
				putPlaylists(ctx, ds, existingPls)
				// Put leaves a smart playlist's counters to its evaluation, so set them directly.
				_, err := db.Db().Exec("UPDATE playlist SET song_count = 42, duration = 123.4, size = 5000 WHERE id = 'smart-id'")
				Expect(err).ToNot(HaveOccurred())

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, "smart.nsp")

				Expect(err).ToNot(HaveOccurred())
				Expect(pls.IsSmartPlaylist()).To(BeTrue())
				Expect(pls.SongCount).To(Equal(42))
				Expect(pls.Duration).To(Equal(float32(123.4)))
				Expect(pls.Size).To(Equal(int64(5000)))
				Expect(storedPlaylist(ctx, ds, "smart-id").Rules.Sort).To(Equal("title"))
			})

			It("skips re-import when the smart playlist file content is unchanged", func() {
				tmpDir := GinkgoT().TempDir()
				useLibrary(tmpDir)

				nsp := `{"name":"My Smart","all":[{"is":{"loved":true}}]}`
				plsFile := filepath.Join(tmpDir, "smart.nsp")
				Expect(os.WriteFile(plsFile, []byte(nsp), 0600)).To(Succeed())

				existingPls := &model.Playlist{
					ID: "smart-id", Name: "My Smart", Path: plsFile, Sync: true,
					OwnerID: "123", SongCount: 42,
					ImportedHash: hashOf(nsp),
				}
				putPlaylists(ctx, ds, existingPls)
				originalTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
				backdate("smart-id", originalTime)

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				_, err := ps.ImportFromFolder(ctx, plsFolder, "smart.nsp")

				Expect(err).ToNot(HaveOccurred())
				stored := storedPlaylist(ctx, ds, "smart-id")
				Expect(stored.UpdatedAt).To(BeTemporally("==", originalTime))
				Expect(stored.IsSmartPlaylist()).To(BeFalse())
			})

			It("re-imports when the smart playlist file content changed", func() {
				tmpDir := GinkgoT().TempDir()
				useLibrary(tmpDir)

				nsp := `{"name":"My Smart","all":[{"is":{"loved":true}}]}`
				plsFile := filepath.Join(tmpDir, "smart.nsp")
				Expect(os.WriteFile(plsFile, []byte(nsp), 0600)).To(Succeed())

				existingPls := &model.Playlist{
					ID: "smart-id", Name: "My Smart", Path: plsFile, Sync: true,
					OwnerID: "123", SongCount: 42,
					ImportedHash: hashOf("old content"),
				}
				putPlaylists(ctx, ds, existingPls)

				plsFolder := &model.Folder{ID: "1", LibraryID: 1, LibraryPath: tmpDir, Path: "", Name: ""}
				_, err := ps.ImportFromFolder(ctx, plsFolder, "smart.nsp")

				Expect(err).ToNot(HaveOccurred())
				Expect(storedPlaylist(ctx, ds, "smart-id").ImportedHash).To(Equal(hashOf(nsp)))
			})
		})

		DescribeTable("Playlist filename Unicode normalization (regression fix-playlist-filename-normalization)",
			func(storedForm, filesystemForm string) {
				tests.SkipOnWindows("/tmp hardcoded in test")
				// Use Polish characters that decompose: ó (U+00F3) -> o + combining acute (U+006F + U+0301)
				plsNameNFC := "Piosenki_Polskie_zółć" // NFC form (composed)
				plsNameNFD := norm.NFD.String(plsNameNFC)
				Expect(plsNameNFD).ToNot(Equal(plsNameNFC)) // Verify they differ

				nameByForm := map[string]string{"NFC": plsNameNFC, "NFD": plsNameNFD}
				storedName := nameByForm[storedForm]
				filesystemName := nameByForm[filesystemForm]

				tmpDir := GinkgoT().TempDir()
				useLibrary(tmpDir)

				// Create the playlist file on disk with the filesystem's normalization form
				plsFile := tmpDir + "/" + filesystemName + ".m3u"
				Expect(os.WriteFile(plsFile, []byte("#PLAYLIST:Test\n"), 0600)).To(Succeed())

				// Pre-populate the DB with the stored normalization form
				storedPath := tmpDir + "/" + storedName + ".m3u"
				existingPls := &model.Playlist{
					ID:      "existing-id",
					Name:    "Existing Playlist",
					Path:    storedPath,
					Sync:    true,
					OwnerID: "123",
				}
				putPlaylists(ctx, ds, existingPls)

				// Import using the filesystem's normalization form
				plsFolder := &model.Folder{
					ID:          "1",
					LibraryID:   1,
					LibraryPath: tmpDir,
					Path:        "",
					Name:        "",
				}
				pls, err := ps.ImportFromFolder(ctx, plsFolder, filesystemName+".m3u")
				Expect(err).ToNot(HaveOccurred())

				// Should update existing playlist, not create new one
				Expect(pls.ID).To(Equal("existing-id"))
				Expect(pls.Name).To(Equal("Existing Playlist"))
				Expect(savedPlaylists()).To(HaveLen(1))
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
				persistencetest.SetLibraries(ctx, ds, songsDir, plsDir)

				// Note: The paths are relative to their respective library roots
				putTracks(ctx, ds, 1, "abc.mp3") // This is songs/abc.mp3 relative to songsDir
				putTracks(ctx, ds, 2, "def.mp3") // This is playlists/def.mp3 relative to plsDir
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

				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(2))
				Expect(pls.Tracks[0].Path).To(Equal("abc.mp3")) // From songsDir library
				Expect(pls.Tracks[1].Path).To(Equal("def.mp3")) // From plsDir library
			})

			It("ignores paths that point outside all libraries", func() {
				// Create a temporary playlist file with path outside libraries
				plsContent := "#PLAYLIST:Outside Test\n../../outside.mp3\n../songs/abc.mp3"
				plsFile := plsDir + "/test.m3u"
				Expect(os.WriteFile(plsFile, []byte(plsContent), 0600)).To(Succeed())

				plsFolder := &model.Folder{
					ID:          "2",
					LibraryID:   2,
					LibraryPath: plsDir,
					Path:        "",
					Name:        "",
				}

				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
				Expect(err).ToNot(HaveOccurred())
				// Should only find abc.mp3, not outside.mp3
				Expect(pls.Tracks).To(HaveLen(1))
				Expect(pls.Tracks[0].Path).To(Equal("abc.mp3"))
			})

			It("handles relative paths with multiple '../' components", func() {
				// Create a nested structure: tmpDir/playlists/subfolder/test.m3u
				subFolder := plsDir + "/subfolder"
				Expect(os.Mkdir(subFolder, 0755)).To(Succeed())

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

				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
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
				persistencetest.SetLibraries(ctx, ds, musicDir, musicClassicalDir)
				putTracks(ctx, ds, 1, "rock.mp3")
				putTracks(ctx, ds, 2, "bach.mp3")

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

				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
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
				persistencetest.SetLibraries(ctx, ds, musicDir, classicalDir)
				putTracks(ctx, ds, 1, "album/track.mp3")
				putTracks(ctx, ds, 2, "album/track.mp3")

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

				pls, err := ps.ImportFromFolder(ctx, plsFolder, "test.m3u")
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

	Describe("ImportFile", func() {
		useLibraryWithRoot := func(dir string) {
			GinkgoHelper()
			useLibrary(dir, "test.mp3", "test.ogg")
			putLibraryRoot(ctx, ds, model.DefaultLibraryID)
		}

		It("resolves file inside a library and imports it", func() {
			tmpDir := GinkgoT().TempDir()
			useLibraryWithRoot(tmpDir)

			plsContent := "#PLAYLIST:My Playlist\ntest.mp3\ntest.ogg\n"
			plsFile := filepath.Join(tmpDir, "my-playlist.m3u")
			Expect(os.WriteFile(plsFile, []byte(plsContent), 0600)).To(Succeed())

			pls, err := ps.ImportFile(ctx, plsFile, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Name).To(Equal("My Playlist"))
			Expect(pls.Tracks).To(HaveLen(2))
			Expect(pls.Path).To(Equal(plsFile))
			Expect(pls.Sync).To(BeTrue())
		})

		It("records path for files outside all libraries", func() {
			tmpDir := GinkgoT().TempDir()
			libDir := filepath.Join(tmpDir, "music")
			Expect(os.Mkdir(libDir, 0755)).To(Succeed())
			useLibrary(libDir, "test.mp3", "test.ogg")

			plsContent := "#PLAYLIST:External Playlist\n" + libDir + "/test.mp3\n"
			plsFile := filepath.Join(tmpDir, "external.m3u")
			Expect(os.WriteFile(plsFile, []byte(plsContent), 0600)).To(Succeed())

			pls, err := ps.ImportFile(ctx, plsFile, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Name).To(Equal("External Playlist"))
			Expect(pls.Path).To(Equal(plsFile))
			Expect(pls.Sync).To(BeFalse())
		})

		It("imports with Sync=false", func() {
			tmpDir := GinkgoT().TempDir()
			useLibraryWithRoot(tmpDir)

			plsFile := filepath.Join(tmpDir, "test.m3u")
			Expect(os.WriteFile(plsFile, []byte("test.mp3\n"), 0600)).To(Succeed())

			pls, err := ps.ImportFile(ctx, plsFile, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Sync).To(BeFalse())
		})

		It("imports with Sync=true", func() {
			tmpDir := GinkgoT().TempDir()
			useLibraryWithRoot(tmpDir)

			plsFile := filepath.Join(tmpDir, "test.m3u")
			Expect(os.WriteFile(plsFile, []byte("test.mp3\n"), 0600)).To(Succeed())

			pls, err := ps.ImportFile(ctx, plsFile, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Sync).To(BeTrue())
		})

		It("upgrades non-synced playlist to synced on re-import with sync=true", func() {
			tmpDir := GinkgoT().TempDir()
			useLibraryWithRoot(tmpDir)

			plsFile := filepath.Join(tmpDir, "test.m3u")
			Expect(os.WriteFile(plsFile, []byte("test.mp3\n"), 0600)).To(Succeed())

			existingPls := &model.Playlist{
				ID: "existing-id", Name: "Existing", Path: plsFile,
				Sync: false, OwnerID: "123",
			}
			putPlaylists(ctx, ds, existingPls)

			pls, err := ps.ImportFile(ctx, plsFile, true)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.ID).To(Equal("existing-id"))
			Expect(pls.Sync).To(BeTrue())
			Expect(storedPlaylist(ctx, ds, "existing-id").Sync).To(BeTrue())
		})

		It("unsyncs a synced smart playlist with sync=false even when content is unchanged", func() {
			tmpDir := GinkgoT().TempDir()
			useLibraryWithRoot(tmpDir)

			nsp := `{"name":"My Smart","all":[{"is":{"loved":true}}]}`
			plsFile := filepath.Join(tmpDir, "smart.nsp")
			Expect(os.WriteFile(plsFile, []byte(nsp), 0600)).To(Succeed())

			existingPls := &model.Playlist{
				ID: "smart-id", Name: "My Smart", Path: plsFile, Sync: true,
				OwnerID: "123", SongCount: 42,
				ImportedHash: hashOf(nsp),
			}
			putPlaylists(ctx, ds, existingPls)

			pls, err := ps.ImportFile(ctx, plsFile, false)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.ID).To(Equal("smart-id"))
			Expect(pls.Sync).To(BeFalse())
			Expect(storedPlaylist(ctx, ds, "smart-id").Sync).To(BeFalse())
		})
	})

	Describe("ImportM3U", func() {
		BeforeEach(func() {
			persistencetest.SetLibraries(ctx, ds, "/music", "/new")
		})

		It("parses well-formed playlists", func() {
			putTracks(ctx, ds, 1, "tests/test.mp3", "tests/test.ogg", "tests/01 Invisible (RED) Edit Version.mp3")
			putTracks(ctx, ds, 2, "downloads/newfile.flac")
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
			stored := storedPlaylist(ctx, ds, pls.ID)
			Expect(stored.OwnerID).To(Equal("123"))
			Expect(trackIDs(stored)).To(Equal(trackIDs(pls)))
		})

		It("sets the playlist name as a timestamp if the #PLAYLIST directive is not present", func() {
			putTracks(ctx, ds, model.DefaultLibraryID,
				"tests/test.mp3",
				"tests/test.ogg",
				"/tests/01 Invisible (RED) Edit Version.mp3",
			)
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

		It("returns only tracks that exist in the database and in the same order as the m3u", func() {
			putTracks(ctx, ds, model.DefaultLibraryID,
				"album1/test1.mp3",
				"album2/test2.mp3",
				"album3/test3.mp3",
			)
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

		Describe("entry limit", func() {
			repeat := func(n int) string {
				return strings.Repeat("/music/tests/test.mp3\n", n)
			}
			BeforeEach(func() {
				putTracks(ctx, ds, model.DefaultLibraryID, "tests/test.mp3")
				conf.Update(func(c *conf.Options) { c.DevMaxPlaylistTracksPerRequest = 3 })
			})

			It("imports a playlist with exactly DevMaxPlaylistTracksPerRequest entries, duplicates included", func() {
				pls, err := ps.ImportM3U(ctx, strings.NewReader(repeat(3)))
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(3))
			})

			It("rejects a playlist over the limit without saving it", func() {
				_, err := ps.ImportM3U(ctx, strings.NewReader(repeat(4)))
				Expect(err).To(MatchError(playlists.ErrTooManyEntries))
				Expect(savedPlaylists()).To(BeEmpty())
			})

			It("imports any number of entries when the limit is 0", func() {
				conf.Update(func(c *conf.Options) { c.DevMaxPlaylistTracksPerRequest = 0 })
				pls, err := ps.ImportM3U(ctx, strings.NewReader(repeat(4)))
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(4))
			})

			It("does not cap playlists imported from disk", func() {
				tmpDir := GinkgoT().TempDir()
				libDir := filepath.Join(tmpDir, "music")
				Expect(os.Mkdir(libDir, 0755)).To(Succeed())
				useLibrary(libDir)
				plsFile := filepath.Join(tmpDir, "big.m3u")
				content := strings.Repeat(libDir+"/tests/test.mp3\n", 4)
				Expect(os.WriteFile(plsFile, []byte(content), 0600)).To(Succeed())

				pls, err := ps.ImportFile(ctx, plsFile, false)
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.Tracks).To(HaveLen(4))
			})
		})

		It("makes a second upload wait for the running one, without reading its body", func() {
			putTracks(ctx, ds, model.DefaultLibraryID, "tests/test.mp3")
			pr, pw := io.Pipe()
			firstDone := make(chan error, 1)
			go func() {
				_, err := ps.ImportM3U(ctx, pr)
				firstDone <- err
			}()
			_, err := pw.Write([]byte("/music/tests/test.mp3\n"))
			Expect(err).ToNot(HaveOccurred())

			waiting, cancel := context.WithCancel(ctx)
			cancel()
			second := &readRecorder{}
			_, err = ps.ImportM3U(waiting, second)
			Expect(err).To(MatchError(context.Canceled))
			Expect(second.read).To(BeFalse())

			Expect(pw.Close()).To(Succeed())
			Eventually(firstDone).Should(Receive(BeNil()))
		})

		It("fails on a read error instead of saving a truncated playlist", func() {
			putTracks(ctx, ds, model.DefaultLibraryID, "tests/test.mp3")
			readErr := errors.New("body too large")
			body := io.MultiReader(strings.NewReader("/music/tests/test.mp3\n"), iotest.ErrReader(readErr))

			_, err := ps.ImportM3U(ctx, body)
			Expect(err).To(MatchError(readErr))
			Expect(savedPlaylists()).To(BeEmpty())
		})

		It("is case-insensitive when comparing paths", func() {
			putTracks(ctx, ds, model.DefaultLibraryID,
				"abc/tEsT1.Mp3",
			)
			m3u := strings.Join([]string{
				"/music/ABC/TeSt1.mP3",
			}, "\n")
			f := strings.NewReader(m3u)
			pls, err := ps.ImportM3U(ctx, f)
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.Tracks).To(HaveLen(1))
			Expect(pls.Tracks[0].Path).To(Equal("abc/tEsT1.Mp3"))
		})

		It("parses #EXTALBUMARTURL with HTTP URL via ImportM3U", func() {
			conf.Update(func(c *conf.Options) { c.EnableM3UExternalAlbumArt = true })

			putTracks(ctx, ds, model.DefaultLibraryID, "tests/test.mp3")
			m3u := "#EXTALBUMARTURL:https://example.com/cover.jpg\n/music/tests/test.mp3\n"
			pls, err := ps.ImportM3U(ctx, strings.NewReader(m3u))
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.ExternalImageURL).To(Equal("https://example.com/cover.jpg"))
		})

		It("ignores relative #EXTALBUMARTURL when imported via API (no folder context)", func() {
			putTracks(ctx, ds, model.DefaultLibraryID, "tests/test.mp3")
			m3u := "#EXTALBUMARTURL:cover.jpg\n/music/tests/test.mp3\n"
			pls, err := ps.ImportM3U(ctx, strings.NewReader(m3u))
			Expect(err).ToNot(HaveOccurred())
			Expect(pls.ExternalImageURL).To(BeEmpty())
		})

		DescribeTable("restricts a local #EXTALBUMARTURL to the owner's libraries",
			func(imageURL, expected string) {
				ctx = request.WithUser(ctx, model.User{ID: "123", Libraries: model.Libraries{{ID: 1, Path: "/music"}}})
				putTracks(ctx, ds, model.DefaultLibraryID, "tests/test.mp3")
				m3u := "#EXTALBUMARTURL:" + imageURL + "\n/music/tests/test.mp3\n"
				pls, err := ps.ImportM3U(ctx, strings.NewReader(m3u))
				Expect(err).ToNot(HaveOccurred())
				Expect(pls.ExternalImageURL).To(Equal(expected))
			},
			Entry("accepts a library the owner can access", "file:///music/cover.jpg", filepath.Clean("/music/cover.jpg")),
			Entry("ignores a library the owner cannot access", "file:///new/cover.jpg", ""),
		)

		// Fullwidth characters (e.g., ＡＢＣＤ) are not handled by SQLite's NOCASE collation,
		// so we need exact matching for non-ASCII characters.
		It("matches fullwidth characters exactly (SQLite NOCASE limitation)", func() {
			// Fullwidth uppercase ＡＣＲＯＳＳ (U+FF21, U+FF23, U+FF32, U+FF2F, U+FF33, U+FF33)
			putTracks(ctx, ds, model.DefaultLibraryID,
				"plex/02 - ＡＣＲＯＳＳ.flac",
			)
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
				putTracks(ctx, ds, model.DefaultLibraryID, dbPath)

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

	Describe("InPath", func() {
		var folder model.Folder

		BeforeEach(func() {
			folder = model.Folder{
				LibraryPath: "/music",
				Path:        "playlists/abc",
				Name:        "folder1",
			}
		})

		It("returns true if PlaylistsPath is empty", func() {
			conf.Update(func(c *conf.Options) { c.PlaylistsPath = "" })
			Expect(playlists.InPath(folder)).To(BeTrue())
		})

		It("returns true if PlaylistsPath is any (**/**)", func() {
			conf.Update(func(c *conf.Options) { c.PlaylistsPath = "**/**" })
			Expect(playlists.InPath(folder)).To(BeTrue())
		})

		It("returns true if folder is in PlaylistsPath", func() {
			tests.SkipOnWindows("path separator bug (#TBD-path-sep-playlists)")
			conf.Update(func(c *conf.Options) { c.PlaylistsPath = "other/**:playlists/**" })
			Expect(playlists.InPath(folder)).To(BeTrue())
		})

		It("returns false if folder is not in PlaylistsPath", func() {
			conf.Update(func(c *conf.Options) { c.PlaylistsPath = "other" })
			Expect(playlists.InPath(folder)).To(BeFalse())
		})

		It("returns true if for a playlist in root of MusicFolder if PlaylistsPath is '.'", func() {
			conf.Update(func(c *conf.Options) { c.PlaylistsPath = "." })
			Expect(playlists.InPath(folder)).To(BeFalse())

			folder2 := model.Folder{
				LibraryPath: "/music",
				Path:        "",
				Name:        ".",
			}

			Expect(playlists.InPath(folder2)).To(BeTrue())
		})
	})
})

func hashOf(content string) string {
	return id.Encode(xxh3.Hash128([]byte(content)).Bytes())
}

type readRecorder struct{ read bool }

func (r *readRecorder) Read([]byte) (int, error) {
	r.read = true
	return 0, io.EOF
}
