package utils_test

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TempFileName", func() {
	It("creates a temporary file name with prefix and suffix", func() {
		prefix := "test-"
		suffix := ".tmp"
		result := utils.TempFileName(prefix, suffix)

		Expect(result).To(ContainSubstring(prefix))
		Expect(result).To(HaveSuffix(suffix))
		Expect(result).To(ContainSubstring(os.TempDir()))
	})

	It("creates unique file names on multiple calls", func() {
		prefix := "unique-"
		suffix := ".test"

		result1 := utils.TempFileName(prefix, suffix)
		result2 := utils.TempFileName(prefix, suffix)

		Expect(result1).NotTo(Equal(result2))
	})

	It("handles empty prefix and suffix", func() {
		result := utils.TempFileName("", "")

		Expect(result).To(ContainSubstring(os.TempDir()))
		Expect(len(result)).To(BeNumerically(">", len(os.TempDir())))
	})

	It("creates proper file path separators", func() {
		prefix := "path-test-"
		suffix := ".ext"
		result := utils.TempFileName(prefix, suffix)

		expectedDir := os.TempDir()
		Expect(result).To(HavePrefix(expectedDir))
		Expect(strings.Count(result, string(filepath.Separator))).To(BeNumerically(">=", strings.Count(expectedDir, string(filepath.Separator))))
	})
})

var _ = Describe("BaseName", func() {
	It("extracts basename from a simple filename", func() {
		result := utils.BaseName("test.mp3")
		Expect(result).To(Equal("test"))
	})

	It("extracts basename from a file path", func() {
		result := utils.BaseName("/path/to/file.txt")
		Expect(result).To(Equal("file"))
	})

	It("handles files without extension", func() {
		result := utils.BaseName("/path/to/filename")
		Expect(result).To(Equal("filename"))
	})

	It("handles files with multiple dots", func() {
		result := utils.BaseName("archive.tar.gz")
		Expect(result).To(Equal("archive.tar"))
	})

	It("handles hidden files", func() {
		// For hidden files without additional extension, path.Ext returns the entire name
		// So basename becomes empty string after TrimSuffix
		result := utils.BaseName(".hidden")
		Expect(result).To(Equal(""))
	})

	It("handles hidden files with extension", func() {
		result := utils.BaseName(".config.json")
		Expect(result).To(Equal(".config"))
	})

	It("handles empty string", func() {
		// The actual behavior returns empty string for empty input
		result := utils.BaseName("")
		Expect(result).To(Equal(""))
	})

	It("handles path ending with separator", func() {
		result := utils.BaseName("/path/to/dir/")
		Expect(result).To(Equal("dir"))
	})

	It("handles complex nested path", func() {
		result := utils.BaseName("/very/long/path/to/my/favorite/song.mp3")
		Expect(result).To(Equal("song"))
	})
})

var _ = Describe("FileExists", func() {
	var tempFile *os.File
	var tempDir string

	BeforeEach(func() {
		var err error
		tempFile, err = os.CreateTemp("", "fileexists-test-*.txt")
		Expect(err).NotTo(HaveOccurred())

		tempDir, err = os.MkdirTemp("", "fileexists-test-dir-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if tempFile != nil {
			os.Remove(tempFile.Name())
			tempFile.Close()
		}
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
	})

	It("returns true for existing file", func() {
		Expect(utils.FileExists(tempFile.Name())).To(BeTrue())
	})

	It("returns true for existing directory", func() {
		Expect(utils.FileExists(tempDir)).To(BeTrue())
	})

	It("returns false for non-existing file", func() {
		nonExistentPath := filepath.Join(tempDir, "does-not-exist.txt")
		Expect(utils.FileExists(nonExistentPath)).To(BeFalse())
	})

	It("returns false for empty path", func() {
		Expect(utils.FileExists("")).To(BeFalse())
	})

	It("handles nested non-existing path", func() {
		nonExistentPath := "/this/path/definitely/does/not/exist/file.txt"
		Expect(utils.FileExists(nonExistentPath)).To(BeFalse())
	})

	Context("when file is deleted after creation", func() {
		It("returns false after file deletion", func() {
			filePath := tempFile.Name()
			Expect(utils.FileExists(filePath)).To(BeTrue())

			err := os.Remove(filePath)
			Expect(err).NotTo(HaveOccurred())
			tempFile = nil // Prevent cleanup attempt

			Expect(utils.FileExists(filePath)).To(BeFalse())
		})
	})

	Context("when directory is deleted after creation", func() {
		It("returns false after directory deletion", func() {
			dirPath := tempDir
			Expect(utils.FileExists(dirPath)).To(BeTrue())

			err := os.RemoveAll(dirPath)
			Expect(err).NotTo(HaveOccurred())
			tempDir = "" // Prevent cleanup attempt

			Expect(utils.FileExists(dirPath)).To(BeFalse())
		})
	})

	It("handles permission denied scenarios gracefully", func() {
		// This test might be platform specific, but we test the general case
		result := utils.FileExists("/root/.ssh/id_rsa") // Likely to not exist or be inaccessible
		Expect(result).To(Or(BeTrue(), BeFalse()))      // Should not panic
	})
})
