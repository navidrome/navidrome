package artwork

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CacheWarmer", func() {
	var (
		fc *mockFileCache
		aw *mockArtwork
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		fc = &mockFileCache{}
		aw = &mockArtwork{}
	})

	Context("initialization", func() {
		It("returns noop when cache is disabled", func() {
			fc.SetDisabled(true)
			cw := NewCacheWarmer(aw, fc)
			_, ok := cw.(*noopCacheWarmer)
			Expect(ok).To(BeTrue())
		})

		It("returns noop when ImageCacheSize is 0", func() {
			conf.Server.ImageCacheSize = "0"
			cw := NewCacheWarmer(aw, fc)
			_, ok := cw.(*noopCacheWarmer)
			Expect(ok).To(BeTrue())
		})

		It("returns noop when EnableArtworkPrecache is false", func() {
			conf.Server.EnableArtworkPrecache = false
			cw := NewCacheWarmer(aw, fc)
			_, ok := cw.(*noopCacheWarmer)
			Expect(ok).To(BeTrue())
		})

		It("returns real implementation when properly configured", func() {
			conf.Server.ImageCacheSize = "100MB"
			conf.Server.EnableArtworkPrecache = true
			fc.SetDisabled(false)
			cw := NewCacheWarmer(aw, fc)
			_, ok := cw.(*cacheWarmer)
			Expect(ok).To(BeTrue())
		})
	})

	Context("buffer management", func() {
		BeforeEach(func() {
			conf.Server.ImageCacheSize = "100MB"
			conf.Server.EnableArtworkPrecache = true
			fc.SetDisabled(false)
		})

		It("drops buffered items when cache becomes disabled", func() {
			cw := NewCacheWarmer(aw, fc).(*cacheWarmer)
			cw.PreCache(model.MustParseArtworkID("al-test"))
			fc.SetDisabled(true)
			Eventually(func() int {
				cw.mutex.Lock()
				defer cw.mutex.Unlock()
				return len(cw.buffer)
			}).Should(Equal(0))
		})

		It("adds multiple items to buffer", func() {
			fc.SetReady(false) // Make cache unavailable so items stay in buffer
			cw := NewCacheWarmer(aw, fc).(*cacheWarmer)
			cw.PreCache(model.MustParseArtworkID("al-1"))
			cw.PreCache(model.MustParseArtworkID("al-2"))
			cw.mutex.Lock()
			defer cw.mutex.Unlock()
			Expect(len(cw.buffer)).To(Equal(2))
		})

		It("deduplicates items in buffer", func() {
			cw := NewCacheWarmer(aw, fc).(*cacheWarmer)
			cw.PreCache(model.MustParseArtworkID("al-1"))
			cw.PreCache(model.MustParseArtworkID("al-1"))
			cw.mutex.Lock()
			defer cw.mutex.Unlock()
			Expect(len(cw.buffer)).To(Equal(1))
		})
	})

	Context("error handling", func() {
		BeforeEach(func() {
			conf.Server.ImageCacheSize = "100MB"
			conf.Server.EnableArtworkPrecache = true
			fc.SetDisabled(false)
		})

		It("continues processing after artwork retrieval error", func() {
			aw.err = errors.New("artwork error")
			cw := NewCacheWarmer(aw, fc).(*cacheWarmer)
			cw.PreCache(model.MustParseArtworkID("al-error"))
			cw.PreCache(model.MustParseArtworkID("al-1"))

			Eventually(func() int {
				cw.mutex.Lock()
				defer cw.mutex.Unlock()
				return len(cw.buffer)
			}).Should(Equal(0))
		})

		It("continues processing after cache error", func() {
			fc.err = errors.New("cache error")
			cw := NewCacheWarmer(aw, fc).(*cacheWarmer)
			cw.PreCache(model.MustParseArtworkID("al-error"))
			cw.PreCache(model.MustParseArtworkID("al-1"))

			Eventually(func() int {
				cw.mutex.Lock()
				defer cw.mutex.Unlock()
				return len(cw.buffer)
			}).Should(Equal(0))
		})
	})

	Context("background processing", func() {
		BeforeEach(func() {
			conf.Server.ImageCacheSize = "100MB"
			conf.Server.EnableArtworkPrecache = true
			fc.SetDisabled(false)
		})

		It("processes items in batches", func() {
			cw := NewCacheWarmer(aw, fc).(*cacheWarmer)
			for i := 0; i < 5; i++ {
				cw.PreCache(model.MustParseArtworkID(fmt.Sprintf("al-%d", i)))
			}

			Eventually(func() int {
				cw.mutex.Lock()
				defer cw.mutex.Unlock()
				return len(cw.buffer)
			}).Should(Equal(0))
		})

		It("wakes up on new items", func() {
			cw := NewCacheWarmer(aw, fc).(*cacheWarmer)

			// Add first batch
			cw.PreCache(model.MustParseArtworkID("al-1"))
			Eventually(func() int {
				cw.mutex.Lock()
				defer cw.mutex.Unlock()
				return len(cw.buffer)
			}).Should(Equal(0))

			// Add second batch
			cw.PreCache(model.MustParseArtworkID("al-2"))
			Eventually(func() int {
				cw.mutex.Lock()
				defer cw.mutex.Unlock()
				return len(cw.buffer)
			}).Should(Equal(0))
		})
	})
})

type mockArtwork struct {
	err error
}

func (m *mockArtwork) Get(ctx context.Context, artID model.ArtworkID, size int, square bool) (io.ReadCloser, time.Time, error) {
	if m.err != nil {
		return nil, time.Time{}, m.err
	}
	return io.NopCloser(strings.NewReader("test")), time.Now(), nil
}

func (m *mockArtwork) GetOrPlaceholder(ctx context.Context, id string, size int, square bool) (io.ReadCloser, time.Time, error) {
	return m.Get(ctx, model.ArtworkID{}, size, square)
}

type mockFileCache struct {
	disabled atomic.Bool
	ready    atomic.Bool
	err      error
}

func (f *mockFileCache) Get(ctx context.Context, item cache.Item) (*cache.CachedStream, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &cache.CachedStream{Reader: io.NopCloser(strings.NewReader("cached"))}, nil
}

func (f *mockFileCache) Available(ctx context.Context) bool {
	return f.ready.Load() && !f.disabled.Load()
}

func (f *mockFileCache) Disabled(ctx context.Context) bool {
	return f.disabled.Load()
}

func (f *mockFileCache) SetDisabled(v bool) {
	f.disabled.Store(v)
	f.ready.Store(true)
}

func (f *mockFileCache) SetReady(v bool) {
	f.ready.Store(v)
}
