package cmd

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("readTargetsFromFile", func() {
	var tempDir string

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "navidrome-test-")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	It("reads valid targets from file", func() {
		filePath := filepath.Join(tempDir, "targets.txt")
		content := "1:Music/Rock\n2:Music/Jazz\n3:Classical\n"
		err := os.WriteFile(filePath, []byte(content), 0600)
		Expect(err).ToNot(HaveOccurred())

		targets, err := readTargetsFromFile(filePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(3))
		Expect(targets[0]).To(Equal(model.ScanTarget{LibraryID: 1, FolderPath: "Music/Rock"}))
		Expect(targets[1]).To(Equal(model.ScanTarget{LibraryID: 2, FolderPath: "Music/Jazz"}))
		Expect(targets[2]).To(Equal(model.ScanTarget{LibraryID: 3, FolderPath: "Classical"}))
	})

	It("skips empty lines", func() {
		filePath := filepath.Join(tempDir, "targets.txt")
		content := "1:Music/Rock\n\n2:Music/Jazz\n\n"
		err := os.WriteFile(filePath, []byte(content), 0600)
		Expect(err).ToNot(HaveOccurred())

		targets, err := readTargetsFromFile(filePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
	})

	It("trims whitespace", func() {
		filePath := filepath.Join(tempDir, "targets.txt")
		content := "  1:Music/Rock  \n\t2:Music/Jazz\t\n"
		err := os.WriteFile(filePath, []byte(content), 0600)
		Expect(err).ToNot(HaveOccurred())

		targets, err := readTargetsFromFile(filePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
		Expect(targets[0].FolderPath).To(Equal("Music/Rock"))
		Expect(targets[1].FolderPath).To(Equal("Music/Jazz"))
	})

	It("returns error for non-existent file", func() {
		_, err := readTargetsFromFile("/nonexistent/file.txt")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("failed to open target file"))
	})

	It("returns error for invalid target format", func() {
		filePath := filepath.Join(tempDir, "targets.txt")
		content := "invalid-format\n"
		err := os.WriteFile(filePath, []byte(content), 0600)
		Expect(err).ToNot(HaveOccurred())

		_, err = readTargetsFromFile(filePath)
		Expect(err).To(HaveOccurred())
	})

	It("handles mixed valid and empty lines", func() {
		filePath := filepath.Join(tempDir, "targets.txt")
		content := "\n1:Music/Rock\n\n\n2:Music/Jazz\n\n"
		err := os.WriteFile(filePath, []byte(content), 0600)
		Expect(err).ToNot(HaveOccurred())

		targets, err := readTargetsFromFile(filePath)
		Expect(err).ToNot(HaveOccurred())
		Expect(targets).To(HaveLen(2))
	})
})
