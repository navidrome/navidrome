package metadata

import (
	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("sanitize truncation with a tiny MaxLength",
	// A value of only continuation bytes drains the partial-rune loop to empty; the loop
	// must stop there instead of slicing value[:-1] and panicking.
	func(maxLength int, value string) {
		Expect(func() {
			Expect(sanitize("file.mp3", "title", model.TagConf{MaxLength: maxLength}, value)).To(Equal(""))
		}).NotTo(Panic())
	},
	Entry("maxLength 1", 1, "\x80\x80"),
	Entry("maxLength 2", 2, "\x80\x80\x80"),
	Entry("maxLength 3", 3, "\x80\x80\x80\x80"),
)
