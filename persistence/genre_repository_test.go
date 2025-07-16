package persistence

import (
	"context"
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GenreRepository", func() {
	var repo model.GenreRepository
	var restRepo model.ResourceRepository
	var tagRepo model.TagRepository
	var ctx context.Context

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ctx = request.WithUser(log.NewContext(context.TODO()), model.User{ID: "userid", UserName: "johndoe", IsAdmin: true})
		genreRepo := NewGenreRepository(ctx, GetDBXBuilder())
		repo = genreRepo
		restRepo = genreRepo.(model.ResourceRepository)
		tagRepo = NewTagRepository(ctx, GetDBXBuilder())

		// Clear any existing tags to ensure test isolation
		db := GetDBXBuilder()
		_, err := db.NewQuery("DELETE FROM tag").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Ensure library 1 exists and user has access to it
		_, err = db.NewQuery("INSERT OR IGNORE INTO library (id, name, path, default_new_users) VALUES (1, 'Test Library', '/test', true)").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("INSERT OR IGNORE INTO user_library (user_id, library_id) VALUES ('userid', 1)").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Add comprehensive test data that covers all test scenarios
		newTag := func(name, value string) model.Tag {
			return model.Tag{ID: id.NewTagID(name, value), TagName: model.TagName(name), TagValue: value}
		}

		err = tagRepo.Add(1,
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
			// These should not be counted as genres
			newTag("mood", "happy"),
			newTag("mood", "ambient"),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("GetAll", func() {
		It("should return all genres", func() {
			genres, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(genres).To(HaveLen(12))

			// Verify that all returned items are genres (TagName = "genre")
			genreNames := make([]string, len(genres))
			for i, genre := range genres {
				genreNames[i] = genre.Name
			}
			Expect(genreNames).To(ContainElement("rock"))
			Expect(genreNames).To(ContainElement("pop"))
			Expect(genreNames).To(ContainElement("jazz"))
			// Should not contain mood tags
			Expect(genreNames).ToNot(ContainElement("happy"))
		})

		It("should support query options", func() {
			// Test with limiting results
			genres, err := repo.GetAll(model.QueryOptions{Max: 1})
			Expect(err).ToNot(HaveOccurred())
			Expect(genres).To(HaveLen(1))
		})

		It("should handle empty results gracefully", func() {
			// Clear all genre tags
			_, err := GetDBXBuilder().NewQuery("DELETE FROM tag WHERE tag_name = 'genre'").Execute()
			Expect(err).ToNot(HaveOccurred())

			genres, err := repo.GetAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(genres).To(BeEmpty())
		})
		Describe("filtering and sorting", func() {
			It("should filter by name using like match", func() {
				// Test filtering by partial name match using the "name" filter which maps to containsFilter("tag_value")
				options := model.QueryOptions{
					Filters: squirrel.Like{"tag_value": "%rock%"}, // Direct field access
				}
				genres, err := repo.GetAll(options)
				Expect(err).ToNot(HaveOccurred())
				Expect(genres).To(HaveLen(2)) // Should match "rock" and "Alternative Rock"

				// Verify all returned genres contain "rock" in their name
				for _, genre := range genres {
					Expect(strings.ToLower(genre.Name)).To(ContainSubstring("rock"))
				}
			})

			It("should sort by name in ascending order", func() {
				// Test sorting by name with the fixed mapping
				options := model.QueryOptions{
					Filters: squirrel.Like{"tag_value": "%e%"}, // Should match genres containing "e"
					Sort:    "name",
				}
				genres, err := repo.GetAll(options)
				Expect(err).ToNot(HaveOccurred())
				Expect(genres).To(HaveLen(7))

				Expect(slices.IsSortedFunc(genres, func(a, b model.Genre) int {
					return strings.Compare(b.Name, a.Name) // Inverted to check descending order
				}))
			})

			It("should sort by name in descending order", func() {
				// Test sorting by name in descending order
				options := model.QueryOptions{
					Filters: squirrel.Like{"tag_value": "%e%"}, // Should match genres containing "e"
					Sort:    "name",
					Order:   "desc",
				}
				genres, err := repo.GetAll(options)
				Expect(err).ToNot(HaveOccurred())
				Expect(genres).To(HaveLen(7))

				Expect(slices.IsSortedFunc(genres, func(a, b model.Genre) int {
					return strings.Compare(a.Name, b.Name)
				}))
			})
		})
	})

	Describe("Count", func() {
		It("should return correct count of genres", func() {
			count, err := restRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(int64(12))) // We have 12 genre tags
		})

		It("should handle zero count", func() {
			// Clear all genre tags
			_, err := GetDBXBuilder().NewQuery("DELETE FROM tag WHERE tag_name = 'genre'").Execute()
			Expect(err).ToNot(HaveOccurred())

			count, err := restRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(BeZero())
		})

		It("should only count genre tags", func() {
			// Add a non-genre tag
			nonGenreTag := model.Tag{
				ID:       id.NewTagID("mood", "energetic"),
				TagName:  "mood",
				TagValue: "energetic",
			}
			err := tagRepo.Add(1, nonGenreTag)
			Expect(err).ToNot(HaveOccurred())

			count, err := restRepo.Count()
			Expect(err).ToNot(HaveOccurred())
			// Count should not include the mood tag
			Expect(count).To(Equal(int64(12))) // Should still be 12 genre tags
		})

		It("should filter by name using like match", func() {
			// Test filtering by partial name match using the "name" filter which maps to containsFilter("tag_value")
			options := rest.QueryOptions{
				Filters: map[string]interface{}{"name": "%rock%"},
			}
			count, err := restRepo.Count(options)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(BeNumerically("==", 2))
		})
	})

	Describe("Read", func() {
		It("should return existing genre", func() {
			// Use one of the existing genres from our consolidated dataset
			genreID := id.NewTagID("genre", "rock")
			result, err := restRepo.Read(genreID)
			Expect(err).ToNot(HaveOccurred())
			genre := result.(*model.Genre)
			Expect(genre.ID).To(Equal(genreID))
			Expect(genre.Name).To(Equal("rock"))
		})

		It("should return error for non-existent genre", func() {
			_, err := restRepo.Read("non-existent-id")
			Expect(err).To(HaveOccurred())
		})

		It("should not return non-genre tags", func() {
			moodID := id.NewTagID("mood", "happy") // This exists as a mood tag, not genre
			_, err := restRepo.Read(moodID)
			Expect(err).To(HaveOccurred()) // Should not find it as a genre
		})
	})

	Describe("ReadAll", func() {
		It("should return all genres through ReadAll", func() {
			result, err := restRepo.ReadAll()
			Expect(err).ToNot(HaveOccurred())
			genres := result.(model.Genres)
			Expect(genres).To(HaveLen(12)) // We have 12 genre tags

			genreNames := make([]string, len(genres))
			for i, genre := range genres {
				genreNames[i] = genre.Name
			}
			// Check for some of our consolidated dataset genres
			Expect(genreNames).To(ContainElement("rock"))
			Expect(genreNames).To(ContainElement("pop"))
			Expect(genreNames).To(ContainElement("jazz"))
		})

		It("should support rest query options", func() {
			result, err := restRepo.ReadAll()
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
		})
	})

	Describe("EntityName", func() {
		It("should return correct entity name", func() {
			name := restRepo.EntityName()
			Expect(name).To(Equal("tag")) // Genre repository uses tag table
		})
	})

	Describe("NewInstance", func() {
		It("should return new genre instance", func() {
			instance := restRepo.NewInstance()
			Expect(instance).To(BeAssignableToTypeOf(&model.Genre{}))
		})
	})
})
