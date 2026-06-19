package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing/fstest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/storage"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"
)

var _ = Describe("walk_dir_tree", func() {
	Describe("walkDirTree", func() {
		var (
			fsys storage.MusicFS
			job  *scanJob
			ctx  context.Context
		)

		Context("full library", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				ctx = GinkgoT().Context()
				fsys = &mockMusicFS{
					FS: fstest.MapFS{
						"root/a/.ndignore":       {Data: []byte("ignored/*")},
						"root/a/f1.mp3":          {},
						"root/a/f2.mp3":          {},
						"root/a/ignored/bad.mp3": {},
						"root/b/cover.jpg":       {},
						"root/c/f3":              {},
						"root/d":                 {},
						"root/d/.ndignore":       {},
						"root/d/f1.mp3":          {},
						"root/d/f2.mp3":          {},
						"root/d/f3.mp3":          {},
						"root/e/original/f1.mp3": {},
						"root/e/symlink":         {Mode: fs.ModeSymlink, Data: []byte("original")},
						"root/f/realsong.mp3":    {Data: []byte("AUDIO")},
						"root/f/legit.mp3":       {Mode: fs.ModeSymlink, Data: []byte("realsong.mp3")},
						"root/f/secret":          {Data: []byte("TOPSECRET")},
						"root/f/evil.mp3":        {Mode: fs.ModeSymlink, Data: []byte("secret")},
					},
				}
				job = &scanJob{
					fs:  fsys,
					lib: model.Library{Path: "/music"},
				}
			})

			// Helper function to call walkDirTree and collect folders from the results channel
			getFolders := func() map[string]*folderEntry {
				results, err := walkDirTree(ctx, job)
				Expect(err).ToNot(HaveOccurred())

				folders := map[string]*folderEntry{}
				g := errgroup.Group{}
				g.Go(func() error {
					for folder := range results {
						folders[folder.path] = folder
					}
					return nil
				})
				_ = g.Wait()
				return folders
			}

			DescribeTable("symlink handling",
				func(followSymlinks bool, expectedFolderCount int) {
					conf.Server.Scanner.FollowSymlinks = followSymlinks
					folders := getFolders()

					Expect(folders).To(HaveLen(expectedFolderCount + 2)) // +2 for `.` and `root`

					// Basic folder structure checks
					Expect(folders["root/a"].audioFiles).To(SatisfyAll(
						HaveLen(2),
						HaveKey("f1.mp3"),
						HaveKey("f2.mp3"),
					))
					Expect(folders["root/a"].imageFiles).To(BeEmpty())
					Expect(folders["root/b"].audioFiles).To(BeEmpty())
					Expect(folders["root/b"].imageFiles).To(SatisfyAll(
						HaveLen(1),
						HaveKey("cover.jpg"),
					))
					Expect(folders["root/c"].audioFiles).To(BeEmpty())
					Expect(folders["root/c"].imageFiles).To(BeEmpty())
					Expect(folders).ToNot(HaveKey("root/d"))

					// Symlink specific checks
					if followSymlinks {
						Expect(folders["root/e/symlink"].audioFiles).To(HaveLen(1))
						Expect(folders["root/f"].audioFiles).To(HaveKey("legit.mp3"))
						Expect(folders["root/f"].audioFiles).To(HaveKey("realsong.mp3"))
						Expect(folders["root/f"].audioFiles).ToNot(HaveKey("evil.mp3"))
					} else {
						Expect(folders).ToNot(HaveKey("root/e/symlink"))
						Expect(folders["root/f"].audioFiles).To(HaveKey("realsong.mp3"))
						Expect(folders["root/f"].audioFiles).ToNot(HaveKey("legit.mp3"))
						Expect(folders["root/f"].audioFiles).ToNot(HaveKey("evil.mp3"))
					}
				},
				Entry("with symlinks enabled", true, 8),
				Entry("with symlinks disabled", false, 7),
			)
		})

		Context("with target folders", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				ctx = GinkgoT().Context()
				fsys = &mockMusicFS{
					FS: fstest.MapFS{
						"Artist/Album1/track1.mp3":      {},
						"Artist/Album1/track2.mp3":      {},
						"Artist/Album2/track1.mp3":      {},
						"Artist/Album2/track2.mp3":      {},
						"Artist/Album2/Sub/track3.mp3":  {},
						"OtherArtist/Album3/track1.mp3": {},
					},
				}
				job = &scanJob{
					fs:  fsys,
					lib: model.Library{Path: "/music"},
				}
			})

			It("should recursively walk all subdirectories of target folders", func() {
				results, err := walkDirTree(ctx, job, "Artist")
				Expect(err).ToNot(HaveOccurred())

				folders := map[string]*folderEntry{}
				g := errgroup.Group{}
				g.Go(func() error {
					for folder := range results {
						folders[folder.path] = folder
					}
					return nil
				})
				_ = g.Wait()

				// Should include the target folder and all its descendants
				Expect(folders).To(SatisfyAll(
					HaveKey("Artist"),
					HaveKey("Artist/Album1"),
					HaveKey("Artist/Album2"),
					HaveKey("Artist/Album2/Sub"),
				))

				// Should not include folders outside the target
				Expect(folders).ToNot(HaveKey("OtherArtist"))
				Expect(folders).ToNot(HaveKey("OtherArtist/Album3"))

				// Verify audio files are present
				Expect(folders["Artist/Album1"].audioFiles).To(HaveLen(2))
				Expect(folders["Artist/Album2"].audioFiles).To(HaveLen(2))
				Expect(folders["Artist/Album2/Sub"].audioFiles).To(HaveLen(1))
			})

			It("should handle multiple target folders", func() {
				results, err := walkDirTree(ctx, job, "Artist/Album1", "OtherArtist")
				Expect(err).ToNot(HaveOccurred())

				folders := map[string]*folderEntry{}
				g := errgroup.Group{}
				g.Go(func() error {
					for folder := range results {
						folders[folder.path] = folder
					}
					return nil
				})
				_ = g.Wait()

				// Should include both target folders and their descendants
				Expect(folders).To(SatisfyAll(
					HaveKey("Artist/Album1"),
					HaveKey("OtherArtist"),
					HaveKey("OtherArtist/Album3"),
				))

				// Should not include other folders
				Expect(folders).ToNot(HaveKey("Artist"))
				Expect(folders).ToNot(HaveKey("Artist/Album2"))
				Expect(folders).ToNot(HaveKey("Artist/Album2/Sub"))
			})

			It("should skip non-existent target folders and preserve them in lastUpdates", func() {
				// Setup job with lastUpdates for both existing and non-existing folders
				job.lastUpdates = map[string]model.FolderUpdateInfo{
					model.FolderID(job.lib, "Artist/Album1"):             {},
					model.FolderID(job.lib, "NonExistent/DeletedFolder"): {},
					model.FolderID(job.lib, "OtherArtist/Album3"):        {},
				}

				// Try to scan existing folder and non-existing folder
				results, err := walkDirTree(ctx, job, "Artist/Album1", "NonExistent/DeletedFolder")
				Expect(err).ToNot(HaveOccurred())

				// Collect results
				folders := map[string]struct{}{}
				for folder := range results {
					folders[folder.path] = struct{}{}
				}

				// Should only include the existing folder
				Expect(folders).To(HaveKey("Artist/Album1"))
				Expect(folders).ToNot(HaveKey("NonExistent/DeletedFolder"))

				// The non-existent folder should still be in lastUpdates (not removed by popLastUpdate)
				Expect(job.lastUpdates).To(HaveKey(model.FolderID(job.lib, "NonExistent/DeletedFolder")))

				// The existing folder should have been removed from lastUpdates
				Expect(job.lastUpdates).ToNot(HaveKey(model.FolderID(job.lib, "Artist/Album1")))

				// Folders not in targets should remain in lastUpdates
				Expect(job.lastUpdates).To(HaveKey(model.FolderID(job.lib, "OtherArtist/Album3")))
			})
		})
	})

	Describe("helper functions", func() {
		dir, _ := os.Getwd()
		fsys := os.DirFS(dir)
		baseDir := filepath.Join("tests", "fixtures")

		Describe("isDirOrSymlinkToDir", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
			})

			Context("with symlinks enabled", func() {
				BeforeEach(func() {
					tests.SkipOnWindows("symlink semantics")
					conf.Server.Scanner.FollowSymlinks = true
				})

				DescribeTable("returns expected result",
					func(dirName string, expected bool) {
						dirEntry := getDirEntry("tests/fixtures", dirName)
						Expect(isDirOrSymlinkToDir(fsys, baseDir, dirEntry)).To(Equal(expected))
					},
					Entry("normal dir", "empty_folder", true),
					Entry("symlink to dir", "symlink2dir", true),
					Entry("regular file", "test.mp3", false),
					Entry("symlink to file", "symlink", false),
				)
			})

			Context("with symlinks disabled", func() {
				BeforeEach(func() {
					conf.Server.Scanner.FollowSymlinks = false
				})

				DescribeTable("returns expected result",
					func(dirName string, expected bool) {
						dirEntry := getDirEntry("tests/fixtures", dirName)
						Expect(isDirOrSymlinkToDir(fsys, baseDir, dirEntry)).To(Equal(expected))
					},
					Entry("normal dir", "empty_folder", true),
					Entry("symlink to dir", "symlink2dir", false),
					Entry("regular file", "test.mp3", false),
					Entry("symlink to file", "symlink", false),
				)
			})
		})

		Describe("resolveEntryName", func() {
			var fsys fs.FS
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				fsys = fstest.MapFS{
					"dir/real.mp3":   {Data: []byte("AUDIO")},
					"dir/mid.mp3":    {Mode: fs.ModeSymlink, Data: []byte("real.mp3")},
					"dir/chain.mp3":  {Mode: fs.ModeSymlink, Data: []byte("mid.mp3")},
					"dir/audio.mp3":  {Mode: fs.ModeSymlink, Data: []byte("real.mp3")},
					"dir/evil.mp3":   {Mode: fs.ModeSymlink, Data: []byte("../outside/passwd")},
					"dir/loop1.mp3":  {Mode: fs.ModeSymlink, Data: []byte("loop2.mp3")},
					"dir/loop2.mp3":  {Mode: fs.ModeSymlink, Data: []byte("loop1.mp3")},
					"dir/dangle.mp3": {Mode: fs.ModeSymlink, Data: []byte("missing.mp3")},
				}
			})

			resolve := func(name string) (string, bool) {
				entries, err := fs.ReadDir(fsys, "dir")
				Expect(err).ToNot(HaveOccurred())
				for _, e := range entries {
					if e.Name() == name {
						return resolveEntryName(GinkgoT().Context(), fsys, "dir", e)
					}
				}
				Fail("entry not found: " + name)
				return "", false
			}

			Context("with symlinks enabled", func() {
				BeforeEach(func() { conf.Server.Scanner.FollowSymlinks = true })

				It("returns the entry name for a plain file", func() {
					name, ok := resolve("real.mp3")
					Expect(ok).To(BeTrue())
					Expect(name).To(Equal("real.mp3"))
				})
				It("resolves a direct symlink to its audio target name", func() {
					name, ok := resolve("audio.mp3")
					Expect(ok).To(BeTrue())
					Expect(name).To(Equal("real.mp3"))
				})
				It("resolves a symlink CHAIN to the final target name", func() {
					name, ok := resolve("chain.mp3")
					Expect(ok).To(BeTrue())
					Expect(name).To(Equal("real.mp3"))
				})
				It("resolves a symlink to a non-audio target name (so caller can reject it)", func() {
					name, ok := resolve("evil.mp3")
					Expect(ok).To(BeTrue())
					Expect(name).To(Equal("passwd"))
				})
				It("rejects a symlink loop", func() {
					_, ok := resolve("loop1.mp3")
					Expect(ok).To(BeFalse())
				})
			})

			Context("with symlinks disabled", func() {
				BeforeEach(func() { conf.Server.Scanner.FollowSymlinks = false })

				It("returns the entry name for a plain file", func() {
					name, ok := resolve("real.mp3")
					Expect(ok).To(BeTrue())
					Expect(name).To(Equal("real.mp3"))
				})
				It("skips any file symlink", func() {
					_, ok := resolve("audio.mp3")
					Expect(ok).To(BeFalse())
				})
			})
		})

		Describe("symlink chain (real fs)", func() {
			BeforeEach(func() {
				tests.SkipOnWindows("symlink semantics")
				DeferCleanup(configtest.SetupConfig())
			})

			classify := func(fsys fs.FS, dirPath, name string) (string, bool) {
				entries, err := fs.ReadDir(fsys, dirPath)
				Expect(err).ToNot(HaveOccurred())
				for _, e := range entries {
					if e.Name() == name {
						return resolveEntryName(GinkgoT().Context(), fsys, dirPath, e)
					}
				}
				Fail("entry not found: " + name)
				return "", false
			}

			Context("committed 3-level fixtures", func() {
				// tests.Init chdirs to the repo root, so the committed fixtures are at "tests/fixtures".
				var fsys fs.FS
				BeforeEach(func() {
					conf.Server.Scanner.FollowSymlinks = true
					wd, err := os.Getwd()
					Expect(err).ToNot(HaveOccurred())
					fsys = os.DirFS(wd)
				})

				It("keeps a 3-level chain that resolves to real audio", func() {
					name, ok := classify(fsys, "tests/fixtures/symlink_chain", "level3.mp3")
					Expect(ok).To(BeTrue())
					Expect(name).To(Equal("test.mp3"))
					Expect(model.IsAudioFile(name)).To(BeTrue())
				})

				It("rejects a 3-level chain that resolves to a non-audio file", func() {
					name, ok := classify(fsys, "tests/fixtures/symlink_chain", "evil3.mp3")
					Expect(ok).To(BeTrue())
					Expect(name).To(Equal("index.html"))
					Expect(model.IsAudioFile(name)).To(BeFalse())
				})

				It("skips the chain entirely when FollowSymlinks is disabled", func() {
					conf.Server.Scanner.FollowSymlinks = false
					_, ok := classify(fsys, "tests/fixtures/symlink_chain", "level3.mp3")
					Expect(ok).To(BeFalse())
					_, ok = classify(fsys, "tests/fixtures/symlink_chain", "evil3.mp3")
					Expect(ok).To(BeFalse())
				})
			})

			Context("out-of-tree escape (temp dir)", func() {
				var root string
				BeforeEach(func() {
					conf.Server.Scanner.FollowSymlinks = true
					root = GinkgoT().TempDir()
					outside := GinkgoT().TempDir()
					Expect(os.WriteFile(filepath.Join(outside, "passwd"), []byte("TOPSECRET"), 0600)).To(Succeed())
					Expect(os.WriteFile(filepath.Join(outside, "real.flac"), []byte("AUDIO"), 0600)).To(Succeed())
					Expect(os.WriteFile(filepath.Join(root, "song.mp3"), []byte("AUDIO"), 0600)).To(Succeed())
					// evil.mp3 escapes to a non-audio target; legit.flac is a valid out-of-tree audio symlink.
					Expect(os.Symlink(filepath.Join(outside, "passwd"), filepath.Join(root, "evil.mp3"))).To(Succeed())
					Expect(os.Symlink(filepath.Join(outside, "real.flac"), filepath.Join(root, "legit.flac"))).To(Succeed())
				})

				It("rejects the absolute-path escape but keeps legit out-of-tree audio", func() {
					fsys := os.DirFS(root)

					name, ok := classify(fsys, ".", "song.mp3")
					Expect(ok).To(BeTrue())
					Expect(model.IsAudioFile(name)).To(BeTrue())

					name, ok = classify(fsys, ".", "legit.flac")
					Expect(ok).To(BeTrue())
					Expect(model.IsAudioFile(name)).To(BeTrue())

					name, ok = classify(fsys, ".", "evil.mp3")
					Expect(ok).To(BeTrue())
					Expect(model.IsAudioFile(name)).To(BeFalse())
				})

				It("skips all file symlinks when FollowSymlinks is disabled", func() {
					conf.Server.Scanner.FollowSymlinks = false
					fsys := os.DirFS(root)
					entries, err := fs.ReadDir(fsys, ".")
					Expect(err).ToNot(HaveOccurred())
					for _, e := range entries {
						_, ok := resolveEntryName(GinkgoT().Context(), fsys, ".", e)
						if e.Type()&fs.ModeSymlink != 0 {
							Expect(ok).To(BeFalse(), e.Name())
						} else {
							Expect(ok).To(BeTrue(), e.Name())
						}
					}
				})
			})
		})

		Describe("isDirIgnored", func() {
			DescribeTable("returns expected result",
				func(dirName string, expected bool) {
					Expect(isDirIgnored(dirName)).To(Equal(expected))
				},
				Entry("normal dir", "empty_folder", false),
				Entry("hidden dir", ".hidden_folder", true),
				Entry("dir starting with ellipsis", "...unhidden_folder", false),
				Entry("recycle bin", "$Recycle.Bin", true),
				Entry("snapshot dir", "#snapshot", true),
			)
		})

		Describe("fullReadDir", func() {
			var (
				fsys fakeFS
				ctx  context.Context
			)

			BeforeEach(func() {
				ctx = GinkgoT().Context()
				fsys = fakeFS{MapFS: fstest.MapFS{
					"root/a/f1": {},
					"root/b/f2": {},
					"root/c/f3": {},
				}}
			})

			DescribeTable("reading directory entries",
				func(failOn string, expectedErr error, expectedNames []string) {
					fsys.failOn = failOn
					fsys.err = expectedErr
					dir, _ := fsys.Open("root")
					entries := fullReadDir(ctx, dir.(fs.ReadDirFile))
					Expect(entries).To(HaveLen(len(expectedNames)))
					for i, name := range expectedNames {
						Expect(entries[i].Name()).To(Equal(name))
					}
				},
				Entry("reads all entries", "", nil, []string{"a", "b", "c"}),
				Entry("skips entries with permission error", "b", nil, []string{"a", "c"}),
				Entry("aborts on fs.ErrNotExist", "", fs.ErrNotExist, []string{}),
			)
		})
	})
})

