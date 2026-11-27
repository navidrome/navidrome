package playlists_test

import (
	"context"
	"testing"
	"time"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlaylists(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Playlists Suite")
}

var _ = configtest.RestoreAfterEachSpec()
var _ = persistencetest.ResetBeforeEachSpec()

var adminUser = model.User{ID: "admin-1", UserName: "admin-1", IsAdmin: true}

func asAdmin(ctx context.Context) context.Context {
	return request.WithUser(ctx, adminUser)
}

// putUsers creates regular users named after their IDs, plus adminUser.
func putUsers(ctx context.Context, ds model.DataStore, ids ...string) {
	GinkgoHelper()
	admin := adminUser
	Expect(ds.User().Put(ctx, &admin)).To(Succeed())
	for _, id := range ids {
		Expect(ds.User().Put(ctx, &model.User{ID: id, UserName: id})).To(Succeed())
	}
}

// putTracks stores one media file per path in the library and returns their IDs, in order.
func putTracks(ctx context.Context, ds model.DataStore, libID int, paths ...string) []string {
	GinkgoHelper()
	ids := make([]string, 0, len(paths))
	for _, path := range paths {
		mf := &model.MediaFile{LibraryID: libID, Path: path, Title: path}
		Expect(ds.MediaFile().Put(ctx, mf)).To(Succeed())
		ids = append(ids, mf.ID)
	}
	return ids
}

// putLibraryRoot stores the root folder of the library, as the scanner would.
func putLibraryRoot(ctx context.Context, ds model.DataStore, libID int) {
	GinkgoHelper()
	lib, err := ds.Library().Get(ctx, libID)
	Expect(err).ToNot(HaveOccurred())
	Expect(ds.Folder().Put(ctx, model.NewFolder(*lib, "."))).To(Succeed())
}

func putPlaylists(ctx context.Context, ds model.DataStore, playlists ...*model.Playlist) {
	GinkgoHelper()
	for _, pls := range playlists {
		Expect(ds.Playlist().Put(ctx, pls)).To(Succeed())
	}
}

func storedPlaylist(ctx context.Context, ds model.DataStore, id string) *model.Playlist {
	GinkgoHelper()
	pls, err := ds.Playlist().GetWithTracks(asAdmin(ctx), id, false, false)
	Expect(err).ToNot(HaveOccurred())
	return pls
}

func trackIDs(pls *model.Playlist) []string {
	ids := make([]string, len(pls.Tracks))
	for i, t := range pls.Tracks {
		ids[i] = t.MediaFileID
	}
	return ids
}

// backdate sets a playlist's updated_at, which every Put overwrites, so a spec can tell whether it was written again.
func backdate(id string, t time.Time) {
	GinkgoHelper()
	_, err := db.Db().ExecContext(GinkgoT().Context(), "UPDATE playlist SET updated_at = ? WHERE id = ?", t, id)
	Expect(err).ToNot(HaveOccurred())
}
