package artwork

import (
	"context"
	"sync/atomic"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type disabledCache struct{}

func (disabledCache) Get(ctx context.Context, item cache.Item) (*cache.CachedStream, error) {
	return nil, nil
}
func (disabledCache) Available(ctx context.Context) bool { return false }
func (disabledCache) Disabled(ctx context.Context) bool  { return true }

var _ = Describe("CacheWarmer", func() {
	It("returns noop when cache is disabled", func() {
		cw := NewCacheWarmer(nil, disabledCache{})
		_, ok := cw.(*noopCacheWarmer)
		Expect(ok).To(BeTrue())
	})

	It("drops buffered items when cache becomes disabled", func() {
		fc := &fakeFileCache{}
		cw := NewCacheWarmer(nil, fc).(*cacheWarmer)
		cw.PreCache(model.MustParseArtworkID("al-test"))
		fc.SetDisabled(true)
		Eventually(func() int {
			cw.mutex.Lock()
			defer cw.mutex.Unlock()
			return len(cw.buffer)
		}).Should(Equal(0))
	})
})

type fakeFileCache struct {
	disabled atomic.Bool
	ready    atomic.Bool
}

func (f *fakeFileCache) Get(ctx context.Context, item cache.Item) (*cache.CachedStream, error) {
	return nil, nil
}
func (f *fakeFileCache) Available(ctx context.Context) bool {
	return f.ready.Load() && !f.disabled.Load()
}
func (f *fakeFileCache) Disabled(ctx context.Context) bool { return f.disabled.Load() }
func (f *fakeFileCache) SetDisabled(v bool)                { f.disabled.Store(v); f.ready.Store(true) }
