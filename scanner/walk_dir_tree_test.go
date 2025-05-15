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
					"root/e/symlink":         {Mode: fs.ModeSymlink, Data: []byte("root/e/original")},
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
				} else {
					Expect(folders).ToNot(HaveKey("root/e/symlink"))
				}
			},
			Entry("with symlinks enabled", true, 7),
			Entry("with symlinks disabled", false, 6),
		)
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
