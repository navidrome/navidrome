package events

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Event", func() {
	It("marshals Event to JSON", func() {
		testEvent := DummyEvent{Text: "some data"}
		data := testEvent.Data(&testEvent)
		Expect(data).To(Equal(`{"Text":"some data"}`))
		name := testEvent.Name(&testEvent)
		Expect(name).To(Equal("dummyEvent"))
	})
})

type DummyEvent struct {
	baseEvent
	Text string
}
