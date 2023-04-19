package diodes_test

import (
	"context"
	"testing"

	"github.com/navidrome/navidrome/tests"
	. "github.com/navidrome/navidrome/utils/diodes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDiodes(t *testing.T) {
	tests.Init(t, false)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Diodes Suite")
}

var _ = Describe("Diode", func() {
	type message struct {
		data string
	}
	var diode *Diode[message]
	var ctx context.Context
	var ctxCancel context.CancelFunc
	var missed int

	BeforeEach(func() {
		missed = 0
		ctx, ctxCancel = context.WithCancel(context.Background())
		diode = New[message](ctx, 2, AlertFunc(func(m int) { missed = m }))
	})

	It("enqueues the data correctly", func() {
		diode.Put(message{data: "1"})
		diode.Put(message{data: "2"})
		Expect(diode.Next()).To(Equal(&message{data: "1"}))
		Expect(diode.Next()).To(Equal(&message{data: "2"}))
		Expect(missed).To(BeZero())
	})

	It("drops messages when Diode is full", func() {
		diode.Put(message{data: "1"})
		diode.Put(message{data: "2"})
		diode.Put(message{data: "3"})
		next, ok := diode.TryNext()
		Expect(ok).To(BeTrue())
		Expect(next).To(Equal(&message{data: "3"}))

		_, ok = diode.TryNext()
		Expect(ok).To(BeFalse())

		Expect(missed).To(Equal(2))
	})

	It("returns nil when Diode is empty and the context is canceled", func() {
		diode.Put(message{data: "1"})
		ctxCancel()
		Expect(diode.Next()).To(Equal(&message{data: "1"}))
		Expect(diode.Next()).To(BeNil())
	})
})
