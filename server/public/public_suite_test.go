package public

import (
	"context"
	"testing"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPublicEndpoints(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Public Endpoints Suite")
}

var _ = persistencetest.ResetBeforeEachSpec()

var defaultShareOwner = model.User{ID: "owner1", UserName: "owner1", IsAdmin: true}

// putShare stores the share, owned by defaultShareOwner (created here) unless it names an owner.
func putShare(ctx context.Context, ds model.DataStore, s model.Share) {
	GinkgoHelper()
	if s.UserID == "" {
		owner := defaultShareOwner
		Expect(ds.User().Put(ctx, &owner)).To(Succeed())
		s.UserID = owner.ID
	}
	_, err := ds.Share().Save(ctx, &s)
	Expect(err).ToNot(HaveOccurred())
}

func putMediaFiles(ctx context.Context, ds model.DataStore, mfs ...model.MediaFile) {
	GinkgoHelper()
	for _, mf := range mfs {
		if mf.LibraryID == 0 {
			mf.LibraryID = model.DefaultLibraryID
		}
		if mf.Path == "" {
			mf.Path = mf.ID + ".mp3"
		}
		Expect(ds.MediaFile().Put(ctx, &mf)).To(Succeed())
	}
}

type shareGetErrorDS struct {
	model.DataStore
	err error
}

func withShareGetError(ds model.DataStore, err error) model.DataStore {
	return persistencetest.Overlay(ds, func(ds model.DataStore) model.DataStore {
		return &shareGetErrorDS{DataStore: ds, err: err}
	})
}

func (d *shareGetErrorDS) Share() model.ShareRepository {
	return &shareGetErrorRepo{ShareRepository: d.DataStore.Share(), err: d.err}
}

type shareGetErrorRepo struct {
	model.ShareRepository
	err error
}

func (r *shareGetErrorRepo) Get(context.Context, string) (*model.Share, error) {
	return nil, r.err
}
