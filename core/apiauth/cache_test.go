package apiauth

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("livenessCache", func() {
	var c *livenessCache
	var t0 time.Time

	BeforeEach(func() {
		c = newLivenessCache(30 * time.Second)
		t0 = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	})

	It("returns an entry until its TTL passes", func() {
		c.put("g1", livenessEntry{userID: "u1", epoch: 2}, t0, c.begin())
		e, ok := c.get("g1", t0.Add(29*time.Second))
		Expect(ok).To(BeTrue())
		Expect(e.userID).To(Equal("u1"))
		Expect(e.epoch).To(Equal(2))
		_, ok = c.get("g1", t0.Add(30*time.Second))
		Expect(ok).To(BeFalse())
	})

	It("forgets evicted entries", func() {
		c.put("g1", livenessEntry{userID: "u1"}, t0, c.begin())
		c.evict("g1")
		_, ok := c.get("g1", t0)
		Expect(ok).To(BeFalse())
	})

	It("drops a fill that started before an eviction of the same grant", func() {
		started := c.begin()                                  // a request reads the grant from the DB...
		c.evict("g1")                                         // ...a logout deletes and evicts it...
		c.put("g1", livenessEntry{userID: "u1"}, t0, started) // ...then the slow request tries to cache it
		_, ok := c.get("g1", t0)
		Expect(ok).To(BeFalse())
	})

	It("still accepts fills of other grants and later fills of the same grant", func() {
		started := c.begin()
		c.evict("g1")
		c.put("g2", livenessEntry{userID: "u2"}, t0, started)
		_, ok := c.get("g2", t0)
		Expect(ok).To(BeTrue())
		c.put("g1", livenessEntry{userID: "u1"}, t0, c.begin())
		_, ok = c.get("g1", t0)
		Expect(ok).To(BeTrue())
	})

	It("still drops a racing fill after the eviction log is trimmed", func() {
		started := c.begin()
		c.evict("g1")
		for i := range maxLivenessEntries {
			c.evict(fmt.Sprint("other", i))
		}
		c.put("fresh", livenessEntry{}, t0, c.begin())
		c.put("g1", livenessEntry{userID: "u1"}, t0, started)
		_, ok := c.get("g1", t0)
		Expect(ok).To(BeFalse())
	})

	It("records the last use without extending the TTL", func() {
		c.put("g1", livenessEntry{userID: "u1"}, t0, c.begin())
		c.markUsed("g1", t0.Add(10*time.Second))
		e, _ := c.get("g1", t0.Add(11*time.Second))
		Expect(e.lastUsedAt).To(Equal(t0.Add(10 * time.Second)))
		_, ok := c.get("g1", t0.Add(30*time.Second))
		Expect(ok).To(BeFalse())
	})

	It("drops expired entries when it grows", func() {
		for i := range maxLivenessEntries {
			c.put(fmt.Sprint(i), livenessEntry{}, t0, c.begin())
		}
		c.put("fresh", livenessEntry{}, t0.Add(time.Minute), c.begin())
		Expect(c.len()).To(Equal(1))
	})

	It("never grows past its cap, even when every entry is live", func() {
		for i := range maxLivenessEntries {
			c.put(fmt.Sprint(i), livenessEntry{}, t0, c.begin())
		}
		c.put("fresh", livenessEntry{}, t0, c.begin())
		Expect(c.len()).To(Equal(maxLivenessEntries))
		_, ok := c.get("fresh", t0)
		Expect(ok).To(BeTrue())

		c.put("fresh", livenessEntry{userID: "u1"}, t0, c.begin())
		Expect(c.len()).To(Equal(maxLivenessEntries))
	})
})
