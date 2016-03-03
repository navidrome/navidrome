package api_test

import (
	"testing"

	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	. "github.com/deluan/gosonic/tests"
	"github.com/deluan/gosonic/tests/mocks"
	"github.com/deluan/gosonic/utils"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"github.com/astaxie/beego"
	"fmt"
)

func getCoverArt(params ...string) (*http.Request, *httptest.ResponseRecorder) {
	url := AddParams("/rest/getCoverArt.view", params...)
	r, _ := http.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)
	beego.Debug("testing TestGetCoverArtDirectory", fmt.Sprintf("\nUrl: %s\nStatus Code: [%d]\n%#v", r.URL, w.Code, w.HeaderMap))
	return r, w
}

func TestGetCoverArt(t *testing.T) {
	Init(t, false)

	mockMediaFileRepo := mocks.CreateMockMediaFileRepo()
	utils.DefineSingleton(new(domain.MediaFileRepository), func() domain.MediaFileRepository {
		return mockMediaFileRepo
	})

	Convey("Subject: GetCoverArt Endpoint", t, func() {
		Convey("Should fail if missing Id parameter", func() {
			_, w := getCoverArt()

			So(w.Body, ShouldReceiveError, responses.ERROR_MISSING_PARAMETER)
		})
		Convey("When id is not found", func() {
			mockMediaFileRepo.SetData(`[]`, 1)
			_, w := getCoverArt("id=NOT_FOUND")

			So(w.Body.Bytes(), ShouldMatchMD5, "963552b04e87a5a55e993f98a0fbdf82")
		})
		Convey("When id is found", func() {
			mockMediaFileRepo.SetData(`[{"Id":"2","HasCoverArt":true,"Path":"tests/fixtures/01 Invisible (RED) Edit Version.mp3"}]`, 1)
			_, w := getCoverArt("id=2")

			So(w.Body.Bytes(), ShouldMatchMD5, "e859a71cd1b1aaeb1ad437d85b306668")
		})
		Reset(func() {
			mockMediaFileRepo.SetData("[]", 0)
			mockMediaFileRepo.SetError(false)
		})
	})
}
