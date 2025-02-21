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
					"Genre": {"pop", "rock"},
				}
				Expect(tags1.Hash()).To(Equal(tags2.Hash()))
			})
			It("should return different values for different tags", func() {
				tags1 := Tags{
					"genre": {"Rock", "Pop"},
				}
				tags2 := Tags{
					"artist": {"The Beatles"},
				}
				Expect(tags1.Hash()).ToNot(Equal(tags2.Hash()))
			})
		})
	})

	Describe("TagList", func() {
		Describe("GroupByFrequency", func() {
			It("should return an empty Tags map for an empty TagList", func() {
				tagList := TagList{}

				groupedTags := tagList.GroupByFrequency()

				Expect(groupedTags).To(BeEmpty())
			})

			It("should handle tags with different frequencies correctly", func() {
				tagList := TagList{
					NewTag("genre", "Jazz"),
					NewTag("genre", "Rock"),
					NewTag("genre", "Pop"),
					NewTag("genre", "Rock"),
					NewTag("artist", "The Rolling Stones"),
					NewTag("artist", "The Beatles"),
					NewTag("artist", "The Beatles"),
				}

				groupedTags := tagList.GroupByFrequency()

				Expect(groupedTags).To(HaveKeyWithValue(TagName("genre"), []string{"Rock", "Jazz", "Pop"}))
				Expect(groupedTags).To(HaveKeyWithValue(TagName("artist"), []string{"The Beatles", "The Rolling Stones"}))
			})

			It("should sort tags by name when frequency is the same", func() {
				tagList := TagList{
					NewTag("genre", "Jazz"),
					NewTag("genre", "Rock"),
					NewTag("genre", "Alternative"),
					NewTag("genre", "Pop"),
				}

				groupedTags := tagList.GroupByFrequency()

				Expect(groupedTags).To(HaveKeyWithValue(TagName("genre"), []string{"Alternative", "Jazz", "Pop", "Rock"}))
			})
			It("should normalize casing", func() {
				tagList := TagList{
					NewTag("genre", "Synthwave"),
					NewTag("genre", "synthwave"),
				}

				groupedTags := tagList.GroupByFrequency()

				Expect(groupedTags).To(HaveKeyWithValue(TagName("genre"), []string{"synthwave"}))
			})
		})
	})
})
