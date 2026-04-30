package criteria

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = Describe("fields", func() {
	Describe("LookupField", func() {
		It("finds built-in fields case-insensitively", func() {
			field, ok := LookupField("Title")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("title"))
		})

		It("resolves aliases to their canonical field name", func() {
			field, ok := LookupField("albumtype")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("releasetype"))
			gomega.Expect(field.IsTag).To(gomega.BeTrue())
		})

		It("finds registered tag names", func() {
			AddTagNames([]string{"task3_mood"})

			field, ok := LookupField("task3_mood")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("task3_mood"))
			gomega.Expect(field.IsTag).To(gomega.BeTrue())
		})

		It("marks registered numeric tags", func() {
			AddTagNames([]string{"task3_score"})
			AddNumericTags([]string{"task3_score"})

			field, ok := LookupField("task3_score")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.IsTag).To(gomega.BeTrue())
			gomega.Expect(field.Numeric).To(gomega.BeTrue())
		})

		It("finds registered roles", func() {
			AddRoles([]string{"task3_producer"})

			field, ok := LookupField("task3_producer")

			gomega.Expect(ok).To(gomega.BeTrue())
			gomega.Expect(field.Name()).To(gomega.Equal("task3_producer"))
			gomega.Expect(field.IsRole).To(gomega.BeTrue())
		})
	})
})
