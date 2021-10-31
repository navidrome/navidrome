package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing/fstest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("walk_dir_tree", func() {
	baseDir := filepath.Join("tests", "fixtures")

	Describe("walkDirTree", func() {
		It("reads all info correctly", func() {
			var collected = dirMap{}
			results := make(walkResults, 5000)
			var err error
			go func() {
				err = walkDirTree(context.TODO(), baseDir, results)
			}()

			for {
				stats, more := <-results
				if !more {
					break
				}
				collected[stats.Path] = stats
			}

			Expect(err).To(BeNil())
			Expect(collected[baseDir]).To(MatchFields(IgnoreExtras, Fields{
				"HasImages":       BeTrue(),
				"HasPlaylist":     BeFalse(),
				"AudioFilesCount": BeNumerically("==", 4),
			}))
			Expect(collected[filepath.Join(baseDir, "playlists")].HasPlaylist).To(BeTrue())
			Expect(collected).To(HaveKey(filepath.Join(baseDir, "symlink2dir")))
			Expect(collected).To(HaveKey(filepath.Join(baseDir, "empty_folder")))
			Expect(collected).To(HaveKey(filepath.Join(baseDir, "folder_with_ignored_things")))
			Expect(collected).NotTo(HaveKey(filepath.Join(baseDir, ".hidden_folder")))
			Expect(collected).To(HaveKey(filepath.Join(baseDir, "...unhidden_folder")))
			Expect(collected).NotTo(HaveKey(filepath.Join(baseDir, "playlists", "junk")))
		})
	})

	Describe("isDirOrSymlinkToDir", func() {
		It("returns true for normal dirs", func() {
			dirEntry, _ := getDirEntry("tests", "fixtures")
			Expect(isDirOrSymlinkToDir(baseDir, dirEntry)).To(BeTrue())
		})
		It("returns true for symlinks to dirs", func() {
			dirEntry, _ := getDirEntry(baseDir, "symlink2dir")
			Expect(isDirOrSymlinkToDir(baseDir, dirEntry)).To(BeTrue())
		})
		It("returns false for files", func() {
			dirEntry, _ := getDirEntry(baseDir, "test.mp3")
			Expect(isDirOrSymlinkToDir(baseDir, dirEntry)).To(BeFalse())
		})
		It("returns false for symlinks to files", func() {
			dirEntry, _ := getDirEntry(baseDir, "symlink")
			Expect(isDirOrSymlinkToDir(baseDir, dirEntry)).To(BeFalse())
		})
	})
	Describe("isDirIgnored", func() {
		It("returns false for normal dirs", func() {
			dirEntry, _ := getDirEntry(baseDir, "empty_folder")
			Expect(isDirIgnored(baseDir, dirEntry)).To(BeFalse())
		})
		It("returns true when folder contains .ndignore file", func() {
			dirEntry, _ := getDirEntry(baseDir, "ignored_folder")
			Expect(isDirIgnored(baseDir, dirEntry)).To(BeTrue())
		})
		It("returns true when folder name starts with a `.`", func() {
			dirEntry, _ := getDirEntry(baseDir, ".hidden_folder")
			Expect(isDirIgnored(baseDir, dirEntry)).To(BeTrue())
		})
		It("returns false when folder name starts with ellipses", func() {
			dirEntry, _ := getDirEntry(baseDir, "...unhidden_folder")
			Expect(isDirIgnored(baseDir, dirEntry)).To(BeFalse())
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
func (fd *fakeDirFile) ReadDir(n int) ([]fs.DirEntry, error) {
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

func getDirEntry(baseDir, name string) (os.DirEntry, error) {
	dirEntries, _ := os.ReadDir(baseDir)
	for _, entry := range dirEntries {
		if entry.Name() == name {
			return entry, nil
		}
	}
	return nil, os.ErrNotExist
}
