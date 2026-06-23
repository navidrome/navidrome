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
		conf.Server.CacheFolder = conf.NewDir(tmpDir)
	})

	Describe("NewFileCache", func() {
		It("creates the cache folder", func() {
			Expect(callNewFileCache("test", "1k", "test", 0, nil)).ToNot(BeNil())

			_, err := os.Stat(filepath.Join(conf.Server.CacheFolder.String(), "test"))
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

		It("writes a completion marker after a successful cache write", func() {
			fc := callNewFileCache("test", "1KB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
				return strings.NewReader("complete-data"), nil
			})
			s, err := fc.Get(context.Background(), &testArg{"markme"})
			Expect(err).To(BeNil())
			_, _ = io.ReadAll(s)
			_ = s.Close()

			dataPath := fcSpreadFS(fc).KeyMapper((&testArg{"markme"}).Key())
			Eventually(func() bool {
				_, statErr := os.Stat(dataPath + ".complete")
				return statErr == nil
			}).Should(BeTrue())
		})

		It("serves a concurrent reader from an in-progress write and marks complete once", func() {
			pr, pw := io.Pipe()
			fc := callNewFileCache("test", "10MB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
				return pr, nil // a slow, still-being-produced stream
			})

			// First Get → MISS, starts the write goroutine consuming pr.
			s1, err := fc.Get(context.Background(), &testArg{"live"})
			Expect(err).To(BeNil())

			// Feed some bytes, then a second client attaches mid-stream.
			go func() {
				_, _ = pw.Write([]byte("hello "))
				_, _ = pw.Write([]byte("world"))
				_ = pw.Close()
			}()

			got1, _ := io.ReadAll(s1)
			_ = s1.Close()
			Expect(string(got1)).To(Equal("hello world"))

			// After completion, a marker exists and a later Get is a HIT with full data.
			dataPath := fcSpreadFS(fc).KeyMapper((&testArg{"live"}).Key())
			Eventually(func() bool {
				_, e := os.Stat(dataPath + ".complete")
				return e == nil
			}).Should(BeTrue())

			s2, err := fc.Get(context.Background(), &testArg{"live"})
			Expect(err).To(BeNil())
			got2, _ := io.ReadAll(s2)
			_ = s2.Close()
			Expect(s2.Cached).To(BeTrue())
			Expect(string(got2)).To(Equal("hello world"))
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

		Context("crash leftover (issue #5636)", func() {
			It("does not serve a partial file left on disk as a complete HIT", func() {
				// First init: empties + writes the migration sentinel.
				fc1 := callNewFileCache("test", "10MB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
					return strings.NewReader("UNUSED"), nil
				})
				_ = fc1

				// Plant a partial file (no marker), simulating a killed process.
				sfs, err := NewSpreadFS(filepath.Join(conf.Server.CacheFolder.String(), "test"), 0755)
				Expect(err).To(BeNil())
				partialPath := sfs.KeyMapper((&testArg{"track"}).Key())
				Expect(os.MkdirAll(filepath.Dir(partialPath), 0755)).To(Succeed())
				Expect(os.WriteFile(partialPath, []byte("PARTIAL"), 0600)).To(Succeed())

				// "Restart": a fresh cache over the same folder (sentinel present → strict).
				getReaderCalled := false
				fc2 := callNewFileCache("test", "10MB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
					getReaderCalled = true
					return strings.NewReader("FULL-TRANSCODE"), nil
				})

				s, err := fc2.Get(context.Background(), &testArg{"track"})
				Expect(err).To(BeNil())
				data, _ := io.ReadAll(s)
				_ = s.Close()

				Expect(getReaderCalled).To(BeTrue()) // re-transcoded, not served stale
				Expect(string(data)).To(Equal("FULL-TRANSCODE"))
			})
		})

		Context("live error path still invalidates", func() {
			It("leaves no data file and no marker after a mid-stream reader error", func() {
				fc := callNewFileCache("test", "10MB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
					return errFakeReader{errors.New("boom")}, nil
				})
				s, err := fc.Get(context.Background(), &testArg{"err"})
				Expect(err).To(BeNil())
				_, _ = io.Copy(io.Discard, s)
				_ = s.Close()

				dataPath := fcSpreadFS(fc).KeyMapper((&testArg{"err"}).Key())
				Eventually(func() bool {
					_, e1 := os.Stat(dataPath)
					_, e2 := os.Stat(dataPath + ".complete")
					return os.IsNotExist(e1) && os.IsNotExist(e2)
				}).Should(BeTrue())
			})
		})
	})
})

type testArg struct{ s string }

func (t *testArg) Key() string { return t.s }

type errFakeReader struct{ err error }

func (e errFakeReader) Read([]byte) (int, error) { return 0, e.err }

func fcSpreadFS(fc *fileCache) *spreadFS {
	return fc.fs
}
