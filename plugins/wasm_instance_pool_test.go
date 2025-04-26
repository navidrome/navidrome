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
		pool := NewWasmInstancePool[*testInstance]("test", 2, time.Second, func(ctx context.Context) *testInstance {
			return &testInstance{}
		})
		inst := pool.Get(ctx)
		Expect(inst).ToNot(BeNil())
		pool.Put(ctx, inst)
		inst2 := pool.Get(ctx)
		Expect(inst2).To(Equal(inst))
		pool.Close(ctx)
	})

	It("should not exceed max instances", func() {
		pool := NewWasmInstancePool[*testInstance]("test", 1, time.Second, func(ctx context.Context) *testInstance {
			return &testInstance{}
		})
		inst1 := pool.Get(ctx)
		inst2 := &testInstance{}
		pool.Put(ctx, inst1)
		pool.Put(ctx, inst2) // should close inst2
		Expect(inst2.closed.Load()).To(BeTrue())
		pool.Close(ctx)
	})

	It("should expire and close instances after TTL", func() {
		pool := NewWasmInstancePool[*testInstance]("test", 2, 100*time.Millisecond, func(ctx context.Context) *testInstance {
			return &testInstance{}
		})
		inst := pool.Get(ctx)
		pool.Put(ctx, inst)
		// Wait for TTL cleanup
		time.Sleep(300 * time.Millisecond)
		Expect(inst.closed.Load()).To(BeTrue())
		pool.Close(ctx)
	})

	It("should close all on pool Close", func() {
		pool := NewWasmInstancePool[*testInstance]("test", 2, time.Second, func(ctx context.Context) *testInstance {
			return &testInstance{}
		})
		inst1 := pool.Get(ctx)
		inst2 := pool.Get(ctx)
		pool.Put(ctx, inst1)
		pool.Put(ctx, inst2)
		pool.Close(ctx)
		Expect(inst1.closed.Load()).To(BeTrue())
		Expect(inst2.closed.Load()).To(BeTrue())
	})

	It("should be safe for concurrent Get/Put", func() {
		pool := NewWasmInstancePool[*testInstance]("test", 4, time.Second, func(ctx context.Context) *testInstance {
			return &testInstance{}
		})
		done := make(chan struct{})
		for i := 0; i < 8; i++ {
			go func() {
				inst := pool.Get(ctx)
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
