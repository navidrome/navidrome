package log

import (
	"testing"

	"github.com/sirupsen/logrus"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRingBuffer(t *testing.T) {
	Convey("Given an empty RingBuffer", t, func() {
		capacity := 5
		rb := NewRingBuffer(capacity)
		
		Convey("It should have no entries", func() {
			So(rb.GetCount(), ShouldEqual, 0)
			So(rb.GetAll(), ShouldHaveLength, 0)
		})
		
		Convey("When adding entries", func() {
			entry1 := &logrus.Entry{Message: "entry1"}
			entry2 := &logrus.Entry{Message: "entry2"}
			
			rb.Add(entry1)
			rb.Add(entry2)
			
			Convey("It should store the entries in order", func() {
				So(rb.GetCount(), ShouldEqual, 2)
				entries := rb.GetAll()
				So(entries, ShouldHaveLength, 2)
				So(entries[0].Message, ShouldEqual, "entry1")
				So(entries[1].Message, ShouldEqual, "entry2")
			})
		})
		
		Convey("When adding more entries than capacity", func() {
			for i := 0; i < capacity+2; i++ {
				rb.Add(&logrus.Entry{Message: "entry" + string(rune('A'+i))})
			}
			
			Convey("It should only store the most recent entries", func() {
				So(rb.GetCount(), ShouldEqual, capacity)
				entries := rb.GetAll()
				So(entries, ShouldHaveLength, capacity)
				// First entry should be C, since A and B were pushed out
				So(entries[0].Message, ShouldEqual, "entryC")
				So(entries[capacity-1].Message, ShouldEqual, "entryG")
			})
		})
		
		Convey("After clearing", func() {
			rb.Add(&logrus.Entry{Message: "entry1"})
			rb.Clear()
			
			Convey("It should have no entries", func() {
				So(rb.GetCount(), ShouldEqual, 0)
				So(rb.GetAll(), ShouldHaveLength, 0)
			})
		})
	})
}