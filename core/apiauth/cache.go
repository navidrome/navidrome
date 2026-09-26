package apiauth

import (
	"maps"
	"sync"
	"time"
)

const maxLivenessEntries = 1024

type livenessEntry struct {
	userID     string
	epoch      int
	lastUsedAt time.Time
	expires    time.Time
}

// livenessCache bounds how long a node trusts "this grant exists" without asking the DB.
type livenessCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	gen     uint64
	entries map[string]livenessEntry
	evicted map[string]uint64 // grant id -> generation of its last eviction
	floor   uint64            // fills started before the last trim of evicted are dropped
}

func newLivenessCache(ttl time.Duration) *livenessCache {
	return &livenessCache{ttl: ttl, entries: map[string]livenessEntry{}, evicted: map[string]uint64{}}
}

func (c *livenessCache) begin() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.gen
}

func (c *livenessCache) get(id string, now time.Time) (livenessEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[id]
	if !ok || !now.Before(e.expires) {
		return livenessEntry{}, false
	}
	return e, true
}

// put ignores a fill whose DB read started before the grant was last evicted.
func (c *livenessCache) put(id string, e livenessEntry, now time.Time, started uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if started < c.floor || c.evicted[id] > started {
		return
	}
	if len(c.entries) >= maxLivenessEntries {
		maps.DeleteFunc(c.entries, func(_ string, v livenessEntry) bool { return !now.Before(v.expires) })
	}
	if _, refresh := c.entries[id]; !refresh && len(c.entries) >= maxLivenessEntries {
		// Dropping an arbitrary live entry only costs that grant one extra DB read.
		for k := range c.entries {
			delete(c.entries, k)
			break
		}
	}
	e.expires = now.Add(c.ttl)
	c.entries[id] = e
}

func (c *livenessCache) evict(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.evicted) >= maxLivenessEntries {
		clear(c.evicted)
		c.floor = c.gen
	}
	c.gen++
	c.evicted[id] = c.gen
	delete(c.entries, id)
}

func (c *livenessCache) markUsed(id string, at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.entries[id]; ok {
		e.lastUsedAt = at
		c.entries[id] = e
	}
}

func (c *livenessCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}
