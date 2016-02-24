package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"runtime"
	"encoding/xml"
	"path/filepath"
	_ "github.com/deluan/gosonic/routers"
	"github.com/astaxie/beego"
	. "github.com/smartystreets/goconvey/convey"
	"fmt"
	"github.com/deluan/gosonic/controllers/responses"
)

func init() {
	_, file, _, _ := runtime.Caller(1)
	appPath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, "../.." + string(filepath.Separator))))
	beego.TestBeegoInit(appPath)
}

func TestCheckParams(t *testing.T) {
	r, _ := http.NewRequest("GET", "/rest/ping.view", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Trace("testing", "TestCheckParams", fmt.Sprintf("\nUrl: %s\n\nCode[%d]\n%s", r.URL, w.Code, w.Body.String()))

	Convey("Subject: Validation\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The errorCode should be 10", func() {
			So(w.Body.String(), ShouldContainSubstring, `error code="10" message=`)
		})
		Convey("The status should be 'fail'", func() {
			v := responses.Subsonic{}
			xml.Unmarshal(w.Body.Bytes(), &v)
			So(v.Status, ShouldEqual, "fail")
		})
	})
}

func TestAuthentication(t *testing.T) {
	r, _ := http.NewRequest("GET", "/rest/ping.view?u=INVALID&p=INVALID&c=test&v=1.0.0", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Trace("testing", "TestCheckParams", fmt.Sprintf("\nUrl: %s\n\nCode[%d]\n%s", r.URL, w.Code, w.Body.String()))

	Convey("Subject: Validation\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The errorCode should be 10", func() {
			So(w.Body.String(), ShouldContainSubstring, `error code="40" message=`)
		})
		Convey("The status should be 'fail'", func() {
			v := responses.Subsonic{}
			xml.Unmarshal(w.Body.Bytes(), &v)
			So(v.Status, ShouldEqual, "fail")
		})
	})
}


