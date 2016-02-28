package controllers_test

import (
	"github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"net/http"
	"net/http/httptest"
	"github.com/astaxie/beego"
	"fmt"
	_ "github.com/deluan/gosonic/routers"
)

func TestMainController(t *testing.T) {
	tests.Init(t, false)

	r, _ := http.NewRequest("GET", "/INVALID_PATH", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Debug("testing", "TestMainController", fmt.Sprintf("\nUrl: %s\nStatus Code: [%d]\n%s", r.URL, w.Code, w.Body.String()))

	Convey("Subject: Error404\n", t, func() {
		Convey("Status code should be 404", func() {
			So(w.Code, ShouldEqual, 404)
		})
	})
}