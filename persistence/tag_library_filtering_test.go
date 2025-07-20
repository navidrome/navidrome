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

const (
	adminUserID   = "userid"
	regularUserID = "2222"
	libraryID1    = 1
	libraryID2    = 2
	libraryID3    = 3

	tagNameGenre = "genre"
	tagValueRock = "rock"
	tagValuePop  = "pop"
	tagValueJazz = "jazz"
)

var _ = Describe("Tag Library Filtering", func() {
	var (
		tagRockID = id.NewTagID(tagNameGenre, tagValueRock)
		tagPopID  = id.NewTagID(tagNameGenre, tagValuePop)
		tagJazzID = id.NewTagID(tagNameGenre, tagValueJazz)
	)

	expectTagValues := func(tagList model.TagList, expected []string) {
		tagValues := make([]string, len(tagList))
		for i, tag := range tagList {
			tagValues[i] = tag.TagValue
		}
		Expect(tagValues).To(ContainElements(expected))
	}

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())

		// Clean up database
		db := GetDBXBuilder()
		_, err := db.NewQuery("DELETE FROM library_tag").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("DELETE FROM tag").Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("DELETE FROM user_library WHERE user_id != {:admin} AND user_id != {:regular}").
			Bind(dbx.Params{"admin": adminUserID, "regular": regularUserID}).Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("DELETE FROM library WHERE id > 1").Execute()
		Expect(err).ToNot(HaveOccurred())

		// Create test libraries
		_, err = db.NewQuery("INSERT INTO library (id, name, path) VALUES ({:id}, {:name}, {:path})").
			Bind(dbx.Params{"id": libraryID2, "name": "Library 2", "path": "/music/lib2"}).Execute()
		Expect(err).ToNot(HaveOccurred())
		_, err = db.NewQuery("INSERT INTO library (id, name, path) VALUES ({:id}, {:name}, {:path})").
			Bind(dbx.Params{"id": libraryID3, "name": "Library 3", "path": "/music/lib3"}).Execute()
		Expect(err).ToNot(HaveOccurred())

		// Give admin access to all libraries
		for _, libID := range []int{libraryID1, libraryID2, libraryID3} {
			_, err = db.NewQuery("INSERT OR IGNORE INTO user_library (user_id, library_id) VALUES ({:user}, {:lib})").
				Bind(dbx.Params{"user": adminUserID, "lib": libID}).Execute()
			Expect(err).ToNot(HaveOccurred())
		}

		// Create test tags
		adminCtx := request.WithUser(log.NewContext(context.TODO()), adminUser)
		tagRepo := NewTagRepository(adminCtx, GetDBXBuilder())

		createTag := func(libraryID int, name, value string) {
			tag := model.Tag{ID: id.NewTagID(name, value), TagName: model.TagName(name), TagValue: value}
			err := tagRepo.Add(libraryID, tag)
			Expect(err).ToNot(HaveOccurred())
		}

		createTag(libraryID1, tagNameGenre, tagValueRock)
		createTag(libraryID2, tagNameGenre, tagValuePop)
		createTag(libraryID3, tagNameGenre, tagValueJazz)
		createTag(libraryID2, tagNameGenre, tagValueRock) // Rock appears in both lib1 and lib2

		// Set tag counts (manually for testing)
		setCounts := func(tagID string, libID, albums, songs int) {
			_, err := db.NewQuery("UPDATE library_tag SET album_count = {:albums}, media_file_count = {:songs} WHERE tag_id = {:tag} AND library_id = {:lib}").
				Bind(dbx.Params{"albums": albums, "songs": songs, "tag": tagID, "lib": libID}).Execute()
			Expect(err).ToNot(HaveOccurred())
		}

		setCounts(tagRockID, libraryID1, 5, 20)
		setCounts(tagPopID, libraryID2, 3, 10)
		setCounts(tagJazzID, libraryID3, 2, 8)
		setCounts(tagRockID, libraryID2, 1, 4)

		// Give regular user access to library 2 only
		_, err = db.NewQuery("INSERT INTO user_library (user_id, library_id) VALUES ({:user}, {:lib})").
			Bind(dbx.Params{"user": regularUserID, "lib": libraryID2}).Execute()
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("TagRepository Library Filtering", func() {
		// Helper to create repository and read all tags
		readAllTags := func(user *model.User, filters ...rest.QueryOptions) model.TagList {
			var ctx context.Context
			if user != nil {
				ctx = request.WithUser(log.NewContext(context.TODO()), *user)
			} else {
				ctx = context.Background() // Headless context
			}

			tagRepo := NewTagRepository(ctx, GetDBXBuilder())
			repo := tagRepo.(model.ResourceRepository)

			var opts rest.QueryOptions
			if len(filters) > 0 {
				opts = filters[0]
			}

			tags, err := repo.ReadAll(opts)
			Expect(err).ToNot(HaveOccurred())
			return tags.(model.TagList)
		}

		// Helper to count tags
		countTags := func(user *model.User) int64 {
			var ctx context.Context
			if user != nil {
				ctx = request.WithUser(log.NewContext(context.TODO()), *user)
			} else {
				ctx = context.Background()
			}

			tagRepo := NewTagRepository(ctx, GetDBXBuilder())
			repo := tagRepo.(model.ResourceRepository)

			count, err := repo.Count()
			Expect(err).ToNot(HaveOccurred())
			return count
		}

		Context("Admin User", func() {
			It("should see all tags regardless of library", func() {
				tags := readAllTags(&adminUser)
				Expect(tags).To(HaveLen(3))
			})
		})

		Context("Regular User with Limited Library Access", func() {
			It("should only see tags from accessible libraries", func() {
				tags := readAllTags(&regularUser)
				// Should see rock (libraries 1,2) and pop (library 2), but not jazz (library 3)
				Expect(tags).To(HaveLen(2))
			})

			It("should respect explicit library_id filters within accessible libraries", func() {
				tags := readAllTags(&regularUser, rest.QueryOptions{
					Filters: map[string]interface{}{"library_id": libraryID2},
				})
				// Should see only tags from library 2: pop and rock(lib2)
				Expect(tags).To(HaveLen(2))
				expectTagValues(tags, []string{tagValuePop, tagValueRock})
			})

			It("should not return tags when filtering by inaccessible library", func() {
				tags := readAllTags(&regularUser, rest.QueryOptions{
					Filters: map[string]interface{}{"library_id": libraryID3},
				})
				// Should return no tags since user can't access library 3
				Expect(tags).To(HaveLen(0))
			})

			It("should filter by library 1 correctly", func() {
				tags := readAllTags(&regularUser, rest.QueryOptions{
					Filters: map[string]interface{}{"library_id": libraryID1},
				})
				// Should see only rock from library 1
				Expect(tags).To(HaveLen(1))
				Expect(tags[0].TagValue).To(Equal(tagValueRock))
			})
		})

		Context("Headless Processes (No User Context)", func() {
			It("should see all tags from all libraries when no user is in context", func() {
				tags := readAllTags(nil) // nil = headless context
				// Should see all tags from all libraries (no filtering applied)
				Expect(tags).To(HaveLen(3))
				expectTagValues(tags, []string{tagValueRock, tagValuePop, tagValueJazz})
			})

			It("should count all tags from all libraries when no user is in context", func() {
				count := countTags(nil)
				// Should count all tags from all libraries
				Expect(count).To(Equal(int64(3)))
			})

			It("should calculate proper statistics from all libraries for headless processes", func() {
				tags := readAllTags(nil)

				// Find the rock tag (appears in libraries 1 and 2)
				var rockTag *model.Tag
				for _, tag := range tags {
					if tag.TagValue == tagValueRock {
						rockTag = &tag
						break
					}
				}
				Expect(rockTag).ToNot(BeNil())

				// Should have stats from all libraries where rock appears
				// Library 1: 5 albums, 20 songs
				// Library 2: 1 album, 4 songs
				// Total: 6 albums, 24 songs
				Expect(rockTag.AlbumCount).To(Equal(6))
				Expect(rockTag.SongCount).To(Equal(24))
			})

			It("should allow headless processes to apply explicit library_id filters", func() {
				tags := readAllTags(nil, rest.QueryOptions{
					Filters: map[string]interface{}{"library_id": libraryID3},
				})
				// Should see only jazz from library 3
				Expect(tags).To(HaveLen(1))
				Expect(tags[0].TagValue).To(Equal(tagValueJazz))
			})
		})

		Context("Admin User with Explicit Library Filtering", func() {
			It("should see all tags when no filter is applied", func() {
				tags := readAllTags(&adminUser)
				Expect(tags).To(HaveLen(3))
			})

			It("should respect explicit library_id filters", func() {
				tags := readAllTags(&adminUser, rest.QueryOptions{
					Filters: map[string]interface{}{"library_id": libraryID3},
				})
				// Should see only jazz from library 3
				Expect(tags).To(HaveLen(1))
				Expect(tags[0].TagValue).To(Equal(tagValueJazz))
			})

			It("should filter by library 2 correctly", func() {
				tags := readAllTags(&adminUser, rest.QueryOptions{
					Filters: map[string]interface{}{"library_id": libraryID2},
				})
				// Should see pop and rock from library 2
				Expect(tags).To(HaveLen(2))
				expectTagValues(tags, []string{tagValuePop, tagValueRock})
			})
		})
	})
})
