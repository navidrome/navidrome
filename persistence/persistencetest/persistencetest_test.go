package persistencetest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newAlbum(id, name string) *model.Album {
	return &model.Album{
		ID: id, Name: name, LibraryID: model.DefaultLibraryID,
		Discs: model.Discs{}, Tags: model.Tags{}, Participants: model.Participants{},
	}
}

func countProperties(prefix string) int {
	var n int
	Expect(db.Db().QueryRow("SELECT count(*) FROM property WHERE id LIKE ?", prefix+"%").Scan(&n)).To(Succeed())
	return n
}

var _ = Describe("Reset", func() {
	var ctx context.Context
	var ds model.DataStore

	BeforeEach(func() {
		ctx = request.WithUser(GinkgoT().Context(), model.User{ID: "admin", UserName: "admin", IsAdmin: true})
		ds = persistence.New(db.Db())
	})

	It("restores rows seeded by migrations", func() {
		transcodings, err := ds.Transcoding().CountAll(ctx)
		Expect(err).ToNot(HaveOccurred())
		Expect(transcodings).To(BeNumerically(">", 0))
		_, err = db.Db().Exec("DELETE FROM transcoding")
		Expect(err).ToNot(HaveOccurred())
		_, err = db.Db().Exec("DELETE FROM library")
		Expect(err).ToNot(HaveOccurred())

		persistencetest.Reset()

		lib, err := ds.Library().Get(ctx, model.DefaultLibraryID)
		Expect(err).ToNot(HaveOccurred())
		Expect(lib.Name).To(Equal(model.DefaultLibraryName))
		Expect(ds.Transcoding().CountAll(ctx)).To(Equal(transcodings))
	})

	It("removes rows a spec added", func() {
		Expect(ds.Album().Put(ctx, newAlbum("al-1", "Kept"))).To(Succeed())

		persistencetest.Reset()

		_, err := ds.Album().Get(ctx, "al-1")
		Expect(err).To(MatchError(model.ErrNotFound))
	})

	It("keeps full-text search in sync with the restored rows", func() {
		Expect(ds.Album().Put(ctx, newAlbum("al-1", "Zyxwvu"))).To(Succeed())
		Expect(ds.Album().Search(ctx, "Zyxwvu")).To(HaveLen(1))

		persistencetest.Reset()
		Expect(ds.Album().Search(ctx, "Zyxwvu")).To(BeEmpty())

		Expect(ds.Album().Put(ctx, newAlbum("al-2", "Zyxwvu"))).To(Succeed())
		Expect(ds.Album().Search(ctx, "Zyxwvu")).To(HaveLen(1))
	})

	It("forgets library paths cached before the reset", func() {
		baseline, err := ds.Library().GetPath(ctx, model.DefaultLibraryID)
		Expect(err).ToNot(HaveOccurred())
		persistencetest.SetLibraries(ctx, ds, "/somewhere/else")
		Expect(ds.Library().GetPath(ctx, model.DefaultLibraryID)).To(Equal("/somewhere/else"))

		persistencetest.Reset()

		Expect(ds.Library().GetPath(ctx, model.DefaultLibraryID)).To(Equal(baseline))
	})
})

var _ = Describe("ResetBeforeEachSpec", func() {
	// Without the hook, whichever of these two specs runs second would see the other's row.
	for _, name := range []string{"first", "second"} {
		It("starts the "+name+" spec from the baseline", func() {
			Expect(countProperties("isolated-")).To(BeZero())
			Expect(persistence.New(db.Db()).Property().Put(GinkgoT().Context(), "isolated-"+name, "v")).To(Succeed())
		})
	}

	// Without the reset before each container, whichever container runs second would see the other's row.
	for _, name := range []string{"first", "second"} {
		Describe("in the "+name+" Ordered container", Ordered, func() {
			It("starts from the baseline", func() {
				Expect(countProperties("ordered-")).To(BeZero())
				Expect(persistence.New(db.Db()).Property().Put(GinkgoT().Context(), "ordered-"+name, "v")).To(Succeed())
			})

			It("still sees the row from the previous spec", func() {
				Expect(countProperties("ordered-")).To(Equal(1))
			})
		})
	}
})

func BenchmarkReset(b *testing.B) {
	tests.Init(b, false)
	b.Cleanup(conf.SnapshotConfig())
	persistencetest.Init(b)
	ctx := request.WithUser(b.Context(), model.User{ID: "admin", UserName: "admin", IsAdmin: true})
	ds := persistence.New(db.Db())
	for b.Loop() {
		// A typical spec leaves a user, an album and a few songs behind.
		b.StopTimer()
		if err := ds.User().Put(ctx, &model.User{ID: "u1", UserName: "u1"}); err != nil {
			b.Fatal(err)
		}
		if err := ds.Album().Put(ctx, newAlbum("al-1", "Album")); err != nil {
			b.Fatal(err)
		}
		for i := range 5 {
			mf := &model.MediaFile{ID: fmt.Sprintf("mf-%d", i), Path: fmt.Sprintf("%d.mp3", i), LibraryID: model.DefaultLibraryID, AlbumID: "al-1"}
			if err := ds.MediaFile().Put(ctx, mf); err != nil {
				b.Fatal(err)
			}
		}
		b.StartTimer()
		persistencetest.Reset()
	}
}
