package scanner

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("walk_dir_tree_windows", func() {
	baseDir := filepath.Join("tests", "fixtures")

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
		It("returns true when folder name is $Recycle.Bin", func() {
			dirEntry, _ := getDirEntry(baseDir, "$Recycle.Bin")
			Expect(isDirIgnored(baseDir, dirEntry)).To(BeTrue())
		})
	})
})
