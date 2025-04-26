package plugins

import (
	"context"
	"sync"
	"time"

	"github.com/navidrome/navidrome/log"
)

// wasmInstancePool is a generic pool with max size and TTL, similar to sync.Pool but with expiration and Close support.
type wasmInstancePool[T any] struct {
	name         string
	new          func(ctx context.Context) T
	maxInstances int
	ttl          time.Duration

	mu      sync.Mutex
	items   []poolItem[T]
	closing chan struct{}
	closed  bool
}

type poolItem[T any] struct {
	value    T
	lastUsed time.Time
}

func NewWasmInstancePool[T any](name string, maxInstances int, ttl time.Duration, newFn func(ctx context.Context) T) *wasmInstancePool[T] {
	p := &wasmInstancePool[T]{
		name:         name,
		new:          newFn,
		maxInstances: maxInstances,
		ttl:          ttl,
		closing:      make(chan struct{}),
	}
	log.Debug(context.Background(), "wasmInstancePool: created new pool", "pool", p.name, "maxInstances", p.maxInstances, "ttl", p.ttl)
	go p.cleanupLoop()
	return p
}

func (p *wasmInstancePool[T]) Get(ctx context.Context) T {
	p.mu.Lock()
	n := len(p.items)
	if n > 0 {
		item := p.items[n-1]
		p.items = p.items[:n-1]
		p.mu.Unlock()
		log.Trace(ctx, "wasmInstancePool: got instance from pool", "pool", p.name)
		return item.value
	}
	p.mu.Unlock()
	inst := p.new(ctx)
	// If creation fails, log error (cannot check nil for all types)
	return inst
}

func (p *wasmInstancePool[T]) Put(ctx context.Context, v T) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		log.Trace(ctx, "wasmInstancePool: pool closed, closing instance", "pool", p.name)
		p.closeItem(ctx, v)
		return
	}
	if len(p.items) < p.maxInstances {
		p.items = append(p.items, poolItem[T]{value: v, lastUsed: time.Now()})
		p.mu.Unlock()
		log.Trace(ctx, "wasmInstancePool: returned instance to pool", "pool", p.name)
	} else {
		p.mu.Unlock()
		log.Trace(ctx, "wasmInstancePool: pool full, closing instance", "pool", p.name)
		p.closeItem(ctx, v)
	}
}

func (p *wasmInstancePool[T]) Close(ctx context.Context) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true
	close(p.closing)
	items := p.items
	p.items = nil
	p.mu.Unlock()
	log.Trace(ctx, "wasmInstancePool: closing pool and all instances", "pool", p.name)
	for _, item := range items {
		p.closeItem(ctx, item.value)
	}
}

func (p *wasmInstancePool[T]) cleanupLoop() {
	ticker := time.NewTicker(p.ttl / 3)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			p.cleanupExpired()
		case <-p.closing:
			return
		}
	}
}

func (p *wasmInstancePool[T]) cleanupExpired() {
	ctx := context.Background()
	now := time.Now()
	p.mu.Lock()
	var keep []poolItem[T]
	for _, item := range p.items {
		if now.Sub(item.lastUsed) > p.ttl {
			p.mu.Unlock()
			log.Trace(ctx, "wasmInstancePool: expiring instance due to TTL", "pool", p.name)
			p.closeItem(ctx, item.value)
			p.mu.Lock()
		} else {
			keep = append(keep, item)
		}
	}
	if len(keep) < len(p.items) {
		log.Trace(ctx, "wasmInstancePool: cleaned up expired instances", "pool", p.name, "numExpired", len(p.items)-len(keep), "numRemaining", len(keep))
	}
	p.items = keep
	p.mu.Unlock()
}

func (p *wasmInstancePool[T]) closeItem(ctx context.Context, v T) {
	if closer, ok := any(v).(interface{ Close(context.Context) error }); ok {
		_ = closer.Close(ctx)
	}
}
