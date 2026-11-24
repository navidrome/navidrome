package subsonic

import (
	"context"
	"io/fs"
	"testing"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	"github.com/navidrome/navidrome/utils/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestSubsonicApi(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Subsonic API Suite")
}

var _ = configtest.RestoreAfterEachSpec()
var _ = persistencetest.ResetBeforeEachSpec()

// newLocalStorage fatals if the default extractor is not registered.
// Register a no-op so storage.For works in sidecar-lyrics tests.
var _ = BeforeSuite(func() {
	local.RegisterExtractor(consts.DefaultScannerExtractor, func(fs.FS, string) local.Extractor {
		return &subsonicNoopExtractor{}
	})
})

type subsonicNoopExtractor struct{}

func (e *subsonicNoopExtractor) Parse(_ ...string) (map[string]metadata.Info, error) { return nil, nil }
func (e *subsonicNoopExtractor) Version() string                                     { return "noop" }

func putUsers(ctx context.Context, ds model.DataStore, users ...model.User) {
	GinkgoHelper()
	for i := range users {
		Expect(ds.User().Put(ctx, &users[i])).To(Succeed())
	}
}

// putAlbumArtists stores the artists as album artists of library libID, the role artist indexes list.
func putAlbumArtists(ctx context.Context, ds model.DataStore, libID int, artists ...model.Artist) {
	GinkgoHelper()
	persistencetest.PutArtists(ctx, ds, libID, artists...)
	persistencetest.SetArtistStats(ctx, libID, map[model.Role]model.ArtistStats{
		model.RoleAlbumArtist: {SongCount: 1, AlbumCount: 1},
	}, slice.Map(artists, func(a model.Artist) string { return a.ID })...)
}

// putItemArtwork records the resolved artwork of an item; an empty hash marks it known absent.
func putItemArtwork(ctx context.Context, ds model.DataStore, kind model.Kind, id, hash string) {
	GinkgoHelper()
	if hash != "" {
		Expect(ds.Artwork().PutImage(ctx, &model.Artwork{Hash: hash, Mime: "image/jpeg"})).To(Succeed())
	}
	Expect(ds.Artwork().PutItemArtwork(ctx, &model.ItemArtwork{ItemKind: kind.Prefix(), ItemID: id,
		ImageType: model.ImageTypePrimary, Hash: hash, Source: "external"})).To(Succeed())
}
