package singleton_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/navidrome/navidrome/utils/singleton"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSingleton(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Singleton Suite")
}

var _ = Describe("Get", func() {
	type T struct{ id string }
	var numInstances int
	constructor := func() interface{} {
		numInstances++
		return &T{id: uuid.NewString()}
	}

	It("calls the constructor to create a new instance", func() {
		instance := singleton.Get(T{}, constructor)
		Expect(numInstances).To(Equal(1))
		Expect(instance).To(BeAssignableToTypeOf(&T{}))
	})

	It("does not call the constructor the next time", func() {
		instance := singleton.Get(T{}, constructor)
		newInstance := singleton.Get(T{}, constructor)

		Expect(newInstance.(*T).id).To(Equal(instance.(*T).id))
		Expect(numInstances).To(Equal(1))
	})

	It("does not call the constructor even if a pointer is passed as the object", func() {
		instance := singleton.Get(T{}, constructor)
		newInstance := singleton.Get(&T{}, constructor)

		Expect(newInstance.(*T).id).To(Equal(instance.(*T).id))
		Expect(numInstances).To(Equal(1))
	})

	It("only calls the constructor once when called concurrently", func() {
		const maxCalls = 2000
		var numCalls int32
		start := sync.WaitGroup{}
		start.Add(1)
		prepare := sync.WaitGroup{}
		prepare.Add(maxCalls)
		done := sync.WaitGroup{}
		done.Add(maxCalls)
		for i := 0; i < maxCalls; i++ {
			go func() {
				start.Wait()
				singleton.Get(T{}, constructor)
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
