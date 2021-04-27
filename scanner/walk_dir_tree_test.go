package scanner

import (
	"context"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("load_tree", func() {

	Describe("walkDirTree", func() {
		It("reads all info correctly", func() {
			var collected = dirMap{}
			results := make(walkResults, 5000)
			var err error
			go func() {
				err = walkDirTree(context.TODO(), "tests/fixtures", results)
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
			dirents, _ := os.ReadDir("tests")
			for _, d := range dirents {
				if d.Name() == "fixtures" {
					Expect(isDirOrSymlinkToDir("tests", d)).To(BeTrue())
				}
			}
		})
		It("returns true for symlinks to dirs", func() {
			dirents, _ := os.ReadDir("tests/fixtures")
			for _, d := range dirents {
				if d.Name() == "symlink2dir" {
					Expect(isDirOrSymlinkToDir("tests/fixtures", d)).To(BeTrue())
				}
			}
		})
		It("returns false for files", func() {
			dirents, _ := os.ReadDir("tests/fixtures")
			for _, d := range dirents {
				if d.Name() == "test.mp3" {
					Expect(isDirOrSymlinkToDir("tests/fixtures", d)).To(BeFalse())
				}
			}
		})
		It("returns false for symlinks to files", func() {
			dirents, _ := os.ReadDir("tests/fixtures")
			for _, d := range dirents {
				if d.Name() == "symlink" {
					Expect(isDirOrSymlinkToDir("tests/fixtures", d)).To(BeFalse())
				}
			}
		})
	})
	Describe("isDirIgnored", func() {
		It("returns false for normal dirs", func() {
			dirents, _ := os.ReadDir("tests/fixtures")
			for _, d := range dirents {
				if d.Name() == "empty_folder" {
					Expect(isDirIgnored("tests/fixtures", d)).To(BeFalse())
				}
			}
		})
		It("returns true when folder contains .ndignore file", func() {
			dirents, _ := os.ReadDir("tests/fixtures")
			for _, d := range dirents {
				if d.Name() == "ignored_folder" {
					Expect(isDirIgnored("tests/fixtures", d)).To(BeTrue())
				}
			}
		})
		It("returns true when folder name starts with a `.`", func() {
			dirents, _ := os.ReadDir("tests/fixtures")
			for _, d := range dirents {
				if d.Name() == ".hidden_folder" {
					Expect(isDirIgnored("tests/fixtures", d)).To(BeTrue())
				}
			}
		})
		It("returns false when folder name starts with ellipses", func() {
			dirents, _ := os.ReadDir("tests/fixtures")
			for _, d := range dirents {
				if d.Name() == "...unhidden_folder" {
					Expect(isDirIgnored("tests/fixtures", d)).To(BeFalse())
				}
			}
		})
	})
})
