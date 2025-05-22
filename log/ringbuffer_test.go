package log

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("RingBuffer", func() {
	var rb *RingBuffer

	BeforeEach(func() {
		rb = NewRingBuffer(5)
	})

	It("should have no entries when empty", func() {
		Expect(rb.GetCount()).To(Equal(0))
		Expect(rb.GetAll()).To(HaveLen(0))
	})

	Context("when adding entries", func() {
		BeforeEach(func() {
			entry1 := &logrus.Entry{Message: "entry1"}
			entry2 := &logrus.Entry{Message: "entry2"}
			rb.Add(entry1)
			rb.Add(entry2)
		})

		It("should store the entries in order", func() {
			Expect(rb.GetCount()).To(Equal(2))
			entries := rb.GetAll()
			Expect(entries).To(HaveLen(2))
			Expect(entries[0].Message).To(Equal("entry1"))
			Expect(entries[1].Message).To(Equal("entry2"))
		})
	})

	Context("when adding more entries than capacity", func() {
		BeforeEach(func() {
			for i := 0; i < 7; i++ {
				rb.Add(&logrus.Entry{Message: "entry" + string(rune('A'+i))})
			}
		})

		It("should only store the most recent entries", func() {
			Expect(rb.GetCount()).To(Equal(5))
			entries := rb.GetAll()
			Expect(entries).To(HaveLen(5))
			// First entry should be C, since A and B were pushed out
			Expect(entries[0].Message).To(Equal("entryC"))
			Expect(entries[4].Message).To(Equal("entryG"))
		})
	})

	Context("after clearing", func() {
		BeforeEach(func() {
			rb.Add(&logrus.Entry{Message: "entry1"})
			rb.Clear()
		})

		It("should have no entries", func() {
			Expect(rb.GetCount()).To(Equal(0))
			Expect(rb.GetAll()).To(HaveLen(0))
		})
	})
})
