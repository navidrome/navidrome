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

		fsCache, err = fscache.NewCacheWithHaunter(fs, fscache.NewLRUHaunterStrategy(
			cache.NewFileHaunter("", maxItems, maxSize, 300*time.Millisecond),
		))
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(fsCache.Clean)

		Expect(createTestFiles(fsCache)).To(Succeed())

		<-time.After(400 * time.Millisecond)
	})

	Context("When maxSize is defined", func() {
		BeforeEach(func() {
			maxSize = 20
		})

		It("removes files", func() {
			Expect(os.ReadDir(cacheDir)).To(HaveLen(4))
			Expect(fsCache.Exists("stream-5")).To(BeFalse(), "stream-5 (empty file) should have been scrubbed")
			// TODO Fix flaky tests
			//Expect(fsCache.Exists("stream-0")).To(BeFalse(), "stream-0 should have been scrubbed")
		})
	})

	XContext("When maxItems is defined", func() {
		BeforeEach(func() {
			maxItems = 3
		})

		It("removes files", func() {
			Expect(os.ReadDir(cacheDir)).To(HaveLen(maxItems))
			Expect(fsCache.Exists("stream-5")).To(BeFalse(), "stream-5 (empty file) should have been scrubbed")
			// TODO Fix flaky tests
			//Expect(fsCache.Exists("stream-0")).To(BeFalse(), "stream-0 should have been scrubbed")
			//Expect(fsCache.Exists("stream-1")).To(BeFalse(), "stream-1 should have been scrubbed")
		})
	})
})

func createTestFiles(c *fscache.FSCache) error {
	// Create 5 normal files and 1 empty
	for i := 0; i < 6; i++ {
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

func createCachedStream(c *fscache.FSCache, name string, contents string) fscache.ReadAtCloser {
	r, w, _ := c.Get(name)
	_, _ = w.Write([]byte(contents))
	_ = w.Close()
	_, _ = io.Copy(io.Discard, r)
	return r
}
