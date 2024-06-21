package cache

import (
	"time"

	"github.com/jellydator/ttlcache/v2"
)

type SimpleCache[V any] interface {
	Add(key string, value V) error
	AddWithTTL(key string, value V, ttl time.Duration) error
	Get(key string) (V, error)
	GetWithLoader(key string, loader func(key string) (V, time.Duration, error)) (V, error)
	Keys() []string
}

type Options struct {
	SizeLimit  int
	DefaultTTL time.Duration
}

func NewSimpleCache[V any](options ...Options) SimpleCache[V] {
	c := ttlcache.NewCache()
	c.SkipTTLExtensionOnHit(true)
	if len(options) > 0 {
		c.SetCacheSizeLimit(options[0].SizeLimit)
		_ = c.SetTTL(options[0].DefaultTTL)
	}

	return &simpleCache[V]{
		data: c,
	}
}

type simpleCache[V any] struct {
	data *ttlcache.Cache
}

func (c *simpleCache[V]) Add(key string, value V) error {
	return c.data.Set(key, value)
}

func (c *simpleCache[V]) AddWithTTL(key string, value V, ttl time.Duration) error {
	return c.data.SetWithTTL(key, value, ttl)
}

func (c *simpleCache[V]) Get(key string) (V, error) {
	v, err := c.data.Get(key)
	if err != nil {
		var zero V
		return zero, err
	}
	return v.(V), nil
}

func (c *simpleCache[V]) GetWithLoader(key string, loader func(key string) (V, time.Duration, error)) (V, error) {
	v, err := c.data.GetByLoader(key, func(key string) (interface{}, time.Duration, error) {
		v, ttl, err := loader(key)
		return v, ttl, err
	})
	if err != nil {
		var zero V
		return zero, err
	}
	return v.(V), nil
}

func (c *simpleCache[V]) Keys() []string {
	return c.data.GetKeys()
}
