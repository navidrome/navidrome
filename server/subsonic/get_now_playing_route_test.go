package subsonic

import (
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetNowPlaying route", func() {
	var (
		router  *Router
		ds      *tests.MockDataStore
		players core.Players
		w       *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		DeferCleanup(configtest.SetupConfig())
		ds = &tests.MockDataStore{MockedUser: tests.CreateMockUserRepo()}
		_ = ds.User(nil).Put(&model.User{UserName: "admin", NewPassword: "wordpass"})
		players = fakePlayers{}
		w = httptest.NewRecorder()
	})

	It("returns 501 when disabled", func() {
		conf.Server.EnableNowPlaying = false
		router = New(ds, nil, nil, nil, players, nil, nil, nil, nil, nil, nil, nil)
		r := httptest.NewRequest("GET", "/getNowPlaying?u=admin&p=wordpass&v=1.16.1&c=test", nil)
		router.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusNotImplemented))
	})
})

type fakePlayers struct{}

func (fakePlayers) Get(_ context.Context, id string) (*model.Player, error) {
	return &model.Player{ID: id}, nil
}

func (fakePlayers) Register(_ context.Context, id, client, userAgent, ip string) (*model.Player, *model.Transcoding, error) {
	return &model.Player{ID: id}, nil, nil
}
