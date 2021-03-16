package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("load_tree", func() {
	dirFS := os.DirFS(".")

	Describe("walkDirTree", func() {
		It("reads all info correctly", func() {
			var collected = dirMap{}
			results := make(walkResults, 5000)
			var err error
			go func() {
				err = walkDirTree(context.TODO(), dirFS, "tests/fixtures", results)
			}()

			for {
				stats, more := <-results
				if !more {
					break
				}
				collected[stats.Path] = stats
			}

			Expect(err).To(BeNil())
			Expect(collected["tests/fixtures"]).To(MatchFields(IgnoreExtras, Fields{
				"HasImages":       BeTrue(),
				"HasPlaylist":     BeFalse(),
				"AudioFilesCount": BeNumerically("==", 4),
			}))
			Expect(collected["tests/fixtures/playlists"].HasPlaylist).To(BeTrue())
			Expect(collected).To(HaveKey("tests/fixtures/symlink2dir"))
			Expect(collected).To(HaveKey("tests/fixtures/empty_folder"))
		})
	})

	Describe("isDirOrSymlinkToDir", func() {
		It("returns true for normal dirs", func() {
			dir, _ := fs.Stat(dirFS, "tests/fixtures")
			Expect(isDirOrSymlinkToDir(dirFS, "tests", dir)).To(BeTrue())
		})
		It("returns true for symlinks to dirs", func() {
			dir, _ := fs.Stat(dirFS, "tests/fixtures/symlink2dir")
			Expect(isDirOrSymlinkToDir(dirFS, "tests/fixtures", dir)).To(BeTrue())
		})
		It("returns false for files", func() {
			dir, _ := fs.Stat(dirFS, "tests/fixtures/test.mp3")
			Expect(isDirOrSymlinkToDir(dirFS, "tests/fixtures", dir)).To(BeFalse())
		})
		It("returns false for symlinks to files", func() {
			dir, _ := fs.Stat(dirFS, "tests/fixtures/symlink")
			Expect(isDirOrSymlinkToDir(dirFS, "tests/fixtures", dir)).To(BeFalse())
		})
	})

	Describe("isDirIgnored", func() {
		baseDir := filepath.Join("tests", "fixtures")
		It("returns false for normal dirs", func() {
			dir, _ := fs.Stat(dirFS, filepath.Join(baseDir, "empty_folder"))
			Expect(isDirIgnored(dirFS, baseDir, dir)).To(BeFalse())
		})
		It("returns true when folder contains .ndignore file", func() {
			dir, _ := fs.Stat(dirFS, filepath.Join(baseDir, "ignored_folder"))
			Expect(isDirIgnored(dirFS, baseDir, dir)).To(BeTrue())
		})
		It("returns true when folder name starts with a `.`", func() {
			dir, _ := fs.Stat(dirFS, filepath.Join(baseDir, ".hidden_folder"))
			Expect(isDirIgnored(dirFS, baseDir, dir)).To(BeTrue())
		})
	})
})
