package model

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tag", func() {
	Describe("NewTag", func() {
		It("should create a new tag", func() {
			tag := NewTag("genre", "Rock")
			tag2 := NewTag("Genre", "Rock")
			tag3 := NewTag("Genre", "rock")
			Expect(tag2.ID).To(Equal(tag.ID))
			Expect(tag3.ID).To(Equal(tag.ID))
		})
	})

	Describe("Tags", func() {
		var tags Tags
		BeforeEach(func() {
			tags = Tags{
				"genre":  {"Rock", "Pop"},
				"artist": {"The Beatles"},
			}
		})
		It("should flatten tags by name", func() {
			flat := tags.Flatten("genre")
			Expect(flat).To(ConsistOf(
				NewTag("genre", "Rock"),
				NewTag("genre", "Pop"),
			))
		})
		It("should flatten tags", func() {
			flat := tags.FlattenAll()
			Expect(flat).To(ConsistOf(
				NewTag("genre", "Rock"),
				NewTag("genre", "Pop"),
				NewTag("artist", "The Beatles"),
			))
		})
		It("should get values by name", func() {
			Expect(tags.Values("genre")).To(ConsistOf("Rock", "Pop"))
			Expect(tags.Values("artist")).To(ConsistOf("The Beatles"))
		})

		Describe("Hash", func() {
			It("should always return the same value for the same tags ", func() {
				tags1 := Tags{
					"genre": {"Rock", "Pop"},
				}
				tags2 := Tags{
					"Genre": {"rock", "pop"},
				}
				Expect(tags1.Hash()).To(Equal(tags2.Hash()))
			})
		})
	})
})
