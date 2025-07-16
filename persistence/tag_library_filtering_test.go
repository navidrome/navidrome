package persistence

import (
	"context"

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

var _ = Describe("Tag Library Filtering", func() {

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())

		// Clean up all relevant tables
		db := GetDBXBuilder()
		_, err := db.NewQuery("DELETE FROM library_tag").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("DELETE FROM tag").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("DELETE FROM user_library WHERE user_id != 'userid' AND user_id != '2222'").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("DELETE FROM library WHERE id > 1").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Create test libraries
		_, err = db.NewQuery("INSERT INTO library (id, name, path) VALUES (2, 'Library 2', '/music/lib2')").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("INSERT INTO library (id, name, path) VALUES (3, 'Library 3', '/music/lib3')").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Ensure admin user has access to all libraries (since admin users should have access to all libraries)
		_, err = db.NewQuery("INSERT OR IGNORE INTO user_library (user_id, library_id) VALUES ('userid', 1)").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("INSERT OR IGNORE INTO user_library (user_id, library_id) VALUES ('userid', 2)").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("INSERT OR IGNORE INTO user_library (user_id, library_id) VALUES ('userid', 3)").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Set up test tags
		newTag := func(name, value string) model.Tag {
			return model.Tag{ID: id.NewTagID(name, value), TagName: model.TagName(name), TagValue: value}
		}

		// Create tags in admin context
		adminCtx := request.WithUser(log.NewContext(context.TODO()), adminUser)
		tagRepo := NewTagRepository(adminCtx, GetDBXBuilder())

		// Add tags to different libraries
		err = tagRepo.Add(1, newTag("genre", "rock"))
		Expect(err).ToNot(HaveOccurred())
		err = tagRepo.Add(2, newTag("genre", "pop"))
		Expect(err).ToNot(HaveOccurred())
		err = tagRepo.Add(3, newTag("genre", "jazz"))
		Expect(err).ToNot(HaveOccurred())
		err = tagRepo.Add(2, newTag("genre", "rock"))
		Expect(err).ToNot(HaveOccurred())

		// Update counts manually for testing
		_, err = db.NewQuery("UPDATE library_tag SET album_count = 5, media_file_count = 20 WHERE tag_id = {:tagId} AND library_id = 1").Bind(dbx.Params{"tagId": id.NewTagID("genre", "rock")}).Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("UPDATE library_tag SET album_count = 3, media_file_count = 10 WHERE tag_id = {:tagId} AND library_id = 2").Bind(dbx.Params{"tagId": id.NewTagID("genre", "pop")}).Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("UPDATE library_tag SET album_count = 2, media_file_count = 8 WHERE tag_id = {:tagId} AND library_id = 3").Bind(dbx.Params{"tagId": id.NewTagID("genre", "jazz")}).Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("UPDATE library_tag SET album_count = 1, media_file_count = 4 WHERE tag_id = {:tagId} AND library_id = 2").Bind(dbx.Params{"tagId": id.NewTagID("genre", "rock")}).Execute()
		Expect(err).ToNot(HaveOccurred())

		// Set up user library access - Regular user has access to libraries 1 and 2 only
		_, err = db.NewQuery("INSERT INTO user_library (user_id, library_id) VALUES ('2222', 2)").Execute()
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("TagRepository Library Filtering", func() {
		Context("Admin User", func() {
			It("should see all tags regardless of library", func() {
				ctx := request.WithUser(log.NewContext(context.TODO()), adminUser)
				tagRepo := NewTagRepository(ctx, GetDBXBuilder())
				repo := tagRepo.(model.ResourceRepository)

				tags, err := repo.ReadAll()
				Expect(err).ToNot(HaveOccurred())
				tagList := tags.(model.TagList)
				Expect(tagList).To(HaveLen(3))
			})
		})

		Context("Regular User with Limited Library Access", func() {
			It("should only see tags from accessible libraries", func() {
				ctx := request.WithUser(log.NewContext(context.TODO()), regularUser)
				tagRepo := NewTagRepository(ctx, GetDBXBuilder())
				repo := tagRepo.(model.ResourceRepository)

				tags, err := repo.ReadAll()
				Expect(err).ToNot(HaveOccurred())
				tagList := tags.(model.TagList)

				// Should see rock (libraries 1,2) and pop (library 2), but not jazz (library 3)
				Expect(tagList).To(HaveLen(2))
			})

			It("should respect explicit library_id filters within accessible libraries", func() {
				ctx := request.WithUser(log.NewContext(context.TODO()), regularUser)
				tagRepo := NewTagRepository(ctx, GetDBXBuilder())
				repo := tagRepo.(model.ResourceRepository)

				// Filter by library 2 (user has access to libraries 1 and 2)
				tags, err := repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"library_id": 2,
					},
				})
				Expect(err).ToNot(HaveOccurred())
				tagList := tags.(model.TagList)

				// Should see only tags from library 2: pop and rock(lib2)
				Expect(tagList).To(HaveLen(2))

				// Verify the tags are correct
				tagValues := make([]string, len(tagList))
				for i, tag := range tagList {
					tagValues[i] = tag.TagValue
				}
				Expect(tagValues).To(ContainElements("pop", "rock"))
			})

			It("should not return tags when filtering by inaccessible library", func() {
				ctx := request.WithUser(log.NewContext(context.TODO()), regularUser)
				tagRepo := NewTagRepository(ctx, GetDBXBuilder())
				repo := tagRepo.(model.ResourceRepository)

				// Try to filter by library 3 (user doesn't have access)
				tags, err := repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"library_id": 3,
					},
				})
				Expect(err).ToNot(HaveOccurred())
				tagList := tags.(model.TagList)

				// Should return no tags since user can't access library 3
				Expect(tagList).To(HaveLen(0))
			})

			It("should filter by library 1 correctly", func() {
				ctx := request.WithUser(log.NewContext(context.TODO()), regularUser)
				tagRepo := NewTagRepository(ctx, GetDBXBuilder())
				repo := tagRepo.(model.ResourceRepository)

				// Filter by library 1 (user has access)
				tags, err := repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"library_id": 1,
					},
				})
				Expect(err).ToNot(HaveOccurred())
				tagList := tags.(model.TagList)

				// Should see only rock from library 1
				Expect(tagList).To(HaveLen(1))
				Expect(tagList[0].TagValue).To(Equal("rock"))
			})
		})

		Context("Admin User with Explicit Library Filtering", func() {
			It("should see all tags when no filter is applied", func() {
				adminCtx := request.WithUser(log.NewContext(context.TODO()), adminUser)
				tagRepo := NewTagRepository(adminCtx, GetDBXBuilder())
				repo := tagRepo.(model.ResourceRepository)

				tags, err := repo.ReadAll()
				Expect(err).ToNot(HaveOccurred())
				tagList := tags.(model.TagList)
				Expect(tagList).To(HaveLen(3))
			})

			It("should respect explicit library_id filters", func() {
				adminCtx := request.WithUser(log.NewContext(context.TODO()), adminUser)
				tagRepo := NewTagRepository(adminCtx, GetDBXBuilder())
				repo := tagRepo.(model.ResourceRepository)

				// Filter by library 3
				tags, err := repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"library_id": 3,
					},
				})
				Expect(err).ToNot(HaveOccurred())
				tagList := tags.(model.TagList)

				// Should see only jazz from library 3
				Expect(tagList).To(HaveLen(1))
				Expect(tagList[0].TagValue).To(Equal("jazz"))
			})

			It("should filter by library 2 correctly", func() {
				adminCtx := request.WithUser(log.NewContext(context.TODO()), adminUser)
				tagRepo := NewTagRepository(adminCtx, GetDBXBuilder())
				repo := tagRepo.(model.ResourceRepository)

				// Filter by library 2
				tags, err := repo.ReadAll(rest.QueryOptions{
					Filters: map[string]interface{}{
						"library_id": 2,
					},
				})
				Expect(err).ToNot(HaveOccurred())
				tagList := tags.(model.TagList)

				// Should see pop and rock from library 2
				Expect(tagList).To(HaveLen(2))

				tagValues := make([]string, len(tagList))
				for i, tag := range tagList {
					tagValues[i] = tag.TagValue
				}
				Expect(tagValues).To(ContainElements("pop", "rock"))
			})
		})
	})
})
