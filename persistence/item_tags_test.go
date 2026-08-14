package persistence

import (
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pocketbase/dbx"
)

var _ = Describe("item genre tag indexes", func() {
	var conn *dbx.DB
	var mr model.MediaFileRepository
	var ar model.AlbumRepository
	var rock, jazz model.Tag

	BeforeEach(func() {
		ctx := request.WithUser(GinkgoT().Context(), model.User{ID: "userid"})
		conn = GetDBXBuilder()
		mr = NewMediaFileRepository(ctx, conn)
		ar = NewAlbumRepository(ctx, conn)
		// Test-only genre values, so they can't collide with the golden fixtures.
		rock = model.NewTag(model.TagGenre, "GenreIdxRock")
		jazz = model.NewTag(model.TagGenre, "GenreIdxJazz")
		// The join tables FK to tag(id); the scanner adds tags before saving items.
		Expect(NewTagRepository(ctx, conn).Add(1, rock, jazz)).To(Succeed())
		// The suite shares one golden DB with no per-test restore, so undo the rows we add
		// (media_file/album deletes cascade to the *_tags join rows; tag deletes cascade too).
		DeferCleanup(func() {
			_, _ = conn.NewQuery("DELETE FROM media_file WHERE id LIKE 'mf-%'").Execute()
			_, _ = conn.NewQuery("DELETE FROM album WHERE id LIKE 'al-%'").Execute()
			_, _ = conn.NewQuery("DELETE FROM tag WHERE id={:r} OR id={:j}").
				Bind(dbx.Params{"r": rock.ID, "j": jazz.ID}).Execute()
		})
	})

	tagIDsFor := func(table, col, id string) []string {
		var rows []struct {
			TagID string `db:"tag_id"`
		}
		err := conn.NewQuery("SELECT tag_id FROM " + table + " WHERE " + col + "={:id}").
			Bind(dbx.Params{"id": id}).All(&rows)
		Expect(err).ToNot(HaveOccurred())
		ids := make([]string, len(rows))
		for i, r := range rows {
			ids[i] = r.TagID
		}
		return ids
	}

	Describe("media files", func() {
		It("writes a media_file_tags row for each genre when the track is saved", func() {
			mf := model.MediaFile{ID: "mf-g1", LibraryID: 1, Path: "/m/g1.mp3", Title: "G1",
				Tags: model.Tags{model.TagGenre: []string{rock.TagValue, jazz.TagValue}}}
			Expect(mr.Put(&mf)).To(Succeed())
			Expect(tagIDsFor("media_file_tags", "media_file_id", "mf-g1")).To(ConsistOf(rock.ID, jazz.ID))
		})

		It("replaces the rows when the genres change", func() {
			mf := model.MediaFile{ID: "mf-g2", LibraryID: 1, Path: "/m/g2.mp3", Title: "G2",
				Tags: model.Tags{model.TagGenre: []string{rock.TagValue}}}
			Expect(mr.Put(&mf)).To(Succeed())
			mf.Tags = model.Tags{model.TagGenre: []string{jazz.TagValue}}
			Expect(mr.Put(&mf)).To(Succeed())
			Expect(tagIDsFor("media_file_tags", "media_file_id", "mf-g2")).To(ConsistOf(jazz.ID))
		})

		It("clears the rows when all genres are removed", func() {
			mf := model.MediaFile{ID: "mf-g3", LibraryID: 1, Path: "/m/g3.mp3", Title: "G3",
				Tags: model.Tags{model.TagGenre: []string{rock.TagValue}}}
			Expect(mr.Put(&mf)).To(Succeed())
			mf.Tags = model.Tags{}
			Expect(mr.Put(&mf)).To(Succeed())
			Expect(tagIDsFor("media_file_tags", "media_file_id", "mf-g3")).To(BeEmpty())
		})
	})

	Describe("albums", func() {
		It("writes an album_tags row for each genre when the album is saved", func() {
			al := model.Album{ID: "al-g1", LibraryID: 1, Name: "AG1",
				Tags: model.Tags{model.TagGenre: []string{rock.TagValue, jazz.TagValue}}}
			Expect(ar.Put(&al)).To(Succeed())
			Expect(tagIDsFor("album_tags", "album_id", "al-g1")).To(ConsistOf(rock.ID, jazz.ID))
		})
	})

	// The native (REST) API filters by genre_id; it must resolve through the join table too.
	Describe("native genre_id filter", func() {
		It("filters media files by genre_id", func() {
			mf := model.MediaFile{ID: "mf-nat1", LibraryID: 1, Path: "/m/nat1.mp3", Title: "Nat1",
				Tags: model.Tags{model.TagGenre: []string{rock.TagValue}}}
			Expect(mr.Put(&mf)).To(Succeed())
			res, err := mr.(model.ResourceRepository).ReadAll(rest.QueryOptions{Filters: map[string]any{"genre_id": rock.ID}})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.(model.MediaFiles)).To(ContainElement(HaveField("ID", "mf-nat1")))
		})

		It("filters albums by genre_id", func() {
			al := model.Album{ID: "al-nat1", LibraryID: 1, Name: "ANat1",
				Tags: model.Tags{model.TagGenre: []string{rock.TagValue}}}
			Expect(ar.Put(&al)).To(Succeed())
			res, err := ar.(model.ResourceRepository).ReadAll(rest.QueryOptions{Filters: map[string]any{"genre_id": rock.ID}})
			Expect(err).ToNot(HaveOccurred())
			Expect(res.(model.Albums)).To(ContainElement(HaveField("ID", "al-nat1")))
		})
	})
})
