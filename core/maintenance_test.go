package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("Maintenance", func() {
	var ds model.DataStore
	var service Maintenance
	var ctx context.Context

	putAlbums := func(albums ...model.Album) {
		GinkgoHelper()
		for _, al := range albums {
			al.LibraryID = model.DefaultLibraryID
			Expect(ds.Album().Put(ctx, &al)).To(Succeed())
		}
	}

	putFiles := func(mfs ...model.MediaFile) {
		GinkgoHelper()
		for _, mf := range mfs {
			mf.LibraryID = model.DefaultLibraryID
			if mf.Path == "" {
				mf.Path = mf.ID + ".mp3"
			}
			Expect(ds.MediaFile().Put(ctx, &mf)).To(Succeed())
		}
	}

	fileExists := func(id string) bool {
		GinkgoHelper()
		exists, err := ds.MediaFile().Exists(ctx, id)
		Expect(err).ToNot(HaveOccurred())
		return exists
	}

	albumExists := func(id string) bool {
		GinkgoHelper()
		exists, err := ds.Album().Exists(ctx, id)
		Expect(err).ToNot(HaveOccurred())
		return exists
	}

	getAlbum := func(id string) *model.Album {
		GinkgoHelper()
		al, err := ds.Album().Get(ctx, id)
		Expect(err).ToNot(HaveOccurred())
		return al
	}

	starred := func(id string) bool {
		GinkgoHelper()
		return getAlbum(id).Starred
	}

	putArtist := func(id string) model.Participant {
		GinkgoHelper()
		Expect(ds.Artist().Put(ctx, &model.Artist{ID: id, Name: "Artist " + id})).To(Succeed())
		Expect(ds.Library().AddArtist(ctx, model.DefaultLibraryID, id)).To(Succeed())
		return model.Participant{ID: id, Name: "Artist " + id}
	}

	artistSongCount := func(id string) int {
		GinkgoHelper()
		a, err := ds.Artist().Get(ctx, id)
		Expect(err).ToNot(HaveOccurred())
		return a.SongCount
	}

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		ds = persistence.New(db.Db())
		admin := model.User{ID: "user1", UserName: "user1", IsAdmin: true}
		Expect(ds.User().Put(ctx, &admin)).To(Succeed())
		ctx = request.WithUser(ctx, admin)
		service = NewMaintenance(ds)
		DeferCleanup(func() { service.(*maintenanceService).wait() })
	})

	Describe("DeleteMissingFiles", func() {
		Context("with specific IDs", func() {
			It("deletes specific missing files and runs GC", func() {
				putAlbums(model.Album{ID: "album1"}, model.Album{ID: "album2"}, model.Album{ID: "album3"})
				putFiles(
					model.MediaFile{ID: "mf1", AlbumID: "album1", Missing: true},
					model.MediaFile{ID: "mf2", AlbumID: "album2", Missing: true},
					model.MediaFile{ID: "mf3", AlbumID: "album3", Missing: true},
				)

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf2"})

				Expect(err).ToNot(HaveOccurred())
				Expect(fileExists("mf1")).To(BeFalse())
				Expect(fileExists("mf2")).To(BeFalse())
				Expect(fileExists("mf3")).To(BeTrue())
				Expect(albumExists("album1")).To(BeFalse(), "GC should purge albums left empty by the deletion")
				Expect(albumExists("album2")).To(BeFalse(), "GC should purge albums left empty by the deletion")
				Expect(albumExists("album3")).To(BeTrue())
			})

			It("triggers artist stats refresh and album refresh after deletion", func() {
				artist := model.Participants{model.RoleArtist: {putArtist("ar1")}}
				putAlbums(model.Album{ID: "album1", Name: "Test Album", SongCount: 5, Participants: artist})
				putFiles(
					model.MediaFile{ID: "mf1", AlbumID: "album1", Album: "Test Album", Missing: true, Participants: artist},
					model.MediaFile{ID: "mf2", AlbumID: "album1", Album: "Test Album", Size: 1000, Duration: 180, Participants: artist},
					model.MediaFile{ID: "mf3", AlbumID: "album1", Album: "Test Album", Size: 2000, Duration: 200, Participants: artist},
				)
				Expect(artistSongCount("ar1")).To(BeZero())

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).ToNot(HaveOccurred())
				service.(*maintenanceService).wait()

				Expect(artistSongCount("ar1")).To(Equal(2), "Artist stats should be refreshed")
				al := getAlbum("album1")
				Expect(al.SongCount).To(Equal(2))
				Expect(al.Size).To(Equal(int64(3000)))
				Expect(al.Duration).To(BeNumerically("==", 380))
			})

			It("returns error if deletion fails", func() {
				putFiles(model.MediaFile{ID: "mf1", Missing: true})
				service = NewMaintenance(withMediaFileErrors(ds, mediaFileErrors{deleteMissing: errors.New("delete failed")}))

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("delete failed"))
				Expect(fileExists("mf1")).To(BeTrue())
			})

			It("continues even if album tracking fails", func() {
				putFiles(model.MediaFile{ID: "mf1", Missing: true})
				service = NewMaintenance(withMediaFileErrors(ds, mediaFileErrors{getAll: errors.New("get failed")}))

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).ToNot(HaveOccurred())
				Expect(fileExists("mf1")).To(BeFalse())
			})

			It("returns error if GC fails", func() {
				putFiles(model.MediaFile{ID: "mf1", AlbumID: "album1", Missing: true})
				service = NewMaintenance(withGCError(ds, errors.New("gc failed")))

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("gc failed"))
			})
		})

		Context("album ID extraction", func() {
			It("extracts unique album IDs from missing files", func() {
				putFiles(
					model.MediaFile{ID: "mf1", AlbumID: "album1", Missing: true},
					model.MediaFile{ID: "mf2", AlbumID: "album1", Missing: true},
					model.MediaFile{ID: "mf3", AlbumID: "album2", Missing: true},
					model.MediaFile{ID: "mf4", AlbumID: "album3", Missing: true},
				)

				ids, err := service.(*maintenanceService).getAffectedAlbumIDs(ctx, []string{"mf1", "mf2", "mf3"})

				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(ConsistOf("album1", "album2"))
			})

			It("skips files without album IDs", func() {
				putFiles(
					model.MediaFile{ID: "mf1", AlbumID: "", Missing: true},
					model.MediaFile{ID: "mf2", AlbumID: "album1", Missing: true},
				)

				ids, err := service.(*maintenanceService).getAffectedAlbumIDs(ctx, []string{"mf1", "mf2"})

				Expect(err).ToNot(HaveOccurred())
				Expect(ids).To(ConsistOf("album1"))
			})
		})
	})

	Describe("DeleteAllMissingFiles", func() {
		It("deletes all missing files and runs GC", func() {
			putAlbums(model.Album{ID: "album1"}, model.Album{ID: "album2"}, model.Album{ID: "album3"}, model.Album{ID: "album4"})
			putFiles(
				model.MediaFile{ID: "mf1", AlbumID: "album1", Missing: true},
				model.MediaFile{ID: "mf2", AlbumID: "album2", Missing: true},
				model.MediaFile{ID: "mf3", AlbumID: "album3", Missing: true},
				model.MediaFile{ID: "mf4", AlbumID: "album4"},
			)

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).ToNot(HaveOccurred())
			for _, id := range []string{"mf1", "mf2", "mf3"} {
				Expect(fileExists(id)).To(BeFalse())
			}
			Expect(fileExists("mf4")).To(BeTrue())
			Expect(albumExists("album1")).To(BeFalse(), "GC should purge albums left empty by the deletion")
			Expect(albumExists("album4")).To(BeTrue())
		})

		It("returns error if deletion fails", func() {
			putFiles(model.MediaFile{ID: "mf1", Missing: true})
			service = NewMaintenance(withMediaFileErrors(ds, mediaFileErrors{deleteAllMissing: errors.New("delete failed")}))

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).To(MatchError(ContainSubstring("delete failed")))
			Expect(fileExists("mf1")).To(BeTrue())
		})

		It("handles empty result gracefully", func() {
			putFiles(model.MediaFile{ID: "mf1"})

			err := service.DeleteAllMissingFiles(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(fileExists("mf1")).To(BeTrue())
		})
	})

	Describe("Album refresh logic", func() {
		Context("when album has no tracks after deletion", func() {
			It("skips the album without updating it", func() {
				putAlbums(model.Album{ID: "album1", Name: "Empty Album", SongCount: 1})
				lastUpdate := time.Now().Add(-time.Hour).Truncate(time.Second)
				_, err := db.Db().Exec("UPDATE album SET updated_at = ? WHERE id = ?", lastUpdate, "album1")
				Expect(err).ToNot(HaveOccurred())

				Expect(service.(*maintenanceService).refreshAlbums(ctx, []string{"album1"})).To(Succeed())

				al := getAlbum("album1")
				Expect(al.SongCount).To(Equal(1), "Album with no tracks should not be updated")
				Expect(al.UpdatedAt).To(BeTemporally("==", lastUpdate))
			})
		})

		Context("when Put fails for one album", func() {
			It("continues processing other albums", func() {
				putAlbums(model.Album{ID: "album1", Name: "Album 1"}, model.Album{ID: "album2", Name: "Album 2"})
				putFiles(
					model.MediaFile{ID: "mf1", AlbumID: "album1", Album: "Album 1", Missing: true},
					model.MediaFile{ID: "mf2", AlbumID: "album1", Album: "Album 1", Size: 1000, Duration: 180},
					model.MediaFile{ID: "mf3", AlbumID: "album2", Album: "Album 2", Missing: true},
					model.MediaFile{ID: "mf4", AlbumID: "album2", Album: "Album 2", Size: 2000, Duration: 200},
				)
				service = NewMaintenance(withAlbumPutFailures(ds, 1))

				err := service.DeleteMissingFiles(ctx, []string{"mf1", "mf3"})

				Expect(err).ToNot(HaveOccurred())
				service.(*maintenanceService).wait()

				refreshed := 0
				for _, id := range []string{"album1", "album2"} {
					if getAlbum(id).SongCount == 1 {
						refreshed++
					}
				}
				Expect(refreshed).To(Equal(1), "the album whose Put did not fail should still be refreshed")
			})
		})

		Context("when media file loading fails", func() {
			It("logs warning but continues when tracking affected albums fails", func() {
				hook, cleanup := tests.LogHook()
				defer cleanup()

				putAlbums(model.Album{ID: "album1", Name: "Album 1"})
				putFiles(model.MediaFile{ID: "mf1", AlbumID: "album1", Missing: true})
				service = NewMaintenance(withMediaFileErrors(ds, mediaFileErrors{getAll: errors.New("get failed")}))

				err := service.DeleteMissingFiles(ctx, []string{"mf1"})

				Expect(err).ToNot(HaveOccurred())
				service.(*maintenanceService).wait()
				Expect(fileExists("mf1")).To(BeFalse())

				Expect(hook.AllEntries()).To(ContainElement(SatisfyAll(
					HaveField("Level", logrus.WarnLevel),
					HaveField("Message", "Error tracking affected albums for refresh"),
				)))
			})
		})
	})

	Describe("RemapMissingFile", func() {
		It("relocates the missing file's identity onto the target and runs GC", func() {
			created := time.Now().Add(-30 * 24 * time.Hour).Truncate(time.Second)
			putAlbums(model.Album{ID: "album1"}, model.Album{ID: "orphan"})
			putFiles(
				model.MediaFile{ID: "m1", Path: "old/song.mp3", AlbumID: "album1", CreatedAt: created, Missing: true},
				model.MediaFile{ID: "t1", Path: "new/song.mp3", AlbumID: "album1", Missing: false},
			)

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(Succeed())

			got, err := ds.MediaFile().Get(ctx, "m1")
			Expect(err).ToNot(HaveOccurred())
			Expect(got.Path).To(Equal("new/song.mp3"))
			Expect(got.Missing).To(BeFalse())
			Expect(got.CreatedAt).To(BeTemporally("==", created))
			Expect(fileExists("t1")).To(BeFalse())
			Expect(albumExists("orphan")).To(BeFalse(), "GC should purge empty albums")
		})

		It("moves the target's annotations, bookmarks and playlist entries onto the surviving id", func() {
			putFiles(
				model.MediaFile{ID: "m1", AlbumID: "album1", Missing: true},
				model.MediaFile{ID: "t1", AlbumID: "album1", Missing: false},
			)
			Expect(ds.MediaFile().SetStar(ctx, true, "t1")).To(Succeed())
			Expect(ds.MediaFile().AddBookmark(ctx, "t1", "here", 42)).To(Succeed())
			pls := &model.Playlist{Name: "pls", OwnerID: "user1"}
			Expect(ds.Playlist().Put(ctx, pls)).To(Succeed())
			_, err := ds.Playlist().Tracks(ctx, pls.ID, false).Add(ctx, []string{"t1"})
			Expect(err).ToNot(HaveOccurred())

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(Succeed())

			got, err := ds.MediaFile().Get(ctx, "m1")
			Expect(err).ToNot(HaveOccurred())
			Expect(got.Starred).To(BeTrue())
			bookmarks, err := ds.MediaFile().GetBookmarks(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(bookmarks).To(HaveLen(1))
			Expect(bookmarks[0].Item.ID).To(Equal("m1"))
			Expect(bookmarks[0].Position).To(Equal(int64(42)))
			ids, err := ds.Playlist().Tracks(ctx, pls.ID, false).GetMediaFileIDs(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(ids).To(Equal([]string{"m1"}))
		})

		It("reassigns album annotations when the old album is emptied", func() {
			putAlbums(model.Album{ID: "album1"}, model.Album{ID: "album2"})
			putFiles(
				model.MediaFile{ID: "m1", AlbumID: "album1", Missing: true},
				model.MediaFile{ID: "t1", AlbumID: "album2", Missing: false},
			)
			Expect(ds.Album().SetStar(ctx, true, "album1")).To(Succeed())

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(Succeed())

			Expect(starred("album2")).To(BeTrue())
		})

		It("does not reassign album annotations when the old album is not emptied", func() {
			putAlbums(model.Album{ID: "album1"}, model.Album{ID: "album2"})
			putFiles(
				model.MediaFile{ID: "m1", AlbumID: "album1", Missing: true},
				model.MediaFile{ID: "m2", AlbumID: "album1", Missing: false},
				model.MediaFile{ID: "t1", AlbumID: "album2", Missing: false},
			)
			Expect(ds.Album().SetStar(ctx, true, "album1")).To(Succeed())

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(Succeed())

			Expect(starred("album1")).To(BeTrue())
			Expect(starred("album2")).To(BeFalse())
		})

		It("does not reassign annotations when the album is unchanged", func() {
			putFiles(
				model.MediaFile{ID: "m1", AlbumID: "album1", Missing: true},
				model.MediaFile{ID: "t1", AlbumID: "album1", Missing: false},
			)
			spy := &reassignSpy{}
			service = NewMaintenance(withReassignSpy(ds, spy))

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(Succeed())

			Expect(spy.calls()).To(BeEmpty())
		})

		It("returns ErrNotFound when the missing file does not exist", func() {
			putFiles(model.MediaFile{ID: "t1", Missing: false})

			Expect(service.RemapMissingFile(ctx, "nope", "t1")).To(MatchError(model.ErrNotFound))
		})

		It("refuses to remap from a file that is not missing", func() {
			putFiles(
				model.MediaFile{ID: "m1", Missing: false},
				model.MediaFile{ID: "t1", Missing: false},
			)

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(MatchError(ErrNotMissing))
		})

		It("refuses to remap a file onto itself", func() {
			putFiles(model.MediaFile{ID: "m1", Missing: true})

			Expect(service.RemapMissingFile(ctx, "m1", "m1")).To(MatchError(ErrSameFile))
		})

		It("refuses to remap onto a target that is itself missing", func() {
			putFiles(
				model.MediaFile{ID: "m1", Missing: true},
				model.MediaFile{ID: "t1", Missing: true},
			)

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(MatchError(ErrTargetMissing))
		})

		It("refreshes artist and album stats right after the remap", func() {
			artist := model.Participants{model.RoleArtist: {putArtist("ar1")}}
			putAlbums(
				model.Album{ID: "album1", Name: "Old Album", SongCount: 2, Size: 1100, Duration: 110},
				model.Album{ID: "album2", Name: "New Album", SongCount: 1, Size: 2000, Duration: 200, Participants: artist},
			)
			putFiles(
				model.MediaFile{ID: "m1", Path: "old/1.mp3", Album: "Old Album", AlbumID: "album1", Missing: true, Size: 100, Duration: 10},
				model.MediaFile{ID: "k1", Path: "old/2.mp3", Album: "Old Album", AlbumID: "album1", Missing: false, Size: 1000, Duration: 100},
				model.MediaFile{ID: "t1", Path: "new/1.mp3", Album: "New Album", AlbumID: "album2", Missing: false, Size: 2000, Duration: 200, Participants: artist},
			)
			Expect(artistSongCount("ar1")).To(BeZero())

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(Succeed())

			Expect(artistSongCount("ar1")).To(Equal(1), "Artist stats should be refreshed")

			oldAlbum := getAlbum("album1")
			Expect(oldAlbum.SongCount).To(Equal(1))
			Expect(oldAlbum.Size).To(Equal(int64(1000)))
			Expect(oldAlbum.Duration).To(BeNumerically("==", 100))

			newAlbum := getAlbum("album2")
			Expect(newAlbum.SongCount).To(Equal(1))
			Expect(newAlbum.Size).To(Equal(int64(2000)))
			Expect(newAlbum.Duration).To(BeNumerically("==", 200))
		})

		It("returns an error if GC fails", func() {
			putFiles(
				model.MediaFile{ID: "m1", AlbumID: "album1", Missing: true},
				model.MediaFile{ID: "t1", AlbumID: "album1", Missing: false},
			)
			service = NewMaintenance(withGCError(ds, errors.New("gc failed")))

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(MatchError(ContainSubstring("gc failed")))
		})

		It("preserves the target's participants on the remapped track", func() {
			Expect(ds.Artist().Put(ctx, &model.Artist{ID: "a1", Name: "Artist", OrderArtistName: "artist", MbzArtistID: "mbz-artist"})).To(Succeed())
			participant := model.Participant{ID: "a1", Name: "Artist", OrderArtistName: "artist", MbzArtistID: "mbz-artist"}
			stored := model.Participant{ID: "a1", Name: "Artist"}
			putFiles(
				model.MediaFile{ID: "m1", AlbumID: "album1", Missing: true},
				model.MediaFile{ID: "t1", AlbumID: "album2", Missing: false, Participants: model.Participants{
					model.RoleArtist: model.ParticipantList{participant},
				}},
			)

			Expect(service.RemapMissingFile(ctx, "m1", "t1")).To(Succeed())

			got, err := ds.MediaFile().GetWithParticipants(ctx, "m1")
			Expect(err).ToNot(HaveOccurred())
			Expect(got.Participants).To(HaveKeyWithValue(model.RoleArtist, model.ParticipantList{stored}))
		})
	})
})

