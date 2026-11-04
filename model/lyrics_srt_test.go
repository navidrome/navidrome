package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseSRT", func() {
	It("parses SRT blocks with the default language", func() {
		content := []byte("1\n00:00:01,000 --> 00:00:02,000\nFirst subtitle\n\n2\n00:00:03,000 --> 00:00:04,000\nSecond subtitle")

		list, err := parseSRT("xxx", content)

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(HaveLen(1))
		Expect(list[0].Lang).To(Equal("xxx"))
		Expect(list[0].Synced).To(BeTrue())
		Expect(list[0].Line).To(Equal([]Line{
			{Start: new(int64(1000)), End: new(int64(2000)), Value: "First subtitle"},
			{Start: new(int64(3000)), End: new(int64(4000)), Value: "Second subtitle"},
		}))
	})

	It("returns nil for input with no valid blocks", func() {
		list, err := parseSRT("xxx", []byte("not actually an srt file"))

		Expect(err).ToNot(HaveOccurred())
		Expect(list).To(BeNil())
	})

	It("preserves zero durations, overlaps, and document order", func() {
		content := []byte("1\n00:00:05,000 --> 00:00:05,000\nMarker\n\n2\n00:00:03,000 --> 00:00:06,000\nEarlier overlapping line")

		list, err := parseSRT("eng", content)

		Expect(err).ToNot(HaveOccurred())
		Expect(list[0].Line).To(Equal([]Line{
			{Start: new(int64(5000)), End: new(int64(5000)), Value: "Marker"},
			{Start: new(int64(3000)), End: new(int64(6000)), Value: "Earlier overlapping line"},
		}))
	})

	DescribeTable("rejects invalid SRT timing",
		func(timing string) {
			list, err := parseSRT("eng", []byte("1\n"+timing+"\nInvalid"))
			Expect(err).To(HaveOccurred())
			Expect(list).To(BeNil())
		},
		Entry("negative duration", "00:00:02,000 --> 00:00:01,999"),
		Entry("minute out of range", "00:60:00,000 --> 01:00:01,000"),
		Entry("second out of range", "00:00:60,000 --> 00:01:01,000"),
	)
})
