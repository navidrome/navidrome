package artwork

import (
	"context"
	"sync/atomic"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/cache"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("CacheWarmer", func() {
	It("returns noop when cache is disabled", func() {
		fc := &mockFileCache{}
		fc.SetDisabled(true)
		cw := NewCacheWarmer(nil, fc)
		_, ok := cw.(*noopCacheWarmer)
		Expect(ok).To(BeTrue())
	})

	It("drops buffered items when cache becomes disabled", func() {
		fc := &mockFileCache{}
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

type mockFileCache struct {
	disabled atomic.Bool
	ready    atomic.Bool
}

func (f *mockFileCache) Get(ctx context.Context, item cache.Item) (*cache.CachedStream, error) {
	return nil, nil
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