type fakeFS struct {
	fstest.MapFS
	failOn string
	err    error
}

func (f *fakeFS) Open(name string) (fs.File, error) {
	dir, err := f.MapFS.Open(name)
	return &fakeDirFile{File: dir, fail: f.failOn, err: f.err}, err
}

type fakeDirFile struct {
	fs.File
	entries []fs.DirEntry
	pos     int
	fail    string
	err     error
}

// Only works with n == -1
func (fd *fakeDirFile) ReadDir(int) ([]fs.DirEntry, error) {
	if fd.err != nil {
		return nil, fd.err
	}
	if fd.entries == nil {
		fd.entries, _ = fd.File.(fs.ReadDirFile).ReadDir(-1)
	}
	var dirs []fs.DirEntry
	for {
		if fd.pos >= len(fd.entries) {
			break
		}
		e := fd.entries[fd.pos]
		fd.pos++
		if e.Name() == fd.fail {
			return dirs, &fs.PathError{Op: "lstat", Path: e.Name(), Err: fs.ErrPermission}
		}
		dirs = append(dirs, e)
	}
	return dirs, nil
}

func getDirEntry(baseDir, name string) os.DirEntry {
	dirEntries, _ := os.ReadDir(baseDir)
	for _, entry := range dirEntries {
		if entry.Name() == name {
			return entry
		}
	}
	panic(fmt.Sprintf("Could not find %s in %s", name, baseDir))
}

