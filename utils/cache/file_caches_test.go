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
				return pr, nil // slow, still-being-produced stream
			})

			// First Get → MISS; the cache starts copying pr into the entry in a goroutine.
			s1, err := fc.Get(context.Background(), &testArg{"live"})
			Expect(err).To(BeNil())

			// Write the first chunk so the entry exists with in-flight bytes,
			// but leave the pipe open so the second reader can attach mid-stream.
			// io.Pipe writes block until the cache goroutine reads them, giving us
			// a deterministic happens-before: the entry is live before we call Get again.
			_, err = pw.Write([]byte("hello "))
			Expect(err).To(BeNil())

			// Second Get while the pipe is still open → attaches to the in-progress entry.
			s2, err := fc.Get(context.Background(), &testArg{"live"})
			Expect(err).To(BeNil())

			// Drain both readers concurrently; they race against the producer below.
			ch1 := make(chan []byte, 1)
			ch2 := make(chan []byte, 1)
			go func() { b, _ := io.ReadAll(s1); ch1 <- b }()
			go func() { b, _ := io.ReadAll(s2); ch2 <- b }()

			// Deliver the rest of the stream and close; both draining goroutines must see it.
			_, err = pw.Write([]byte("world"))
			Expect(err).To(BeNil())
			Expect(pw.Close()).To(Succeed())

			Expect(string(<-ch1)).To(Equal("hello world"))
			Expect(string(<-ch2)).To(Equal("hello world"))
			_ = s1.Close()
			_ = s2.Close()

			// Exactly one completion marker must appear.
			dataPath := fcSpreadFS(fc).KeyMapper((&testArg{"live"}).Key())
			Eventually(func() bool {
				_, e := os.Stat(dataPath + ".complete")
				return e == nil
			}).Should(BeTrue())

			// Steady-state HIT: full data, Cached flag set.
			s3, err := fc.Get(context.Background(), &testArg{"live"})
			Expect(err).To(BeNil())
			got3, _ := io.ReadAll(s3)
			_ = s3.Close()
			Expect(s3.Cached).To(BeTrue())
			Expect(string(got3)).To(Equal("hello world"))
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

			It("does not write a completion marker when the write fails after partial bytes", func() {
				// Mimics a transcode that produces real output and then dies:
				// the bytes land on disk, but the entry must NOT be marked complete.
				fc := callNewFileCache("test", "10MB", "test", 0, func(ctx context.Context, arg Item) (io.Reader, error) {
					return &partialThenErrReader{data: []byte("PARTIAL-OUTPUT"), err: errors.New("transcoder died")}, nil
				})
				s, err := fc.Get(context.Background(), &testArg{"partial"})
				Expect(err).To(BeNil())
				_, _ = io.Copy(io.Discard, s)
				_ = s.Close()

				dataPath := fcSpreadFS(fc).KeyMapper((&testArg{"partial"}).Key())
				// The marker must never appear for a failed write. Give the async
				// writer time to finish, then assert the marker stays absent.
				Consistently(func() bool {
					_, e := os.Stat(dataPath + ".complete")
					return os.IsNotExist(e)
				}).Should(BeTrue())
			})
		})
	})
})

type testArg struct{ s string }

func (t *testArg) Key() string { return t.s }

type errFakeReader struct{ err error }

func (e errFakeReader) Read([]byte) (int, error) { return 0, e.err }

// partialThenErrReader emits data once, then fails — mimicking a transcoder
// that produces some output and then dies mid-stream.
type partialThenErrReader struct {
	data []byte
	err  error
	done bool
}

func (r *partialThenErrReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, r.err
	}
	r.done = true
	return copy(p, r.data), nil
}

func fcSpreadFS(fc *fileCache) *spreadFS {
	return fc.fs
}
