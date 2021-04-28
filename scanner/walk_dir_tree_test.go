package scanner

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

func getDirEntry(baseDir, name string) (os.DirEntry, error) {
	dirents, _ := os.ReadDir(baseDir)
	for _, d := range dirents {
		if d.Name() == name {
			return d, nil
		}
	}
	return nil, os.ErrNotExist
}

var _ = Describe("load_tree", func() {

	Describe("walkDirTree", func() {
		It("reads all info correctly", func() {
			baseDir := filepath.Join("tests", "fixtures")
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
		})
	})

	Describe("isDirOrSymlinkToDir", func() {
		baseDir := filepath.Join("tests", "fixtures")
		It("returns true for normal dirs", func() {
			dirent, _ := getDirEntry("tests", "fixtures")
			Expect(isDirOrSymlinkToDir(baseDir, dirent)).To(BeTrue())
		})
		It("returns true for symlinks to dirs", func() {
			dirent, _ := getDirEntry(baseDir, "symlink2dir")
			Expect(isDirOrSymlinkToDir(baseDir, dirent)).To(BeTrue())
		})
		It("returns false for files", func() {
			dirent, _ := getDirEntry(baseDir, "test.mp3")
			Expect(isDirOrSymlinkToDir(baseDir, dirent)).To(BeFalse())
		})
		It("returns false for symlinks to files", func() {
			dirent, _ := getDirEntry(baseDir, "symlink")
			Expect(isDirOrSymlinkToDir(baseDir, dirent)).To(BeFalse())
		})
	})
	Describe("isDirIgnored", func() {
		baseDir := filepath.Join("tests", "fixtures")
		It("returns false for normal dirs", func() {
			dirent, _ := getDirEntry(baseDir, "empty_folder")
			Expect(isDirIgnored(baseDir, dirent)).To(BeFalse())
		})
		It("returns true when folder contains .ndignore file", func() {
			dirent, _ := getDirEntry(baseDir, "ignored_folder")
			Expect(isDirIgnored(baseDir, dirent)).To(BeTrue())
		})
		It("returns true when folder name starts with a `.`", func() {
			dirent, _ := getDirEntry(baseDir, ".hidden_folder")
			Expect(isDirIgnored(baseDir, dirent)).To(BeTrue())
		})
		It("returns false when folder name starts with ellipses", func() {
			dirent, _ := getDirEntry(baseDir, "...unhidden_folder")
			Expect(isDirIgnored(baseDir, dirent)).To(BeFalse())
		})
	})
})