type mediaFileErrors struct {
	getAll, deleteMissing, deleteAllMissing error
}

func withMediaFileErrors(ds model.DataStore, errs mediaFileErrors) model.DataStore {
	return persistencetest.Overlay(ds, func(ds model.DataStore) model.DataStore {
		return &mediaFileErrorsDS{DataStore: ds, errs: errs}
	})
}

type mediaFileErrorsDS struct {
	model.DataStore
	errs mediaFileErrors
}

func (d *mediaFileErrorsDS) MediaFile() model.MediaFileRepository {
	return &mediaFileErrorsRepo{MediaFileRepository: d.DataStore.MediaFile(), errs: d.errs}
}

type mediaFileErrorsRepo struct {
	model.MediaFileRepository
	errs mediaFileErrors
}

func (r *mediaFileErrorsRepo) GetAll(ctx context.Context, options ...model.QueryOptions) (model.MediaFiles, error) {
	if r.errs.getAll != nil {
		return nil, r.errs.getAll
	}
	return r.MediaFileRepository.GetAll(ctx, options...)
}

func (r *mediaFileErrorsRepo) DeleteMissing(ctx context.Context, ids []string) error {
	if r.errs.deleteMissing != nil {
		return r.errs.deleteMissing
	}
	return r.MediaFileRepository.DeleteMissing(ctx, ids)
}

