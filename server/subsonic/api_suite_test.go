package subsonic

import (
	"context"
	"io/fs"
	"testing"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/storage/local"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/metadata"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
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

// putTracks stores the media files in library 1 unless they name another, with a path derived from each ID.
func putTracks(ctx context.Context, ds model.DataStore, tracks ...model.MediaFile) {
	GinkgoHelper()
	for i := range tracks {
		mf := tracks[i]
		if mf.LibraryID == 0 {
			mf.LibraryID = model.DefaultLibraryID
		}
		if mf.Path == "" {
			mf.Path = mf.ID + ".mp3"
		}
		Expect(ds.MediaFile().Put(ctx, &mf)).To(Succeed())
	}
}

func putAlbums(ctx context.Context, ds model.DataStore, albums ...model.Album) {
	GinkgoHelper()
	for i := range albums {
		al := albums[i]
		if al.LibraryID == 0 {
			al.LibraryID = model.DefaultLibraryID
		}
		Expect(ds.Album().Put(ctx, &al)).To(Succeed())
	}
}

// putAlbumArtists stores the artists as album artists of library libID, the role artist indexes list.
func putAlbumArtists(ctx context.Context, ds model.DataStore, libID int, artists ...model.Artist) {
	GinkgoHelper()
	for i := range artists {
		Expect(ds.Artist().Put(ctx, &artists[i])).To(Succeed())
		Expect(ds.Library().AddArtist(ctx, libID, artists[i].ID)).To(Succeed())
		_, err := db.Db().ExecContext(ctx, `UPDATE library_artist SET stats = '{"albumartist":{"m":1,"a":1}}' WHERE library_id = ? AND artist_id = ?`,
			libID, artists[i].ID)
		Expect(err).ToNot(HaveOccurred())
	}
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
