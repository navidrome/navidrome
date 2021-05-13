package events

import (
	"context"

	"code.cloudfoundry.org/go-diodes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("diode", func() {
	var diode *diode
	var ctx context.Context
	var ctxCancel context.CancelFunc
	var missed int

	BeforeEach(func() {
		missed = 0
		ctx, ctxCancel = context.WithCancel(context.Background())
		diode = newDiode(ctx, 2, diodes.AlertFunc(func(m int) { missed = m }))
	})

	It("enqueues the data correctly", func() {
		diode.set(message{Data: "1"})
		diode.set(message{Data: "2"})
		Expect(diode.next()).To(Equal(&message{Data: "1"}))
		Expect(diode.next()).To(Equal(&message{Data: "2"}))
		Expect(missed).To(BeZero())
	})

	It("drops messages when diode is full", func() {
		diode.set(message{Data: "1"})
		diode.set(message{Data: "2"})
		diode.set(message{Data: "3"})
		next, ok := diode.tryNext()
		Expect(ok).To(BeTrue())
		Expect(next).To(Equal(&message{Data: "3"}))

		_, ok = diode.tryNext()
		Expect(ok).To(BeFalse())

		Expect(missed).To(Equal(2))
	})

	It("returns nil when diode is empty and the context is canceled", func() {
		diode.set(message{Data: "1"})
		ctxCancel()
		Expect(diode.next()).To(Equal(&message{Data: "1"}))
		Expect(diode.next()).To(BeNil())
	})
})
