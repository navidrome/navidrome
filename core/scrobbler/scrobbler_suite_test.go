package scrobbler

import (
	"context"
	"testing"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAgents(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scrobbler Test Suite")
}

var _ = configtest.RestoreAfterEachSpec()
var _ = persistencetest.ResetBeforeEachSpec()

// newRealDataStore gives a plain (non-Ginkgo) test its own database. Call it outside the
// synctest bubble, so the connection pool's goroutines do not belong to the bubble.
func newRealDataStore(t *testing.T) model.DataStore {
	t.Helper()
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	return persistence.New(db.Db())
}

// testingT is the part of *testing.T and GinkgoT() that putUsers and putTracks need.
type testingT interface {
	Helper()
	Context() context.Context
	Fatalf(format string, args ...any)
}

func putUsers(t testingT, ds model.DataStore, users ...model.User) {
	t.Helper()
	for _, u := range users {
		if err := ds.User().Put(t.Context(), &u); err != nil {
			t.Fatalf("storing user %s: %v", u.ID, err)
		}
	}
}

// putTracks stores one media file per ID, in library 1.
func putTracks(t testingT, ds model.DataStore, ids ...string) {
	t.Helper()
	ctx := t.Context()
	for _, id := range ids {
		mf := &model.MediaFile{ID: id, LibraryID: model.DefaultLibraryID, Path: id + ".mp3", Title: "Track " + id}
		if err := ds.MediaFile().Put(ctx, mf); err != nil {
			t.Fatalf("storing track %s: %v", id, err)
		}
	}
}
