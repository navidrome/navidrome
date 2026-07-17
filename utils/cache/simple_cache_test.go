package cache

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SimpleCache", func() {
	var (
		cache SimpleCache[string, string]
	)

	BeforeEach(func() {
		cache = NewSimpleCache[string, string]()
	})

	Describe("Add and Get", func() {
		It("should add and retrieve a value", func() {
			err := cache.Add("key", "value")
			Expect(err).NotTo(HaveOccurred())

			value, err := cache.Get("key")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal("value"))
		})
	})

	Describe("AddWithTTL and Get", func() {
		It("should add a value with TTL and retrieve it", func() {
			err := cache.AddWithTTL("key", "value", 1*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			value, err := cache.Get("key")
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal("value"))
		})

		It("should not retrieve a value after its TTL has expired", func() {
			err := cache.AddWithTTL("key", "value", 10*time.Millisecond)
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(50 * time.Millisecond)

			_, err = cache.Get("key")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("GetWithLoader", func() {
		It("should retrieve a value using the loader function", func() {
			loader := func(key string) (string, time.Duration, error) {
				return fmt.Sprintf("%s=value", key), 1 * time.Minute, nil
			}

			value, err := cache.GetWithLoader("key", loader)
			Expect(err).NotTo(HaveOccurred())
			Expect(value).To(Equal("key=value"))
		})

		It("should return the error returned by the loader function", func() {
			loader := func(key string) (string, time.Duration, error) {
				return "", 0, errors.New("some error")
			}

			_, err := cache.GetWithLoader("key", loader)
			Expect(err).To(HaveOccurred())
		})

		It("suppresses concurrent loads for the same key", func() {
			var calls atomic.Int32
			release := make(chan struct{})
			started := make(chan struct{}, 10)
			loader := func(key string) (string, time.Duration, error) {
				calls.Add(1)
				started <- struct{}{}
				<-release
				return "shared", time.Minute, nil
			}

			const n = 5
			var wg sync.WaitGroup
			results := make([]string, n)
			errs := make([]error, n)
			for i := range n {
				wg.Go(func() {
					results[i], errs[i] = cache.GetWithLoader("key", loader)
				})
			}

			Eventually(started).Should(Receive())
			Consistently(started).ShouldNot(Receive())
			close(release)
			wg.Wait()

			Expect(calls.Load()).To(Equal(int32(1)))
			for i := range n {
				Expect(errs[i]).ToNot(HaveOccurred())
				Expect(results[i]).To(Equal("shared"))
			}
		})

		It("returns the loader error to all concurrent callers", func() {
			release := make(chan struct{})
			started := make(chan struct{}, 10)
			loader := func(key string) (string, time.Duration, error) {
				started <- struct{}{}
				<-release
				return "", 0, errors.New("load failed")
			}

			const n = 3
			var wg sync.WaitGroup
			errs := make([]error, n)
			for i := range n {
				wg.Go(func() {
					_, errs[i] = cache.GetWithLoader("key", loader)
				})
			}

			Eventually(started).Should(Receive())
			Consistently(started).ShouldNot(Receive())
			close(release)
			wg.Wait()

			for i := range n {
				Expect(errs[i]).To(MatchError(ContainSubstring("load failed")))
			}
		})

		It("supports interface value types with nil results", func() {
			c := NewSimpleCache[string, any]()
			v, err := c.GetWithLoader("key", func(string) (any, time.Duration, error) {
				return nil, time.Minute, nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(BeNil())
		})

		It("cleans up the in-flight registration when the loader panics", func() {
			Expect(func() {
				_, _ = cache.GetWithLoader("key", func(string) (string, time.Duration, error) {
					panic("boom")
				})
			}).To(PanicWith("boom"))

			// Without cleanup this would deadlock on the never-completed flight
			v, err := cache.GetWithLoader("key", func(string) (string, time.Duration, error) {
				return "ok", 0, nil
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal("ok"))
		})

		It("loads different keys independently", func() {
			release := make(chan struct{})
			started := make(chan struct{}, 10)
			loader := func(key string) (string, time.Duration, error) {
				started <- struct{}{}
				<-release
				return key + "=value", time.Minute, nil
			}

			var wg sync.WaitGroup
			for _, key := range []string{"key1", "key2"} {
				wg.Go(func() {
					value, err := cache.GetWithLoader(key, loader)
					Expect(err).ToNot(HaveOccurred())
					Expect(value).To(Equal(key + "=value"))
				})
			}

			// Both loaders must be in flight at once: distinct keys are not suppressed
			Eventually(started).Should(Receive())
			Eventually(started).Should(Receive())
			close(release)
			wg.Wait()
		})
	})

	Describe("Keys and Values", func() {
		It("should return all keys and all values", func() {
			err := cache.Add("key1", "value1")
			Expect(err).NotTo(HaveOccurred())

			err = cache.Add("key2", "value2")
			Expect(err).NotTo(HaveOccurred())

			keys := cache.Keys()
			Expect(keys).To(ConsistOf("key1", "key2"))

			values := cache.Values()
			Expect(values).To(ConsistOf("value1", "value2"))
		})

		Context("when there are expired items in the cache", func() {
			It("should not return expired items", func() {
				Expect(cache.Add("key0", "value0")).To(Succeed())
				for i := 1; i <= 3; i++ {
					err := cache.AddWithTTL(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i), 10*time.Millisecond)
					Expect(err).NotTo(HaveOccurred())
				}

				time.Sleep(50 * time.Millisecond)

				Expect(cache.Keys()).To(ConsistOf("key0"))
				Expect(cache.Values()).To(ConsistOf("value0"))
			})
		})
	})

	Describe("Options", func() {
		Context("when size limit is set", func() {
			BeforeEach(func() {
				cache = NewSimpleCache[string, string](Options{
					SizeLimit: 2,
				})
			})

			It("should drop the oldest item when the size limit is reached", func() {
				for i := 1; i <= 3; i++ {
					err := cache.Add(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
					Expect(err).NotTo(HaveOccurred())
				}

				Expect(cache.Keys()).To(ConsistOf("key2", "key3"))
			})
		})

		Context("when default TTL is set", func() {
			BeforeEach(func() {
				cache = NewSimpleCache[string, string](Options{
					DefaultTTL: 10 * time.Millisecond,
				})
			})

			It("should expire items after the default TTL", func() {
				Expect(cache.AddWithTTL("key0", "value0", 1*time.Minute)).To(Succeed())
				for i := 1; i <= 3; i++ {
					err := cache.Add(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
					Expect(err).NotTo(HaveOccurred())
				}

				time.Sleep(50 * time.Millisecond)

				for i := 1; i <= 3; i++ {
					_, err := cache.Get(fmt.Sprintf("key%d", i))
					Expect(err).To(HaveOccurred())
				}
				Expect(cache.Get("key0")).To(Equal("value0"))
			})
		})

		Describe("OnExpiration", func() {
			It("should call callback when item expires", func() {
				cache = NewSimpleCache[string, string]()
				expired := make(chan struct{})
				cache.OnExpiration(func(k, v string) { close(expired) })
				Expect(cache.AddWithTTL("key", "value", 10*time.Millisecond)).To(Succeed())
				select {
				case <-expired:
				case <-time.After(100 * time.Millisecond):
					Fail("expiration callback not called")
				}
			})
		})
	})
})
