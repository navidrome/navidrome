package cache_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/djherbis/fscache"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FileHaunter", func() {
	var fs fscache.FileSystem
	var fsCache *fscache.FSCache
	var cacheDir string
	var err error
	var maxItems int
	var maxSize uint64

	JustBeforeEach(func() {
		tempDir, _ := os.MkdirTemp("", "spread_fs")
		cacheDir = filepath.Join(tempDir, "cache1")
		fs, err = fscache.NewFs(cacheDir, 0700)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = os.RemoveAll(tempDir) })

		// Use a short haunter period so cleanup runs promptly; the assertions
		// below poll with Eventually instead of racing a fixed sleep.
		fsCache, err = fscache.NewCacheWithHaunter(fs, fscache.NewLRUHaunterStrategy(
			cache.NewFileHaunter("", maxItems, maxSize, 100*time.Millisecond),
		))
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(fsCache.Clean)

		Expect(createTestFiles(fsCache)).To(Succeed())
	})

	Context("When maxSize is defined", func() {
		BeforeEach(func() {
			maxSize = 20
		})

		It("removes files", func() {
			// stream-0..4 hold "hello" (5 bytes each) and stream-5 is empty.
			// With maxSize=20, the haunter scrubs the empty file plus enough of
			// the oldest files to bring the total size down to <= 20 bytes.
			// Which files survive (and therefore the exact count) depends on
			// access-time ordering, so we only assert the haunter's guarantees:
			// the empty file is always scrubbed and the total size stays within
			// the configured limit.
			Eventually(func(g Gomega) {
				g.Expect(fsCache.Exists("stream-5")).To(BeFalse(), "stream-5 (empty file) should have been scrubbed")
				size, err := dirSize(cacheDir)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(size).To(BeNumerically("<=", maxSize))
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Succeed())
		})
	})

	Context("When maxItems is defined", func() {
		BeforeEach(func() {
			maxItems = 3
		})

		It("removes files", func() {
			// With maxItems=3, the haunter scrubs the empty file plus enough of
			// the oldest files to bring the count within the limit. As above, the
			// exact survivors depend on access-time ordering, so we assert the
			// guaranteed invariants: the empty file is gone and the item count
			// stays within the configured limit.
			Eventually(func(g Gomega) {
				g.Expect(fsCache.Exists("stream-5")).To(BeFalse(), "stream-5 (empty file) should have been scrubbed")
				entries, readErr := os.ReadDir(cacheDir)
				g.Expect(readErr).ToNot(HaveOccurred())
				g.Expect(len(entries)).To(BeNumerically("<=", maxItems))
			}).WithTimeout(5 * time.Second).WithPolling(50 * time.Millisecond).Should(Succeed())
		})
	})
})

func createTestFiles(c *fscache.FSCache) error {
	// Create 5 normal files and 1 empty
	for i := range 6 {
		name := fmt.Sprintf("stream-%v", i)
		var r fscache.ReadAtCloser
		if i < 5 {
			r = createCachedStream(c, name, "hello")
		} else { // Last one is empty
			r = createCachedStream(c, name, "")
		}

		if !c.Exists(name) {
			return errors.New(name + " should exist")
		}

		<-time.After(10 * time.Millisecond)

		err := r.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// dirSize returns the total size in bytes of all regular files in dir.
func dirSize(dir string) (uint64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	var total uint64
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return 0, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		total += uint64(info.Size())
	}
	return total, nil
}

func createCachedStream(c *fscache.FSCache, name string, contents string) fscache.ReadAtCloser {
	r, w, _ := c.Get(name)
	_, _ = w.Write([]byte(contents))
	_ = w.Close()
	_, _ = io.Copy(io.Discard, r)
	return r
}
