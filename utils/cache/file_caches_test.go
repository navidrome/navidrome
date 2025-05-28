package cache

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Call NewFileCache and wait for it to be ready
func callNewFileCache(name, cacheSize, cacheFolder string, maxItems int, getReader ReadFunc) *fileCache {
	fc := NewFileCache(name, cacheSize, cacheFolder, maxItems, getReader).(*fileCache)
	Eventually(func() bool { return fc.ready.Load() }).Should(BeTrue())
	return fc
}

var _ = Describe("File Caches", func() {
	BeforeEach(func() {
		tmpDir, _ := os.MkdirTemp("", "file_caches")
		DeferCleanup(func() {
			configtest.SetupConfig()
			_ = os.RemoveAll(tmpDir)
		})
		conf.Server.CacheFolder = tmpDir
	})

	Describe("NewFileCache", func() {
		It("creates the cache folder", func() {
			Expect(callNewFileCache("test", "1k", "test", 0, nil)).ToNot(BeNil())

			_, err := os.Stat(filepath.Join(conf.Server.CacheFolder, "test"))
			Expect(os.IsNotExist(err)).To(BeFalse())
		})

		It("creates the cache folder with invalid size", func() {
			fc := callNewFileCache("test", "abc", "test", 0, nil)
			Expect(fc.cache).ToNot(BeNil())
			Expect(fc.disabled).To(BeFalse())
		})

		It("returns empty if cache size is '0'", func() {
			fc := callNewFileCache("test", "0", "test", 0, nil)
			Expect(fc.cache).To(BeNil())
			Expect(fc.disabled).To(BeTrue())
		})

		It("reports when cache is disabled", func() {
			fc := callNewFileCache("test", "0", "test", 0, nil)
			Expect(fc.Disabled(context.Background())).To(BeTrue())
			fc = callNewFileCache("test", "1KB", "test", 0, nil)
			Expect(fc.Disabled(context.Background())).To(BeFalse())
		})
	})

	Describe("FileCache", func() {
		It("caches data if cache is enabled", func() {
			called := false
			fc := callNewFileCache("test", "1KB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
				called = true
				return strings.NewReader(arg.Key()), nil
			})
			// First call is a MISS
			s, err := fc.Get(context.Background(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(s.Cached).To(BeFalse())
			Expect(s.Closer).To(BeNil())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))

			// Second call is a HIT
			called = false
			s, err = fc.Get(context.Background(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))
			Expect(s.Cached).To(BeTrue())
			Expect(s.Closer).ToNot(BeNil())
			Expect(called).To(BeFalse())
		})

		It("does not cache data if cache is disabled", func() {
			called := false
			fc := callNewFileCache("test", "0", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
				called = true
				return strings.NewReader(arg.Key()), nil
			})
			// First call is a MISS
			s, err := fc.Get(context.Background(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(s.Cached).To(BeFalse())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))

			// Second call is also a MISS
			called = false
			s, err = fc.Get(context.Background(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))
			Expect(s.Cached).To(BeFalse())
			Expect(called).To(BeTrue())
		})

		Context("reader errors", func() {
			When("creating a reader fails", func() {
				It("does not cache", func() {
					fc := callNewFileCache("test", "1KB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
						return nil, errors.New("failed")
					})

					_, err := fc.Get(context.Background(), &testArg{"test"})
					Expect(err).To(MatchError("failed"))
				})
			})
			When("reader returns error", func() {
				It("does not cache", func() {
					fc := callNewFileCache("test", "1KB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
						return errFakeReader{errors.New("read failure")}, nil
					})

					s, err := fc.Get(context.Background(), &testArg{"test"})
					Expect(err).ToNot(HaveOccurred())
					_, _ = io.Copy(io.Discard, s)
					// TODO How to make the fscache reader return the underlying reader error?
					//Expect(err).To(MatchError("read failure"))

					// Data should not be cached (or eventually be removed from cache)
					Eventually(func() bool {
						s, _ = fc.Get(context.Background(), &testArg{"test"})
						if s != nil {
							return s.Cached
						}
						return false
					}).Should(BeFalse())
				})
			})
		})
	})
})

type testArg struct{ s string }

func (t *testArg) Key() string { return t.s }

type errFakeReader struct{ err error }

func (e errFakeReader) Read([]byte) (int, error) { return 0, e.err }
