package tagwriter

import (
	"os"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TagWriter", func() {
	var tw TagWriter
	var testDir string

	BeforeEach(func() {
		tw = New()
		conf.Server.EnableTagEditing = true
		var err error
		testDir, err = os.MkdirTemp("", "tagwriter-test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(testDir)
		conf.Server.EnableTagEditing = false
		ClearLocks()
	})

	Describe("WriteTags", func() {
		It("returns error when feature is disabled", func() {
			conf.Server.EnableTagEditing = false
			err := tw.WriteTags("test.mp3", Tags{"title": "Test"})
			Expect(err).To(Equal(ErrFeatureDisabled))
		})

		It("returns error for unsupported formats", func() {
			testFile := filepath.Join(testDir, "test.ogg")
			f, err := os.Create(testFile)
			Expect(err).NotTo(HaveOccurred())
			f.Close()

			err = tw.WriteTags(testFile, Tags{"title": "Test"})
			Expect(err).To(Equal(ErrUnsupportedFormat))
		})

		It("returns error for non-existent file", func() {
			err := tw.WriteTags("/nonexistent/path/test.mp3", Tags{"title": "Test"})
			Expect(err).To(HaveOccurred())
		})

		It("returns error for read-only file", func() {
			testFile := filepath.Join(testDir, "readonly.mp3")
			f, err := os.Create(testFile)
			Expect(err).NotTo(HaveOccurred())
			f.Close()
			os.Chmod(testFile, 0444)

			err = tw.WriteTags(testFile, Tags{"title": "Test"})
			Expect(err).To(Equal(ErrReadOnlyFile))

			os.Chmod(testFile, 0644)
		})

		It("returns error for directory", func() {
			err := tw.WriteTags(testDir, Tags{"title": "Test"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("directory"))
		})

		It("returns no error for empty tags", func() {
			testFile := filepath.Join(testDir, "test.mp3")
			f, err := os.Create(testFile)
			Expect(err).NotTo(HaveOccurred())
			f.Close()

			err = tw.WriteTags(testFile, Tags{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("SupportedFormats", func() {
		It("returns supported formats", func() {
			formats := SupportedFormats()
			Expect(formats).To(ContainElements(".mp3", ".mp2", ".flac"))
		})
	})

	Describe("IsSupportedFormat", func() {
		It("returns true for supported formats", func() {
			Expect(IsSupportedFormat("test.mp3")).To(BeTrue())
			Expect(IsSupportedFormat("test.MP3")).To(BeTrue())
			Expect(IsSupportedFormat("test.flac")).To(BeTrue())
			Expect(IsSupportedFormat("test.FLAC")).To(BeTrue())
		})

		It("returns false for unsupported formats", func() {
			Expect(IsSupportedFormat("test.ogg")).To(BeFalse())
			Expect(IsSupportedFormat("test.wav")).To(BeFalse())
			Expect(IsSupportedFormat("test.m4a")).To(BeFalse())
		})
	})

	Describe("File Locking", func() {
		It("acquires and releases lock", func() {
			testFile := filepath.Join(testDir, "locktest.mp3")
			f, err := os.Create(testFile)
			Expect(err).NotTo(HaveOccurred())
			f.Close()

			lock, err := LockFile(testFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(lock).NotTo(BeNil())

			err = UnlockFile(lock)
			Expect(err).NotTo(HaveOccurred())
		})

		It("allows multiple locks from same process", func() {
			testFile := filepath.Join(testDir, "multilock.mp3")
			f, err := os.Create(testFile)
			Expect(err).NotTo(HaveOccurred())
			f.Close()

			lock1, err := LockFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			lock2, err := LockFile(testFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(lock1).To(Equal(lock2))

			err = UnlockFile(lock1)
			Expect(err).NotTo(HaveOccurred())

			err = UnlockFile(lock2)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns error for non-existent file in LockFile", func() {
			_, err := LockFile("/nonexistent/file.mp3")
			Expect(err).To(HaveOccurred())
		})
	})
})