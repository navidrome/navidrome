package singleton_test

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/navidrome/navidrome/model/id"
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
	var numInstancesCreated int
	constructor := func() *T {
		numInstancesCreated++
		return &T{id: id.NewRandom()}
	}

	It("calls the constructor to create a new instance", func() {
		instance := singleton.GetInstance(constructor)
		Expect(numInstancesCreated).To(Equal(1))
		Expect(instance).To(BeAssignableToTypeOf(&T{}))
	})

	It("does not call the constructor the next time", func() {
		instance := singleton.GetInstance(constructor)
		newInstance := singleton.GetInstance(constructor)

		Expect(newInstance.id).To(Equal(instance.id))
		Expect(numInstancesCreated).To(Equal(1))
	})

	It("makes a distinction between a type and its pointer", func() {
		instance := singleton.GetInstance(constructor)
		newInstance := singleton.GetInstance(func() T {
			numInstancesCreated++
			return T{id: id.NewRandom()}
		})

		Expect(instance).To(BeAssignableToTypeOf(&T{}))
		Expect(newInstance).To(BeAssignableToTypeOf(T{}))
		Expect(newInstance.id).ToNot(Equal(instance.id))
		Expect(numInstancesCreated).To(Equal(2))
	})

	It("only calls the constructor once when called concurrently", func() {
		// This test creates 80000 goroutines that call GetInstance concurrently. If the constructor is called more than once, the test will fail.
		const numCallsToDo = 80000
		var numCallsDone atomic.Uint32

		// This WaitGroup is used to make sure all goroutines are ready before the test starts
		prepare := sync.WaitGroup{}
		prepare.Add(numCallsToDo)

		// This WaitGroup is used to synchronize the start of all goroutines as simultaneous as possible
		start := sync.WaitGroup{}
		start.Add(1)

		// This WaitGroup is used to wait for all goroutines to be done
		done := sync.WaitGroup{}
		done.Add(numCallsToDo)

		numInstancesCreated = 0
		for i := 0; i < numCallsToDo; i++ {
			go func() {
				// This is needed to make sure the test does not hang if it fails
				defer GinkgoRecover()

				// Wait for all goroutines to be ready
				start.Wait()
				instance := singleton.GetInstance(func() struct{ I int } {
					numInstancesCreated++
					return struct{ I int }{I: numInstancesCreated}
				})
				// Increment the number of calls done
				numCallsDone.Add(1)

				// Flag the main WaitGroup that this goroutine is done
				done.Done()

				// Make sure the instance we get is always the same one
				Expect(instance.I).To(Equal(1))
			}()
			// Flag that this goroutine is ready to start
			prepare.Done()
		}
		prepare.Wait() // Wait for all goroutines to be ready
		start.Done()   // Start all goroutines
		done.Wait()    // Wait for all goroutines to be done

		Expect(numCallsDone.Load()).To(Equal(uint32(numCallsToDo)))
		Expect(numInstancesCreated).To(Equal(1))
	})
})
