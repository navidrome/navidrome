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
		pool := newWasmInstancePool[*testInstance]("test", 2, time.Second, func(ctx context.Context) (*testInstance, error) {
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
		pool := newWasmInstancePool[*testInstance]("test", 1, time.Second, func(ctx context.Context) (*testInstance, error) {
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
		pool := newWasmInstancePool[*testInstance]("test", 2, 100*time.Millisecond, func(ctx context.Context) (*testInstance, error) {
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
		pool := newWasmInstancePool[*testInstance]("test", 2, time.Second, func(ctx context.Context) (*testInstance, error) {
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
		pool := newWasmInstancePool[*testInstance]("test", 4, time.Second, func(ctx context.Context) (*testInstance, error) {
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
})
