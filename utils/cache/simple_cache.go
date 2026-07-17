package cache

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

type SimpleCache[K comparable, V any] interface {
	Add(key K, value V) error
	AddWithTTL(key K, value V, ttl time.Duration) error
	Get(key K) (V, error)
	GetWithLoader(key K, loader func(key K) (V, time.Duration, error)) (V, error)
	Remove(key K)
	Keys() []K
	Values() []V
	Len() int
	OnExpiration(fn func(K, V)) func()
}

type Options struct {
	SizeLimit  uint64
	DefaultTTL time.Duration
}

func NewSimpleCache[K comparable, V any](options ...Options) SimpleCache[K, V] {
	opts := []ttlcache.Option[K, V]{
		ttlcache.WithDisableTouchOnHit[K, V](),
	}
	if len(options) > 0 {
		o := options[0]
		if o.SizeLimit > 0 {
			opts = append(opts, ttlcache.WithCapacity[K, V](o.SizeLimit))
		}
		if o.DefaultTTL > 0 {
			opts = append(opts, ttlcache.WithTTL[K, V](o.DefaultTTL))
		}
	}

	c := ttlcache.New[K, V](opts...)
	cache := &simpleCache[K, V]{
		data:  c,
		loads: make(map[K]*flight[V]),
	}
	go cache.data.Start()

	// Automatic cleanup to prevent goroutine leak when cache is garbage collected
	runtime.AddCleanup(cache, func(ttlCache *ttlcache.Cache[K, V]) {
		ttlCache.Stop()
	}, cache.data)

	return cache
}

const evictionTimeout = 1 * time.Hour

type simpleCache[K comparable, V any] struct {
	data             *ttlcache.Cache[K, V]
	evictionDeadline atomic.Pointer[time.Time]
	loadsMu          sync.Mutex
	loads            map[K]*flight[V]
}

// flight tracks an in-progress load so concurrent misses of the same key share it.
type flight[V any] struct {
	done chan struct{}
	val  V
	err  error
}

func (f *flight[V]) result() (V, error) {
	if f.err != nil {
		var zero V
		return zero, fmt.Errorf("cache error: loader returned %w", f.err)
	}
	return f.val, nil
}

func (c *simpleCache[K, V]) Add(key K, value V) error {
	c.evictExpired()
	return c.AddWithTTL(key, value, ttlcache.DefaultTTL)
}

func (c *simpleCache[K, V]) AddWithTTL(key K, value V, ttl time.Duration) error {
	c.evictExpired()
	item := c.data.Set(key, value, ttl)
	if item == nil {
		return errors.New("failed to add item")
	}
	return nil
}

func (c *simpleCache[K, V]) Remove(key K) {
	c.data.Delete(key)
}

func (c *simpleCache[K, V]) Get(key K) (V, error) {
	item := c.data.Get(key)
	if item == nil {
		var zero V
		return zero, errors.New("item not found")
	}
	return item.Value(), nil
}

// GetWithLoader loads misses via the loader, deduplicating concurrent loads of
// the same key: one loader call runs, and every waiter shares its result (or error).
func (c *simpleCache[K, V]) GetWithLoader(key K, loader func(key K) (V, time.Duration, error)) (V, error) {
	if item := c.data.Get(key); item != nil {
		return item.Value(), nil
	}

	c.loadsMu.Lock()
	if f, ok := c.loads[key]; ok {
		c.loadsMu.Unlock()
		<-f.done
		return f.result()
	}
	f := &flight[V]{done: make(chan struct{}), err: errLoaderPanicked}
	c.loads[key] = f
	c.loadsMu.Unlock()

	// Deregister even if the loader panics, so waiters get an error instead of
	// blocking forever on a flight that will never complete.
	defer func() {
		close(f.done)
		c.loadsMu.Lock()
		delete(c.loads, key)
		c.loadsMu.Unlock()
	}()

	if item := c.data.Get(key); item != nil { // a flight may have completed since the miss
		f.val, f.err = item.Value(), nil
	} else {
		c.evictExpired()
		var ttl time.Duration
		f.val, ttl, f.err = loader(key)
		if f.err == nil {
			c.data.Set(key, f.val, ttl)
		}
	}
	return f.result()
}

var errLoaderPanicked = errors.New("loader panicked")

func (c *simpleCache[K, V]) evictExpired() {
	if c.evictionDeadline.Load() == nil || c.evictionDeadline.Load().Before(time.Now()) {
		c.data.DeleteExpired()
		c.evictionDeadline.Store(new(time.Now().Add(evictionTimeout)))
	}
}

func (c *simpleCache[K, V]) Keys() []K {
	res := make([]K, 0, c.data.Len())
	c.data.Range(func(item *ttlcache.Item[K, V]) bool {
		if !item.IsExpired() {
			res = append(res, item.Key())
		}
		return true
	})
	return res
}

func (c *simpleCache[K, V]) Values() []V {
	res := make([]V, 0, c.data.Len())
	c.data.Range(func(item *ttlcache.Item[K, V]) bool {
		if !item.IsExpired() {
			res = append(res, item.Value())
		}
		return true
	})
	return res
}

func (c *simpleCache[K, V]) Len() int {
	return c.data.Len()
}

func (c *simpleCache[K, V]) OnExpiration(fn func(K, V)) func() {
	return c.data.OnEviction(func(_ context.Context, reason ttlcache.EvictionReason, item *ttlcache.Item[K, V]) {
		if reason == ttlcache.EvictionReasonExpired {
			fn(item.Key(), item.Value())
		}
	})
}
