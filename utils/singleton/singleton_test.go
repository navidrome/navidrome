package singleton_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/navidrome/navidrome/utils/singleton"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSingleton(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Singleton Suite")
}

var _ = Describe("GetInstance", func() {
	type T struct{ id string }
	var numInstances int
	constructor := func() *T {
		numInstances++
		return &T{id: uuid.NewString()}
	}

	It("calls the constructor to create a new instance", func() {
		instance := singleton.GetInstance(constructor)
		Expect(numInstances).To(Equal(1))
		Expect(instance).To(BeAssignableToTypeOf(&T{}))
	})

	It("does not call the constructor the next time", func() {
		instance := singleton.GetInstance(constructor)
		newInstance := singleton.GetInstance(constructor)

		Expect(newInstance.id).To(Equal(instance.id))
		Expect(numInstances).To(Equal(1))
	})

	It("makes a distinction between a type and its pointer", func() {
		instance := singleton.GetInstance(constructor)
		newInstance := singleton.GetInstance(func() T {
			numInstances++
			return T{id: uuid.NewString()}
		})

		Expect(instance).To(BeAssignableToTypeOf(&T{}))
		Expect(newInstance).To(BeAssignableToTypeOf(T{}))
		Expect(newInstance.id).ToNot(Equal(instance.id))
		Expect(numInstances).To(Equal(2))
	})

	It("only calls the constructor once when called concurrently", func() {
		const maxCalls = 8000
		var numCalls int32
		start := sync.WaitGroup{}
		start.Add(1)
		prepare := sync.WaitGroup{}
		prepare.Add(maxCalls)
		done := sync.WaitGroup{}
		done.Add(maxCalls)
		numInstances = 0
		for i := 0; i < maxCalls; i++ {
			go func() {
				start.Wait()
				singleton.GetInstance(func() struct{ I int } {
					numInstances++
					return struct{ I int }{I: 1}
				})
				atomic.AddInt32(&numCalls, 1)
				done.Done()
			}()
			prepare.Done()
		}
		prepare.Wait()
		start.Done()
		done.Wait()

		Expect(numCalls).To(Equal(int32(maxCalls)))
		Expect(numInstances).To(Equal(1))
	})
})