// mockMusicFS is a mock implementation of the MusicFS interface that supports symlinks
type mockMusicFS struct {
	storage.MusicFS
	fs.FS
}

// Open resolves symlinks
func (m *mockMusicFS) Open(name string) (fs.File, error) {
	f, err := m.FS.Open(name)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	if info.Mode()&fs.ModeSymlink != 0 {
		// For symlinks, read the target path from the Data field
		target := string(m.FS.(fstest.MapFS)[name].Data)
		f.Close()
		return m.FS.Open(target)
	}

	return f, nil
}

// Stat uses Open to resolve symlinks
func (m *mockMusicFS) Stat(name string) (fs.FileInfo, error) {
	f, err := m.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}

// ReadDir uses Open to resolve symlinks
func (m *mockMusicFS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, err := m.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if dirFile, ok := f.(fs.ReadDirFile); ok {
		return dirFile.ReadDir(-1)
	}
	return nil, fmt.Errorf("not a directory")
}

// ReadLink returns the target of the named symbolic link (implements fs.ReadLinkFS).
func (m *mockMusicFS) ReadLink(name string) (string, error) {
	mapFS := m.FS.(fstest.MapFS)
	entry, ok := mapFS[name]
	if !ok {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: fs.ErrNotExist}
	}
	if entry.Mode&fs.ModeSymlink == 0 {
		return "", &fs.PathError{Op: "readlink", Path: name, Err: fmt.Errorf("not a symlink")}
	}
	return string(entry.Data), nil
}

// Lstat returns FileInfo for the named file without following symlinks (implements fs.ReadLinkFS).
func (m *mockMusicFS) Lstat(name string) (fs.FileInfo, error) {
	mapFS := m.FS.(fstest.MapFS)
	if _, ok := mapFS[name]; !ok {
		return nil, &fs.PathError{Op: "lstat", Path: name, Err: fs.ErrNotExist}
	}
	f, err := m.FS.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}
