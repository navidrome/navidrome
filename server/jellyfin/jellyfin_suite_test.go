package jellyfin

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/persistence/persistencetest"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestJellyfinApi(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	persistencetest.Init(t)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Jellyfin API Suite")
}

var _ = configtest.RestoreAfterEachSpec()
var _ = persistencetest.ResetBeforeEachSpec()

// testID maps a readable label to a deterministic canonical id, so fixtures exercise the same
// id shape production uses.
func testID(label string) string { return id.NewHash("jellyfin-test", label) }

// invoke runs a handler through normalizeQueryKeys, mirroring the router. These unit tests call
// handlers directly (with withChiURLParam for path params) instead of routing, so without this the
// case-insensitive query folding real requests get would be skipped and PascalCase params dropped.
func invoke(h http.HandlerFunc, w http.ResponseWriter, r *http.Request) {
	normalizeQueryKeys(h).ServeHTTP(w, r)
}

// setTokenEpoch writes token_epoch directly: User().Put never stores it.
func setTokenEpoch(userID string, epoch int) {
	GinkgoHelper()
	_, err := db.Db().ExecContext(GinkgoT().Context(), "UPDATE user SET token_epoch = ? WHERE id = ?", epoch, userID)
	Expect(err).ToNot(HaveOccurred())
}

type failingUserRepo struct {
	model.UserRepository
	err error
}

func (f failingUserRepo) Get(context.Context, string) (*model.User, error) { return nil, f.err }

type failingPropertyRepo struct {
	model.PropertyRepository
	err error
}

func (f failingPropertyRepo) Get(context.Context, string) (string, error) { return "", f.err }

// putUser stores u and returns it as the repository loads it, with the libraries it can access.
func putUser(ctx context.Context, ds model.DataStore, u model.User) model.User {
	GinkgoHelper()
	Expect(ds.User().Put(ctx, &u)).To(Succeed())
	stored, err := ds.User().Get(ctx, u.ID)
	Expect(err).ToNot(HaveOccurred())
	return *stored
}

// putArtists stores artists in library 1.
func putArtists(ctx context.Context, ds model.DataStore, artists ...model.Artist) {
	GinkgoHelper()
	putArtistsIn(ctx, ds, model.DefaultLibraryID, artists...)
}

// putArtistsIn stores artists in libID, credited as artist and album artist so the role-filtered
// listings include them.
func putArtistsIn(ctx context.Context, ds model.DataStore, libID int, artists ...model.Artist) {
	GinkgoHelper()
	ids := make([]string, len(artists))
	inTx(ds, func(tx model.DataStore) {
		for i, ar := range artists {
			if ar.OrderArtistName == "" {
				ar.OrderArtistName = strings.ToLower(ar.Name)
			}
			persistencetest.PutArtists(ctx, tx, libID, ar)
			ids[i] = ar.ID
		}
	})
	persistencetest.SetArtistStats(ctx, libID, map[model.Role]model.ArtistStats{
		model.RoleArtist: {SongCount: 1}, model.RoleAlbumArtist: {SongCount: 1},
	}, ids...)
}

// inTx runs seed in one transaction, which makes seeding thousands of rows fast.
func inTx(ds model.DataStore, seed func(tx model.DataStore)) {
	GinkgoHelper()
	Expect(ds.WithTx(func(tx model.DataStore) error {
		seed(tx)
		return nil
	})).To(Succeed())
}

// credits names the stored artists artistIDs in role.
func credits(role model.Role, artistIDs ...string) model.Participants {
	p := model.Participants{}
	for _, id := range artistIDs {
		p.Add(role, model.Artist{ID: id})
	}
	return p
}
