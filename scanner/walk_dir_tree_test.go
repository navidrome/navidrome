package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing/fstest"

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
				},
			}
		})

		It("walks all directories", func() {
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
		})
	})

	Describe("helper functions", func() {
		dir, _ := os.Getwd()
		fsys := os.DirFS(dir)
		baseDir := filepath.Join("tests", "fixtures")

		Describe("isDirOrSymlinkToDir", func() {
			It("returns true for normal dirs", func() {
				dirEntry := getDirEntry("tests", "fixtures")
				Expect(isDirOrSymlinkToDir(fsys, baseDir, dirEntry)).To(BeTrue())
			})
			It("returns true for symlinks to dirs", func() {
				dirEntry := getDirEntry(baseDir, "symlink2dir")
				Expect(isDirOrSymlinkToDir(fsys, baseDir, dirEntry)).To(BeTrue())
			})
			It("returns false for files", func() {
				dirEntry := getDirEntry(baseDir, "test.mp3")
				Expect(isDirOrSymlinkToDir(fsys, baseDir, dirEntry)).To(BeFalse())
			})
			It("returns false for symlinks to files", func() {
				dirEntry := getDirEntry(baseDir, "symlink")
				Expect(isDirOrSymlinkToDir(fsys, baseDir, dirEntry)).To(BeFalse())
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

type mockMusicFS struct {
	storage.MusicFS
	fs.FS
}

func (m *mockMusicFS) Open(name string) (fs.File, error) {
	return m.FS.Open(name)
}
