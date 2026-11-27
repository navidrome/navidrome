package configtest_test

import (
	"github.com/navidrome/navidrome/conf"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("RestoreAfterEachSpec", func() {
	// Either spec can run first: the second one sees the first one's change if it leaks.
	It("starts with the original value (A)", func() {
		Expect(conf.Server().UIWelcomeMessage).To(BeEmpty())
		conf.Update(func(c *conf.Options) { c.UIWelcomeMessage = "A" })
	})

	It("starts with the original value (B)", func() {
		Expect(conf.Server().UIWelcomeMessage).To(BeEmpty())
		conf.Update(func(c *conf.Options) { c.UIWelcomeMessage = "B" })
	})

	// Either container can run first: the second one sees the first one's change if it leaks.
	for _, name := range []string{"first", "second"} {
		Describe("in the "+name+" Ordered container", Ordered, func() {
			It("starts with the original value", func() {
				Expect(conf.Server().UIWelcomeMessage).To(BeEmpty())
				conf.Update(func(c *conf.Options) { c.UIWelcomeMessage = name })
			})

			It("keeps the change for the rest of the container", func() {
				Expect(conf.Server().UIWelcomeMessage).To(Equal(name))
			})
		})
	}

	Describe("in an Ordered container", Ordered, func() {
		BeforeAll(func() {
			conf.Update(func(c *conf.Options) { c.UIWelcomeMessage = "ordered" })
		})

		It("sees the BeforeAll change in the first spec", func() {
			Expect(conf.Server().UIWelcomeMessage).To(Equal("ordered"))
		})

		It("still sees the BeforeAll change in later specs", func() {
			Expect(conf.Server().UIWelcomeMessage).To(Equal("ordered"))
		})
	})
})
