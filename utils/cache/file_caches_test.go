package cache

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/utils/cache/item"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Call NewFileCache and wait for it to be ready
func callNewFileCache(name, cacheSize, cacheFolder string, maxItems int, getReader ReadFunc) *fileCache {
	fc := NewFileCache(name, cacheSize, cacheFolder, maxItems, getReader)
	Eventually(func() bool { return fc.Ready(context.TODO()) }).Should(BeTrue())
	return fc
}

var _ = Describe("File Caches", func() {
	BeforeEach(func() {
		conf.Server.DataFolder, _ = os.MkdirTemp("", "file_caches")
	})
	AfterEach(func() {
		_ = os.RemoveAll(conf.Server.DataFolder)
	})

	Describe("NewFileCache", func() {
		It("creates the cache folder", func() {
			Expect(callNewFileCache("test", "1k", "test", 0, nil)).ToNot(BeNil())

			_, err := os.Stat(filepath.Join(conf.Server.DataFolder, "test"))
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
	})

	Describe("FileCache", func() {
		It("caches data if cache is enabled", func() {
			called := false
			fc := callNewFileCache("test", "1KB", "test", 0, func(ctx context.Context, arg item.Item) (io.Reader, error) {
				called = true
				return strings.NewReader(arg.Key()), nil
			})
			// First call is a MISS
			s, err := fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(s.Cached).To(BeFalse())
			Expect(s.Closer).To(BeNil())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))

			// Second call is a HIT
			called = false
			s, err = fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))
			Expect(s.Cached).To(BeTrue())
			Expect(s.Closer).ToNot(BeNil())
			Expect(called).To(BeFalse())
		})

		It("does not cache data if cache is disabled", func() {
			called := false
			fc := callNewFileCache("test", "0", "test", 0, func(ctx context.Context, arg item.Item) (io.Reader, error) {
				called = true
				return strings.NewReader(arg.Key()), nil
			})
			// First call is a MISS
			s, err := fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(s.Cached).To(BeFalse())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))

			// Second call is also a MISS
			called = false
			s, err = fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))
			Expect(s.Cached).To(BeFalse())
			Expect(called).To(BeTrue())
		})
	})

	Describe("InvalidateCache", func() {
		It("caches data if cache is enabled", func() {
			called := false
			fc := callNewFileCache("test", "1KB", "test", 0, func(ctx context.Context, arg item.Item) (io.Reader, error) {
				called = true
				return strings.NewReader(arg.Key()), nil
			})
			// First call is a MISS
			s1, err := fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(s1.Cached).To(BeFalse())
			Expect(s1.Closer).To(BeNil())
			Expect(io.ReadAll(s1)).To(Equal([]byte("test")))

			// Second call is a HIT
			called = false
			s2, err := fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(io.ReadAll(s2)).To(Equal([]byte("test")))
			Expect(s2.Cached).To(BeTrue())
			Expect(s2.Closer).ToNot(BeNil())
			Expect(called).To(BeFalse())

			// Third call, invalidate
			err = s1.Close()
			Expect(err).To(BeNil())
			err = s2.Close()
			Expect(err).To(BeNil())
			err = fc.Invalidate(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())

			// Fourth call is MISS again
			s3, err := fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(s3.Cached).To(BeFalse())
			Expect(s3.Closer).To(BeNil())
			Expect(io.ReadAll(s3)).To(Equal([]byte("test")))
		})

		It("does not cache data if cache is disabled", func() {
			called := false
			fc := callNewFileCache("test", "0", "test", 0, func(ctx context.Context, arg item.Item) (io.Reader, error) {
				called = true
				return strings.NewReader(arg.Key()), nil
			})
			// First call is a MISS
			s, err := fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(s.Cached).To(BeFalse())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))

			// Second call is also a MISS
			called = false
			s, err = fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))
			Expect(s.Cached).To(BeFalse())
			Expect(called).To(BeTrue())

			// Third call, invalidate
			err = fc.Invalidate(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())

			// Fourth call is MISS again
			s, err = fc.Get(context.TODO(), &testArg{"test"})
			Expect(err).To(BeNil())
			Expect(s.Cached).To(BeFalse())
			Expect(io.ReadAll(s)).To(Equal([]byte("test")))
		})
	})
})

type testArg struct{ s string }

func (t *testArg) Key() string { return t.s }
