package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"runtime"
	"path/filepath"
	_ "github.com/deluan/gosonic/routers"

	"github.com/astaxie/beego"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	_, file, _, _ := runtime.Caller(1)
	apppath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, "../.." + string(filepath.Separator))))
	beego.TestBeegoInit(apppath)
}

// TestGet is a sample to run an endpoint test
func TestPing(t *testing.T) {
	r, _ := http.NewRequest("GET", "/rest/ping.view", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Trace("testing", "TestPing", "Code[%d]\n%s", w.Code, w.Body.String())

	Convey("Subject: Ping Endpoint\n", t, func() {
		Convey("Status Code Should Be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The Result Should Not Be Empty", func() {
			So(w.Body.Len(), ShouldBeGreaterThan, 0)
		})
		Convey("The Result Should Be A Pong", func() {
			So(w.Body.String(), ShouldEqual, "<subsonic-response xmlns=\"http://subsonic.org/restapi\" status=\"ok\" version=\"1.0.0\"></subsonic-response>")
		})

	})
}

