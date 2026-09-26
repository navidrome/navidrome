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

// login runs the full client flow for a user whose password is "pw": grant, resolve, then mint.
func login(ctx context.Context, svc *Service, u model.User) (*Issued, *Principal, *AccessToken) {
	issued, err := svc.Login(ctx, u.UserName, "pw", meta, nil)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	p, err := svc.ResolveGrant(ctx, issued.Secret, "")
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	tok, err := svc.Mint(ctx, p, nil)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return issued, p, tok
}

func mustMint(ctx context.Context, svc *Service, p *Principal) string {
	tok, err := svc.Mint(ctx, p, nil)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return tok.Token
}