func (r *mediaFileErrorsRepo) DeleteAllMissing(ctx context.Context) (int64, error) {
	if r.errs.deleteAllMissing != nil {
		return 0, r.errs.deleteAllMissing
	}
	return r.MediaFileRepository.DeleteAllMissing(ctx)
}

func withGCError(ds model.DataStore, err error) model.DataStore {
	return persistencetest.Overlay(ds, func(ds model.DataStore) model.DataStore {
		return &gcErrorDS{DataStore: ds, err: err}
	})
}

type gcErrorDS struct {
	model.DataStore
	err error
}

func (d *gcErrorDS) GC(context.Context, ...int) error { return d.err }

// withAlbumPutFailures makes the first n album Puts fail, across every Album() call and transaction.
func withAlbumPutFailures(ds model.DataStore, n int32) model.DataStore {
	remaining := &atomic.Int32{}
	remaining.Store(n)
	return persistencetest.Overlay(ds, func(ds model.DataStore) model.DataStore {
		return &albumPutFailuresDS{DataStore: ds, remaining: remaining}
	})
}

type albumPutFailuresDS struct {
	model.DataStore
	remaining *atomic.Int32
}

func (d *albumPutFailuresDS) Album() model.AlbumRepository {
	return &albumPutFailuresRepo{AlbumRepository: d.DataStore.Album(), remaining: d.remaining}
}

