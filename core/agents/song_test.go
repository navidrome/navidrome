package agents

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Song.Equals", func() {
	base := Song{ID: "1", Name: "S", Artists: []Artist{{ID: "x", Name: "A"}}}
	It("true for identical songs incl Artists", func() {
		Expect(base.Equals(base)).To(BeTrue())
	})
	It("false when Artists differ", func() {
		other := base
		other.Artists = []Artist{{ID: "y", Name: "B"}}
		Expect(base.Equals(other)).To(BeFalse())
	})
	It("false when a scalar differs", func() {
		other := base
		other.Name = "T"
		Expect(base.Equals(other)).To(BeFalse())
	})
	It("true when both have empty Artists and equal scalars", func() {
		a := Song{ID: "1", Name: "S"}
		Expect(a.Equals(a)).To(BeTrue())
	})
})
