package apiauth

import (
	"context"
	"path/filepath"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/db"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
	"github.com/navidrome/navidrome/persistence"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var realDS model.DataStore

// One database for the whole suite: db.Db() is a process-wide singleton.
var _ = BeforeSuite(func() {
	DeferCleanup(configtest.SetupConfig())
	conf.Server.DbPath = filepath.Join(GinkgoT().TempDir(), "apiauth.db") + "?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000"
	DeferCleanup(db.Init(GinkgoT().Context()))
	realDS = persistence.New(db.Db())
})

func createUser(ctx context.Context, password string, admin bool) model.User {
	name := "user-" + id.NewRandom()
	u := model.User{UserName: name, Name: name, NewPassword: password, IsAdmin: admin}
	ExpectWithOffset(1, realDS.User().Put(ctx, &u)).To(Succeed())
	stored, err := realDS.User().FindByUsername(ctx, name)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return *stored
}

// login signs u in (nil scopes asks for all) and authenticates with the new grant secret.
func login(ctx context.Context, svc *Service, u model.User, password string, scopes []string) (*Issued, *Principal) {
	issued, err := svc.Login(ctx, u.UserName, password, meta, scopes)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	p, err := svc.Authenticate(ctx, issued.Secret, "")
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return issued, p
}