type albumPutFailuresRepo struct {
	model.AlbumRepository
	remaining *atomic.Int32
}

func (r *albumPutFailuresRepo) Put(ctx context.Context, al *model.Album) error {
	if r.remaining.Add(-1) >= 0 {
		return errors.New("put failed")
	}
	return r.AlbumRepository.Put(ctx, al)
}

// reassignSpy records album annotation reassignments, which are no-ops in the DB when both IDs are equal.
type reassignSpy struct {
	mu  sync.Mutex
	got []string
}

func (s *reassignSpy) calls() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.got
}

func withReassignSpy(ds model.DataStore, spy *reassignSpy) model.DataStore {
	return persistencetest.Overlay(ds, func(ds model.DataStore) model.DataStore {
		return &reassignSpyDS{DataStore: ds, spy: spy}
	})
}

type reassignSpyDS struct {
	model.DataStore
	spy *reassignSpy
}

func (d *reassignSpyDS) Album() model.AlbumRepository {
	return &reassignSpyRepo{AlbumRepository: d.DataStore.Album(), spy: d.spy}
}

type reassignSpyRepo struct {
	model.AlbumRepository
	spy *reassignSpy
}

func (r *reassignSpyRepo) ReassignAnnotation(ctx context.Context, prevID, newID string) error {
	r.spy.mu.Lock()
	r.spy.got = append(r.spy.got, prevID+"->"+newID)
	r.spy.mu.Unlock()
	return r.AlbumRepository.ReassignAnnotation(ctx, prevID, newID)
}
