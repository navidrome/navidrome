package singleton_test

import (
	"testing"

	"github.com/navidrome/navidrome/utils/singleton"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestSingleton(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Singleton Suite")
}

var _ = Describe("Get", func() {
	type T struct{ val int }
	var wasCalled bool
	var instance interface{}
	constructor := func() interface{} {
		wasCalled = true
		return &T{}
	}

	BeforeEach(func() {
		instance = singleton.Get(T{}, constructor)
	})

	It("calls the constructor to create a new instance", func() {
		Expect(wasCalled).To(BeTrue())
		Expect(instance).To(BeAssignableToTypeOf(&T{}))
	})

	It("does not call the constructor the next time", func() {
		instance.(*T).val = 10
		wasCalled = false

		newInstance := singleton.Get(T{}, constructor)

		Expect(newInstance.(*T).val).To(Equal(10))
		Expect(wasCalled).To(BeFalse())
	})

	It("does not call the constructor even if a pointer is passed as the object", func() {
		instance.(*T).val = 20
		wasCalled = false

		newInstance := singleton.Get(&T{}, constructor)

		Expect(newInstance.(*T).val).To(Equal(20))
		Expect(wasCalled).To(BeFalse())
	})
})
