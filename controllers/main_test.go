package controllers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/astaxie/beego"
	_ "github.com/deluan/gosonic/init"
	"github.com/deluan/gosonic/tests"
	. "github.com/smartystreets/goconvey/convey"
)

func TestErrorHandler(t *testing.T) {
	tests.Init(t, false)

	r, _ := http.NewRequest("GET", "/INVALID_PATH", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Debug("testing", "TestErrorHandler", fmt.Sprintf("\nUrl: %s\nStatus Code: [%d]\n%s", r.URL, w.Code, w.Body.String()))

	Convey("When requesting an non-existing URL\n", t, func() {
		Convey("Then the status code should be 404", func() {
			So(w.Code, ShouldEqual, 404)
		})
	})
}
