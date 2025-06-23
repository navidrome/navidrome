package plugins

import (
	"context"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type testInstance struct {
	closed atomic.Bool
}

func (t *testInstance) Close(ctx context.Context) error {
	t.closed.Store(true)
	return nil
}

var _ = Describe("wasmInstancePool", func() {
	var (
		ctx = context.Background()
	)

	It("should Get and Put instances", func() {
		pool := newWasmInstancePool[*testInstance]("test", 2, 10, 5*time.Second, time.Second, func(ctx context.Context) (*testInstance, error) {
			return &testInstance{}, nil
		})
		inst, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		Expect(inst).ToNot(BeNil())
		pool.Put(ctx, inst)
		inst2, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		Expect(inst2).To(Equal(inst))
		pool.Close(ctx)
	})

	It("should not exceed max instances", func() {
		pool := newWasmInstancePool[*testInstance]("test", 1, 10, 5*time.Second, time.Second, func(ctx context.Context) (*testInstance, error) {
			return &testInstance{}, nil
		})
		inst1, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		inst2 := &testInstance{}
		pool.Put(ctx, inst1)
		pool.Put(ctx, inst2) // should close inst2
		Expect(inst2.closed.Load()).To(BeTrue())
		pool.Close(ctx)
	})

	It("should expire and close instances after TTL", func() {
		pool := newWasmInstancePool[*testInstance]("test", 2, 10, 5*time.Second, 100*time.Millisecond, func(ctx context.Context) (*testInstance, error) {
			return &testInstance{}, nil
		})
		inst, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		pool.Put(ctx, inst)
		// Wait for TTL cleanup
		time.Sleep(300 * time.Millisecond)
		Expect(inst.closed.Load()).To(BeTrue())
		pool.Close(ctx)
	})

	It("should close all on pool Close", func() {
		pool := newWasmInstancePool[*testInstance]("test", 2, 10, 5*time.Second, time.Second, func(ctx context.Context) (*testInstance, error) {
			return &testInstance{}, nil
		})
		inst1, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		inst2, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		pool.Put(ctx, inst1)
		pool.Put(ctx, inst2)
		pool.Close(ctx)
		Expect(inst1.closed.Load()).To(BeTrue())
		Expect(inst2.closed.Load()).To(BeTrue())
	})

	It("should be safe for concurrent Get/Put", func() {
		pool := newWasmInstancePool[*testInstance]("test", 4, 10, 5*time.Second, time.Second, func(ctx context.Context) (*testInstance, error) {
			return &testInstance{}, nil
		})
		done := make(chan struct{})
		for i := 0; i < 8; i++ {
			go func() {
				inst, err := pool.Get(ctx)
				Expect(err).To(BeNil())
				pool.Put(ctx, inst)
				done <- struct{}{}
			}()
		}
		for i := 0; i < 8; i++ {
			<-done
		}
		pool.Close(ctx)
	})

	It("should enforce max concurrent instances limit", func() {
		callCount := atomic.Int32{}
		pool := newWasmInstancePool[*testInstance]("test", 2, 3, 100*time.Millisecond, time.Second, func(ctx context.Context) (*testInstance, error) {
			callCount.Add(1)
			return &testInstance{}, nil
		})

		// Get 3 instances (should hit the limit)
		inst1, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		inst2, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		inst3, err := pool.Get(ctx)
		Expect(err).To(BeNil())

		// Should have created exactly 3 instances at this point
		Expect(callCount.Load()).To(Equal(int32(3)))

		// Fourth call should timeout without creating a new instance
		start := time.Now()
		_, err = pool.Get(ctx)
		duration := time.Since(start)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("timeout waiting for available instance"))
		Expect(duration).To(BeNumerically(">=", 100*time.Millisecond))
		Expect(duration).To(BeNumerically("<", 200*time.Millisecond))

		// Still should have only 3 instances (timeout didn't create new one)
		Expect(callCount.Load()).To(Equal(int32(3)))

		// Return one instance and try again - should succeed by reusing returned instance
		pool.Put(ctx, inst1)
		inst4, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		Expect(inst4).To(Equal(inst1)) // Should be the same instance we returned

		// Still should have only 3 instances total (reused inst1)
		Expect(callCount.Load()).To(Equal(int32(3)))

		pool.Put(ctx, inst2)
		pool.Put(ctx, inst3)
		pool.Put(ctx, inst4)
		pool.Close(ctx)
	})

	It("should handle concurrent waiters properly", func() {
		pool := newWasmInstancePool[*testInstance]("test", 1, 2, time.Second, time.Second, func(ctx context.Context) (*testInstance, error) {
			return &testInstance{}, nil
		})

		// Fill up the concurrent slots
		inst1, err := pool.Get(ctx)
		Expect(err).To(BeNil())
		inst2, err := pool.Get(ctx)
		Expect(err).To(BeNil())

		// Start multiple waiters
		waiterResults := make(chan error, 3)
		for i := 0; i < 3; i++ {
			go func() {
				_, err := pool.Get(ctx)
				waiterResults <- err
			}()
		}

		// Wait a bit to ensure waiters are queued
		time.Sleep(50 * time.Millisecond)

		// Return instances one by one
		pool.Put(ctx, inst1)
		pool.Put(ctx, inst2)

		// Two waiters should succeed, one should timeout
		successCount := 0
		timeoutCount := 0
		for i := 0; i < 3; i++ {
			select {
			case err := <-waiterResults:
				if err == nil {
					successCount++
				} else {
					timeoutCount++
				}
			case <-time.After(2 * time.Second):
				Fail("Test timed out waiting for waiter results")
			}
		}

		Expect(successCount).To(Equal(2))
		Expect(timeoutCount).To(Equal(1))

		pool.Close(ctx)
	})
})
