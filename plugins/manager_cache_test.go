package plugins

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/dustin/go-humanize"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("purgeCacheBySize", func() {
	var (
		tmpDir string
		ctx    context.Context
	)

	BeforeEach(func() {
		var err error
		ctx = GinkgoT().Context()
		tmpDir, err = os.MkdirTemp("", "cache-purge-test-*")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})

	createFileWithSize := func(path string, sizeBytes int64, modTime time.Time) {
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, 0755)
		Expect(err).ToNot(HaveOccurred())

		f, err := os.Create(path)
		Expect(err).ToNot(HaveOccurred())
		defer f.Close()

		// Write random data to reach desired size
		if sizeBytes > 0 {
			err = f.Truncate(sizeBytes)
			Expect(err).ToNot(HaveOccurred())
		}

		// Set modification time
		err = os.Chtimes(path, modTime, modTime)
		Expect(err).ToNot(HaveOccurred())
	}

	getDirSize := func(dir string) uint64 {
		var total uint64
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			total += uint64(info.Size())
			return nil
		})
		Expect(err).ToNot(HaveOccurred())
		return total
	}

	Context("when maxSize is invalid or zero", func() {
		It("should not remove any files with invalid size", func() {
			cacheDir := filepath.Join(tmpDir, "cache")
			createFileWithSize(filepath.Join(cacheDir, "file1.bin"), 1000, time.Now())
			createFileWithSize(filepath.Join(cacheDir, "file2.bin"), 1000, time.Now())

			purgeCacheBySize(ctx, cacheDir, "invalid")

			Expect(getDirSize(cacheDir)).To(Equal(uint64(2000)))
		})

		It("should not remove any files when maxSize is 0", func() {
			cacheDir := filepath.Join(tmpDir, "cache")
			createFileWithSize(filepath.Join(cacheDir, "file1.bin"), 1000, time.Now())
			createFileWithSize(filepath.Join(cacheDir, "file2.bin"), 1000, time.Now())

			purgeCacheBySize(ctx, cacheDir, "0")

			Expect(getDirSize(cacheDir)).To(Equal(uint64(2000)))
		})
	})

	Context("when cache directory doesn't exist", func() {
		It("should not error", func() {
			nonExistentDir := filepath.Join(tmpDir, "nonexistent")
			Expect(func() {
				purgeCacheBySize(ctx, nonExistentDir, "100MB")
			}).ToNot(Panic())
		})
	})

	Context("when total size is under limit", func() {
		It("should not remove any files", func() {
			cacheDir := filepath.Join(tmpDir, "cache")
			createFileWithSize(filepath.Join(cacheDir, "file1.bin"), 1000, time.Now())
			createFileWithSize(filepath.Join(cacheDir, "file2.bin"), 1000, time.Now())

			purgeCacheBySize(ctx, cacheDir, "10KB")

			Expect(getDirSize(cacheDir)).To(Equal(uint64(2000)))
		})
	})

	Context("when total size exceeds limit", func() {
		It("should remove oldest files first", func() {
			cacheDir := filepath.Join(tmpDir, "cache")
			now := time.Now()

			// Create files with different ages (1MB each)
			oldestFile := filepath.Join(cacheDir, "old.bin")
			middleFile := filepath.Join(cacheDir, "middle.bin")
			newestFile := filepath.Join(cacheDir, "new.bin")

			createFileWithSize(oldestFile, 1*1024*1024, now.Add(-3*time.Hour))
			createFileWithSize(middleFile, 1*1024*1024, now.Add(-2*time.Hour))
			createFileWithSize(newestFile, 1*1024*1024, now.Add(-1*time.Hour))

			// Set limit to 2MiB - should remove oldest file
			purgeCacheBySize(ctx, cacheDir, "2MiB")

			// Oldest should be removed
			_, err := os.Stat(oldestFile)
			Expect(os.IsNotExist(err)).To(BeTrue(), "oldest file should be removed")

			// Others should remain
			_, err = os.Stat(middleFile)
			Expect(err).ToNot(HaveOccurred(), "middle file should remain")

			_, err = os.Stat(newestFile)
			Expect(err).ToNot(HaveOccurred(), "newest file should remain")
		})

		It("should remove multiple files to get under limit", func() {
			cacheDir := filepath.Join(tmpDir, "cache")
			now := time.Now()

			// Create 5 files, 1MiB each (total 5MiB)
			for i := 0; i < 5; i++ {
				path := filepath.Join(cacheDir, filepath.Join("dir", "file"+string(rune('0'+i))+".bin"))
				createFileWithSize(path, 1*1024*1024, now.Add(-time.Duration(5-i)*time.Hour))
			}

			// Set limit to 2.5MiB - should remove oldest 3 files (leaving 2MiB)
			purgeCacheBySize(ctx, cacheDir, "2.5MiB")

			finalSize := getDirSize(cacheDir)
			limit, _ := humanize.ParseBytes("2.5MiB")
			Expect(finalSize).To(BeNumerically("<=", limit))
		})

		It("should remove empty parent directories after removing files", func() {
			cacheDir := filepath.Join(tmpDir, "cache")
			now := time.Now()

			// Create files in subdirectories
			oldFile := filepath.Join(cacheDir, "subdir1", "old.bin")
			newFile := filepath.Join(cacheDir, "subdir2", "new.bin")

			createFileWithSize(oldFile, 2*1024*1024, now.Add(-2*time.Hour))
			createFileWithSize(newFile, 2*1024*1024, now.Add(-1*time.Hour))

			// Set limit to 2MiB - should remove old file and its parent dir
			purgeCacheBySize(ctx, cacheDir, "2MiB")

			// Old file and its parent dir should be removed
			_, err := os.Stat(oldFile)
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Stat(filepath.Join(cacheDir, "subdir1"))
			Expect(os.IsNotExist(err)).To(BeTrue(), "empty parent directory should be removed")

			// New file and its parent dir should remain
			_, err = os.Stat(newFile)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(filepath.Join(cacheDir, "subdir2"))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
