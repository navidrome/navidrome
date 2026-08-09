package persistence

import (
	"context"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("MediaFileTagRepository", func() {
	var repo model.MediaFileTagRepository

	BeforeEach(func() {
		ctx := log.NewContext(context.TODO())
		ctx = request.WithUser(ctx, adminUser)
		repo = NewMediaFileTagRepository(ctx, GetDBXBuilder())
	})

	AfterEach(func() {
		for _, tagName := range []string{"genre:rock", "genre:jazz", "my-favorite"} {
			_ = repo.UntagSong(songDayInALife.ID, tagName)
			_ = repo.UntagSong(songComeTogether.ID, tagName)
		}
	})

	Describe("TagSong", func() {
		It("records the given source", func() {
			Expect(repo.TagSong(songDayInALife.ID, "genre:rock", model.MediaFileTagSourceAI)).To(Succeed())

			aiTags, err := repo.TagsForSong(songDayInALife.ID, model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())
			Expect(aiTags).To(ContainElement("genre:rock"))

			userTags, err := repo.TagsForSong(songDayInALife.ID, model.MediaFileTagSourceUser)
			Expect(err).To(BeNil())
			Expect(userTags).ToNot(ContainElement("genre:rock"))
		})

		It("is idempotent - tagging the same song/tag twice does not error or duplicate", func() {
			Expect(repo.TagSong(songDayInALife.ID, "genre:rock", model.MediaFileTagSourceAI)).To(Succeed())
			Expect(repo.TagSong(songDayInALife.ID, "genre:rock", model.MediaFileTagSourceAI)).To(Succeed())

			tags, err := repo.TagsForSong(songDayInALife.ID, "")
			Expect(err).To(BeNil())
			count := 0
			for _, t := range tags {
				if t == "genre:rock" {
					count++
				}
			}
			Expect(count).To(Equal(1))
		})
	})

	Describe("TagsForSong", func() {
		BeforeEach(func() {
			Expect(repo.TagSong(songDayInALife.ID, "genre:rock", model.MediaFileTagSourceAI)).To(Succeed())
			Expect(repo.TagSong(songDayInALife.ID, "my-favorite", model.MediaFileTagSourceUser)).To(Succeed())
		})

		It("returns tags of any source when source is empty", func() {
			tags, err := repo.TagsForSong(songDayInALife.ID, "")
			Expect(err).To(BeNil())
			Expect(tags).To(ContainElements("genre:rock", "my-favorite"))
		})

		It("filters to AI-sourced tags only", func() {
			tags, err := repo.TagsForSong(songDayInALife.ID, model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())
			Expect(tags).To(ContainElement("genre:rock"))
			Expect(tags).ToNot(ContainElement("my-favorite"))
		})

		It("filters to user-sourced tags only", func() {
			tags, err := repo.TagsForSong(songDayInALife.ID, model.MediaFileTagSourceUser)
			Expect(err).To(BeNil())
			Expect(tags).To(ContainElement("my-favorite"))
			Expect(tags).ToNot(ContainElement("genre:rock"))
		})
	})

	Describe("AllTagNames", func() {
		BeforeEach(func() {
			Expect(repo.TagSong(songDayInALife.ID, "genre:rock", model.MediaFileTagSourceAI)).To(Succeed())
			Expect(repo.TagSong(songComeTogether.ID, "my-favorite", model.MediaFileTagSourceUser)).To(Succeed())
		})

		It("filters distinct tag names by source", func() {
			aiNames, err := repo.AllTagNames(model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())
			Expect(aiNames).To(ContainElement("genre:rock"))
			Expect(aiNames).ToNot(ContainElement("my-favorite"))

			userNames, err := repo.AllTagNames(model.MediaFileTagSourceUser)
			Expect(err).To(BeNil())
			Expect(userNames).To(ContainElement("my-favorite"))
			Expect(userNames).ToNot(ContainElement("genre:rock"))
		})
	})

	Describe("SongIDsForTag", func() {
		BeforeEach(func() {
			Expect(repo.TagSong(songDayInALife.ID, "genre:jazz", model.MediaFileTagSourceAI)).To(Succeed())
			Expect(repo.TagSong(songComeTogether.ID, "genre:jazz", model.MediaFileTagSourceUser)).To(Succeed())
		})

		It("returns every song with the tag when source is empty", func() {
			ids, err := repo.SongIDsForTag("genre:jazz", "")
			Expect(err).To(BeNil())
			Expect(ids).To(ContainElements(songDayInALife.ID, songComeTogether.ID))
		})

		It("filters to songs tagged by the given source only", func() {
			ids, err := repo.SongIDsForTag("genre:jazz", model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())
			Expect(ids).To(ContainElement(songDayInALife.ID))
			Expect(ids).ToNot(ContainElement(songComeTogether.ID))
		})
	})

	Describe("TagCounts", func() {
		BeforeEach(func() {
			Expect(repo.TagSong(songDayInALife.ID, "genre:rock", model.MediaFileTagSourceAI)).To(Succeed())
			Expect(repo.TagSong(songComeTogether.ID, "genre:rock", model.MediaFileTagSourceAI)).To(Succeed())
			Expect(repo.TagSong(songDayInALife.ID, "my-favorite", model.MediaFileTagSourceUser)).To(Succeed())
		})

		It("counts distinct songs per tag name, filtered by source", func() {
			counts, err := repo.TagCounts(model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())

			byName := map[string]int{}
			for _, c := range counts {
				byName[c.TagName] = c.Count
			}
			Expect(byName["genre:rock"]).To(Equal(2))
			Expect(byName).ToNot(HaveKey("my-favorite"))
		})

		It("returns counts for the other source independently", func() {
			counts, err := repo.TagCounts(model.MediaFileTagSourceUser)
			Expect(err).To(BeNil())

			byName := map[string]int{}
			for _, c := range counts {
				byName[c.TagName] = c.Count
			}
			Expect(byName["my-favorite"]).To(Equal(1))
			Expect(byName).ToNot(HaveKey("genre:rock"))
		})

		It("counts across any source when source is empty", func() {
			counts, err := repo.TagCounts("")
			Expect(err).To(BeNil())

			byName := map[string]int{}
			for _, c := range counts {
				byName[c.TagName] = c.Count
			}
			Expect(byName["genre:rock"]).To(Equal(2))
			Expect(byName["my-favorite"]).To(Equal(1))
		})
	})

	Describe("UntagSong", func() {
		It("removes the tag regardless of source", func() {
			Expect(repo.TagSong(songDayInALife.ID, "my-favorite", model.MediaFileTagSourceUser)).To(Succeed())
			Expect(repo.UntagSong(songDayInALife.ID, "my-favorite")).To(Succeed())

			tags, err := repo.TagsForSong(songDayInALife.ID, "")
			Expect(err).To(BeNil())
			Expect(tags).ToNot(ContainElement("my-favorite"))
		})
	})

	Describe("Option B: shared AI tags", func() {
		// AI-sourced tags are shared, admin-written data: any admin's rows count as "the" shared
		// AI tags for the aggregate/browse reads, not just whoever's currently logged in. My tags
		// stay private per caller, unaffected - regression coverage for that split.
		var secondAdmin model.User

		BeforeEach(func() {
			secondAdmin = model.User{ID: "second-admin-id", UserName: "second-admin", IsAdmin: true}
			secondAdmin.NewPassword = "password"
			userRepo := NewUserRepository(log.NewContext(context.TODO()), GetDBXBuilder())
			Expect(userRepo.Put(&secondAdmin)).To(Succeed())

			// adminUser (the package fixture, "userid") writes the AI tag.
			Expect(repo.TagSong(songDayInALife.ID, "genre:rock", model.MediaFileTagSourceAI)).To(Succeed())
		})

		AfterEach(func() {
			_, _ = GetDBXBuilder().NewQuery("DELETE FROM user WHERE id = 'second-admin-id'").Execute()
		})

		It("is visible to a different admin, not just the admin who wrote it", func() {
			otherAdminCtx := request.WithUser(log.NewContext(context.TODO()), secondAdmin)
			otherAdminRepo := NewMediaFileTagRepository(otherAdminCtx, GetDBXBuilder())

			names, err := otherAdminRepo.AllTagNames(model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())
			Expect(names).To(ContainElement("genre:rock"))

			ids, err := otherAdminRepo.SongIDsForTag("genre:rock", model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())
			Expect(ids).To(ContainElement(songDayInALife.ID))

			counts, err := otherAdminRepo.TagCounts(model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())
			found := false
			for _, c := range counts {
				if c.TagName == "genre:rock" {
					found = true
				}
			}
			Expect(found).To(BeTrue())
		})

		It("is visible to a non-admin user with no AI tags of their own (Option B's whole point)", func() {
			nonAdminUser := model.User{ID: "opt-b-nonadmin-id", UserName: "opt-b-nonadmin"}
			nonAdminUser.NewPassword = "password"
			userRepo := NewUserRepository(log.NewContext(context.TODO()), GetDBXBuilder())
			Expect(userRepo.Put(&nonAdminUser)).To(Succeed())
			defer func() {
				_, _ = GetDBXBuilder().NewQuery("DELETE FROM user WHERE id = 'opt-b-nonadmin-id'").Execute()
			}()

			nonAdminCtx := request.WithUser(log.NewContext(context.TODO()), nonAdminUser)
			nonAdminRepo := NewMediaFileTagRepository(nonAdminCtx, GetDBXBuilder())

			names, err := nonAdminRepo.AllTagNames(model.MediaFileTagSourceAI)
			Expect(err).To(BeNil())
			Expect(names).To(ContainElement("genre:rock"), "a permitted non-admin must see the same AI tags an admin does, not their own (empty) set")
		})

		It("does not extend the same sharing to My Tags", func() {
			Expect(repo.TagSong(songDayInALife.ID, "admins-personal-note", model.MediaFileTagSourceUser)).To(Succeed())
			defer func() { _ = repo.UntagSong(songDayInALife.ID, "admins-personal-note") }()

			otherAdminCtx := request.WithUser(log.NewContext(context.TODO()), secondAdmin)
			otherAdminRepo := NewMediaFileTagRepository(otherAdminCtx, GetDBXBuilder())

			names, err := otherAdminRepo.AllTagNames(model.MediaFileTagSourceUser)
			Expect(err).To(BeNil())
			Expect(names).ToNot(ContainElement("admins-personal-note"), "My Tags must stay private per writer, even between two admins")
		})
	})
})
