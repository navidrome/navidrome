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
		var fsys storage.MusicFS
		BeforeEach(func() {
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
		})

		Context("with symlinks enabled", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.Scanner.FollowSymlinks = true
			})

			It("walks all directories including symlinks", func() {
				job := &scanJob{
					fs:  fsys,
					lib: model.Library{Path: "/music"},
				}
				ctx := context.Background()
				results, err := walkDirTree(ctx, job)
				Expect(err).ToNot(HaveOccurred())

				folders := map[string]*folderEntry{}

				g := errgroup.Group{}
				g.Go(func() error {
					for folder := range results {
						if folder.path == "." || folder.path == "root" {
							continue // Skip root folders
						}
						folders[folder.path] = folder
					}
					return nil
				})
				_ = g.Wait()

				Expect(folders).To(HaveLen(7))
				Expect(folders["root/a/ignored"].audioFiles).To(BeEmpty())
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
				Expect(folders["root/e/symlink"].audioFiles).To(HaveLen(1))
			})
		})

		Context("with symlinks disabled", func() {
			BeforeEach(func() {
				DeferCleanup(configtest.SetupConfig())
				conf.Server.Scanner.FollowSymlinks = false
			})

			It("skips symlinks", func() {
				job := &scanJob{
					fs:  fsys,
					lib: model.Library{Path: "/music"},
				}
				ctx := context.Background()
				results, err := walkDirTree(ctx, job)
				Expect(err).ToNot(HaveOccurred())

				folders := map[string]*folderEntry{}

				g := errgroup.Group{}
				g.Go(func() error {
					for folder := range results {
						if folder.path == "." || folder.path == "root" {
							continue // Skip root folders
						}
						folders[folder.path] = folder
					}
					return nil
				})
				_ = g.Wait()

				Expect(folders).To(HaveLen(6))
				Expect(folders["root/a/ignored"].audioFiles).To(BeEmpty())
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
				Expect(folders).ToNot(HaveKey("root/e/symlink"))
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
			It("returns false for normal dirs", func() {
				Expect(isDirIgnored("empty_folder")).To(BeFalse())
			})
			It("returns true when folder name starts with a `.`", func() {
				Expect(isDirIgnored(".hidden_folder")).To(BeTrue())
			})
			It("returns false when folder name starts with ellipses", func() {
				Expect(isDirIgnored("...unhidden_folder")).To(BeFalse())
			})
			It("returns true when folder name is $Recycle.Bin", func() {
				Expect(isDirIgnored("$Recycle.Bin")).To(BeTrue())
			})
			It("returns true when folder name is #snapshot", func() {
				Expect(isDirIgnored("#snapshot")).To(BeTrue())
			})
		})
	})

	Describe("fullReadDir", func() {
		var fsys fakeFS
		var ctx context.Context
		BeforeEach(func() {
			ctx = context.Background()
			fsys = fakeFS{MapFS: fstest.MapFS{
				"root/a/f1": {},
				"root/b/f2": {},
				"root/c/f3": {},
			}}
		})
		It("reads all entries", func() {
			dir, _ := fsys.Open("root")
			entries := fullReadDir(ctx, dir.(fs.ReadDirFile))
			Expect(entries).To(HaveLen(3))
			Expect(entries[0].Name()).To(Equal("a"))
			Expect(entries[1].Name()).To(Equal("b"))
			Expect(entries[2].Name()).To(Equal("c"))
		})
		It("skips entries with permission error", func() {
			fsys.failOn = "b"
			dir, _ := fsys.Open("root")
			entries := fullReadDir(ctx, dir.(fs.ReadDirFile))
			Expect(entries).To(HaveLen(2))
			Expect(entries[0].Name()).To(Equal("a"))
			Expect(entries[1].Name()).To(Equal("c"))
		})
		It("aborts if it keeps getting 'readdirent: no such file or directory'", func() {
			fsys.err = fs.ErrNotExist
			dir, _ := fsys.Open("root")
			entries := fullReadDir(ctx, dir.(fs.ReadDirFile))
			Expect(entries).To(BeEmpty())
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
