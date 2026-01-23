package persistence

import (
	"context"
	"slices"
	"strings"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("TagRepository", func() {
	var repo model.TagRepository
	var restRepo model.ResourceRepository
	var ctx context.Context

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: true})
		tagRepo := NewTagRepository(ctx, GetDBXBuilder())
		repo = tagRepo
		restRepo = tagRepo.(model.ResourceRepository)

		// Clean the database before each test to ensure isolation
		db := GetDBXBuilder()
		_, err := db.NewQuery("DELETE FROM tag").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("DELETE FROM library_tag").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Ensure library 1 exists (if it doesn't already)
		_, err = db.NewQuery("INSERT OR IGNORE INTO library (id, name, path, default_new_users) VALUES (1, 'Test Library', '/test', true)").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Ensure the admin user has access to library 1
		_, err = db.NewQuery("INSERT OR IGNORE INTO user_library (user_id, library_id) VALUES ('userid', 1)").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Add comprehensive test data that covers all test scenarios
		newTag := func(name, value string) model.Tag {
			return model.Tag{ID: id.NewTagID(name, value), TagName: model.TagName(name), TagValue: value}
		}

		err = repo.Add(1,
			// Genre tags
			newTag("genre", "rock"),
			newTag("genre", "pop"),
			newTag("genre", "jazz"),
			newTag("genre", "electronic"),
			newTag("genre", "classical"),
			newTag("genre", "ambient"),
			newTag("genre", "techno"),
			newTag("genre", "house"),
			newTag("genre", "trance"),
			newTag("genre", "Alternative Rock"),
			newTag("genre", "Blues"),
			newTag("genre", "Country"),
			// Mood tags
			newTag("mood", "happy"),
			newTag("mood", "sad"),
			newTag("mood", "energetic"),
			newTag("mood", "calm"),
			// Other tag types
			newTag("instrument", "guitar"),
			newTag("instrument", "piano"),
			newTag("decade", "1980s"),
			newTag("decade", "1990s"),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Add", func() {
		It("should handle adding new tags", func() {
			newTag := model.Tag{
				ID:       id.NewTagID("genre", "experimental"),
				TagName:  "genre",
				TagValue: "experimental",
			}

			err := repo.Add(1, newTag)
			Expect(err).ToNot(HaveOccurred())

			// Verify tag was added
			result, err := restRepo.Read(newTag.ID)
			Expect(err).ToNot(HaveOccurred())
			resultTag := result.(*model.Tag)
			Expect(resultTag.TagValue).To(Equal("experimental"))

			// Check count increased
			count, err := restRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(21))) // 20 from dataset + 1 new
		})

		It("should handle duplicate tags gracefully", func() {
			// Try to add a duplicate tag
			duplicateTag := model.Tag{
				ID:       id.NewTagID("genre", "rock"), // This already exists
				TagName:  "genre",
				TagValue: "rock",
			}

			count, err := restRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(20))) // Still 20 tags

			err = repo.Add(1, duplicateTag)
			Expect(err).ToNot(HaveOccurred()) // Should not error

			// Count should remain the same
			count, err = restRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(20))) // Still 20 tags
		})
	})

	Describe("UpdateCounts", func() {
		It("should update tag counts successfully", func() {
			err := repo.UpdateCounts()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle empty database gracefully", func() {
			// Clear the database first
			db := GetDBXBuilder()
			_, err := db.NewQuery("DELETE FROM tag").Execute()
			Expect(err).ToNot(HaveOccurred())

			err = repo.UpdateCounts()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle albums with non-existent tag IDs in JSON gracefully", func() {
			// Regression test for foreign key constraint error
			// Create an album with tag IDs in JSON that don't exist in tag table
			db := GetDBXBuilder()

			// First, create a non-existent tag ID (this simulates tags in JSON that aren't in tag table)
			nonExistentTagID := id.NewTagID("genre", "nonexistent-genre")

			// Create album with JSON containing the non-existent tag ID
			albumWithBadTags := `{"genre":[{"id":"` + nonExistentTagID + `","value":"nonexistent-genre"}]}`

			// Insert album directly into database with the problematic JSON
			_, err := db.NewQuery("INSERT INTO album (id, name, library_id, tags) VALUES ({:id}, {:name}, {:lib}, {:tags})").
				Bind(dbx.Params{
					"id":   "test-album-bad-tags",
					"name": "Album With Bad Tags",
					"lib":  1,
					"tags": albumWithBadTags,
				}).Execute()
			Expect(err).ToNot(HaveOccurred())

			// This should not fail with foreign key constraint error
			err = repo.UpdateCounts()
			Expect(err).ToNot(HaveOccurred())

			// Cleanup
			_, err = db.NewQuery("DELETE FROM album WHERE id = {:id}").
				Bind(dbx.Params{"id": "test-album-bad-tags"}).Execute()
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle media files with non-existent tag IDs in JSON gracefully", func() {
			// Regression test for foreign key constraint error with media files
			db := GetDBXBuilder()

			// Create a non-existent tag ID
			nonExistentTagID := id.NewTagID("genre", "another-nonexistent-genre")

			// Create media file with JSON containing the non-existent tag ID
			mediaFileWithBadTags := `{"genre":[{"id":"` + nonExistentTagID + `","value":"another-nonexistent-genre"}]}`

			// Insert media file directly into database with the problematic JSON
			_, err := db.NewQuery("INSERT INTO media_file (id, title, library_id, tags) VALUES ({:id}, {:title}, {:lib}, {:tags})").
				Bind(dbx.Params{
					"id":    "test-media-bad-tags",
					"title": "Media File With Bad Tags",
					"lib":   1,
					"tags":  mediaFileWithBadTags,
				}).Execute()
			Expect(err).ToNot(HaveOccurred())

			// This should not fail with foreign key constraint error
			err = repo.UpdateCounts()
			Expect(err).ToNot(HaveOccurred())

			// Cleanup
			_, err = db.NewQuery("DELETE FROM media_file WHERE id = {:id}").
				Bind(dbx.Params{"id": "test-media-bad-tags"}).Execute()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("Count", func() {
		It("should return correct count of tags", func() {
			count, err := restRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(20))) // From the test dataset
		})
	})

	Describe("Read", func() {
		It("should return existing tag", func() {
			rockID := id.NewTagID("genre", "rock")
			result, err := restRepo.Read(rockID)
			Expect(err).ToNot(HaveOccurred())
			resultTag := result.(*model.Tag)
			Expect(resultTag.ID).To(Equal(rockID))
			Expect(resultTag.TagName).To(Equal(model.TagName("genre")))
			Expect(resultTag.TagValue).To(Equal("rock"))
		})

		It("should return error for non-existent tag", func() {
			_, err := restRepo.Read("non-existent-id")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("ReadAll", func() {
		It("should return all tags from dataset", func() {
			result, err := restRepo.ReadAll()
			Expect(err).ToNot(HaveOccurred())
			tags := result.(model.TagList)
			Expect(tags).To(HaveLen(20))
		})

		It("should filter tags by partial value correctly", func() {
			options := rest.QueryOptions{
				Filters: map[string]interface{}{"name": "%rock%"}, // Tags containing 'rock'
			}
			result, err := restRepo.ReadAll(options)
			Expect(err).ToNot(HaveOccurred())
			tags := result.(model.TagList)
			Expect(tags).To(HaveLen(2)) // "rock" and "Alternative Rock"

			// Verify all returned tags contain 'rock' in their value
			for _, tag := range tags {
				Expect(strings.ToLower(tag.TagValue)).To(ContainSubstring("rock"))
			}
		})

		It("should filter tags by partial value using LIKE", func() {
			options := rest.QueryOptions{
				Filters: map[string]interface{}{"name": "%e%"}, // Tags containing 'e'
			}
			result, err := restRepo.ReadAll(options)
			Expect(err).ToNot(HaveOccurred())
			tags := result.(model.TagList)
			Expect(tags).To(HaveLen(8)) // electronic, house, trance, energetic, Blues, decade x2, Alternative Rock

			// Verify all returned tags contain 'e' in their value
			for _, tag := range tags {
				Expect(strings.ToLower(tag.TagValue)).To(ContainSubstring("e"))
			}
		})

		It("should sort tags by value ascending", func() {
			options := rest.QueryOptions{
				Filters: map[string]interface{}{"name": "%r%"}, // Tags containing 'r'
				Sort:    "name",
				Order:   "asc",
			}
			result, err := restRepo.ReadAll(options)
			Expect(err).ToNot(HaveOccurred())
			tags := result.(model.TagList)
			Expect(tags).To(HaveLen(7))

			Expect(slices.IsSortedFunc(tags, func(a, b model.Tag) int {
				return strings.Compare(strings.ToLower(a.TagValue), strings.ToLower(b.TagValue))
			}))
		})

		It("should sort tags by value descending", func() {
			options := rest.QueryOptions{
				Filters: map[string]interface{}{"name": "%r%"}, // Tags containing 'r'
				Sort:    "name",
				Order:   "desc",
			}
			result, err := restRepo.ReadAll(options)
			Expect(err).ToNot(HaveOccurred())
			tags := result.(model.TagList)
			Expect(tags).To(HaveLen(7))

			Expect(slices.IsSortedFunc(tags, func(a, b model.Tag) int {
				return strings.Compare(strings.ToLower(b.TagValue), strings.ToLower(a.TagValue)) // Descending order
			}))
		})
	})

	Describe("EntityName", func() {
		It("should return correct entity name", func() {
			name := restRepo.EntityName()
			Expect(name).To(Equal("tag"))
		})
	})

	Describe("NewInstance", func() {
		It("should return new tag instance", func() {
			instance := restRepo.NewInstance()
			Expect(instance).To(BeAssignableToTypeOf(model.Tag{}))
		})
	})
})
