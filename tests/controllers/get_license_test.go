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
	"encoding/xml"
	"fmt"
)

func init() {
	_, file, _, _ := runtime.Caller(1)
	appPath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, "../.." + string(filepath.Separator))))
	beego.TestBeegoInit(appPath)
}

// TestGet is a sample to run an endpoint test
func TestGetLicense(t *testing.T) {
	r, _ := http.NewRequest("GET", "/rest/getLicense.view", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Trace("testing", "TestGetLicense", fmt.Sprintf("Code[%d]\n%s", w.Code, w.Body.String()))

	Convey("Subject: GetLicense Endpoint\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
		Convey("The license should always be valid", func() {
			v := new(string)
			err := xml.Unmarshal(w.Body.Bytes(), &v)
			So(err, ShouldBeNil)
			So(w.Body.String(), ShouldContainSubstring, `license valid="true"`)
		})

	})
}

