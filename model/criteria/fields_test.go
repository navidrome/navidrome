package criteria

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("fields", func() {
	Describe("mapFields", func() {
		It("ignores random fields", func() {
			m := map[string]any{"random": "123"}
			m = mapFields(m)
			gomega.Expect(m).To(gomega.BeEmpty())
		})
	})
})
