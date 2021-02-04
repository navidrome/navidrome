package events

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event", func() {
	It("marshals Event to JSON", func() {
		testEvent := TestEvent{Test: "some data"}
		json := testEvent.Prepare(&testEvent)
		Expect(json).To(Equal(`{"name":"testEvent","Test":"some data"}`))
	})
})

type TestEvent struct {
	baseEvent
	Test string
}
