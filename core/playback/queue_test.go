package playback

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Queues", func() {
	var queue *Queue

	BeforeEach(func() {
		queue = NewQueue()
	})

	Describe("use empty queue", func() {
		It("is empty", func() {
			Expect(queue.Items).To(BeEmpty())
			Expect(queue.Index).To(Equal(-1))
		})

	})
})
