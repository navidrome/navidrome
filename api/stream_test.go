package api_test

import (
	"testing"

	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/persistence"
	. "github.com/deluan/gosonic/tests"
	"github.com/deluan/gosonic/utils"
	. "github.com/smartystreets/goconvey/convey"
)

func stream(params ...string) (*http.Request, *httptest.ResponseRecorder) {
	url := AddParams("/rest/stream.view", params...)
	r, _ := http.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)
	beego.Debug("testing TestStream", fmt.Sprintf("\nUrl: %s\nStatus Code: [%d]\n%#v", r.URL, w.Code, w.HeaderMap))
	return r, w
}

func TestStream(t *testing.T) {
	Init(t, false)

	mockMediaFileRepo := persistence.CreateMockMediaFileRepo()
	utils.DefineSingleton(new(domain.MediaFileRepository), func() domain.MediaFileRepository {
		return mockMediaFileRepo
	})

	Convey("Subject: Stream Endpoint", t, func() {
		Convey("Should fail if missing Id parameter", func() {
			_, w := stream()

			So(w.Body, ShouldReceiveError, responses.ERROR_MISSING_PARAMETER)
		})
		Convey("When id is not found", func() {
			mockMediaFileRepo.SetData(`[]`, 1)
			_, w := stream("id=NOT_FOUND")

			So(w.Body, ShouldReceiveError, responses.ERROR_DATA_NOT_FOUND)
		})
		Convey("When id is found", func() {
			mockMediaFileRepo.SetData(`[{"Id":"2","HasCoverArt":true,"Path":"tests/fixtures/01 Invisible (RED) Edit Version.mp3"}]`, 1)
			_, w := stream("id=2")

			So(w.Body.Bytes(), ShouldMatchMD5, "258dd4f0e70ee5c8dee3cb33c966acec")
		})
		Reset(func() {
			mockMediaFileRepo.SetData("[]", 0)
			mockMediaFileRepo.SetError(false)
		})
	})
}
