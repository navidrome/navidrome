package scanner

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("load_tree", func() {
	Describe("isDirOrSymlinkToDir", func() {
		It("returns true for normal dirs", func() {
			dir, _ := os.Stat("tests/fixtures")
			Expect(isDirOrSymlinkToDir("tests", dir)).To(BeTrue())
		})
		It("returns true for symlinks to dirs", func() {
			dir, _ := os.Stat("tests/fixtures/symlink2dir")
			Expect(isDirOrSymlinkToDir("tests/fixtures", dir)).To(BeTrue())
		})
		It("returns false for files", func() {
			dir, _ := os.Stat("tests/fixtures/test.mp3")
			Expect(isDirOrSymlinkToDir("tests/fixtures", dir)).To(BeFalse())
		})
		It("returns false for symlinks to files", func() {
			dir, _ := os.Stat("tests/fixtures/symlink")
			Expect(isDirOrSymlinkToDir("tests/fixtures", dir)).To(BeFalse())
		})
	})

	Describe("isDirIgnored", func() {
		baseDir := filepath.Join("tests", "fixtures")
		It("returns false for normal dirs", func() {
			dir, _ := os.Stat(filepath.Join(baseDir, "empty_folder"))
			Expect(isDirIgnored(baseDir, dir)).To(BeFalse())
		})
		It("returns true when folder contains .ndignore file", func() {
			dir, _ := os.Stat(filepath.Join(baseDir, "ignored_folder"))
			Expect(isDirIgnored(baseDir, dir)).To(BeTrue())
		})
	})
})
