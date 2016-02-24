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
	"fmt"
)

func init() {
	_, file, _, _ := runtime.Caller(1)
	appPath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, "../.." + string(filepath.Separator))))
	beego.TestBeegoInit(appPath)
}

// TestGet is a sample to run an endpoint test
func TestGetMusicFolders(t *testing.T) {
	r, _ := http.NewRequest("GET", "/rest/getMusicFolders.view", nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Trace("testing", "TestGetMusicFolders", fmt.Sprintf("Code[%d]\n%s", w.Code, w.Body.String()))

	Convey("Subject: GetMusicFolders Endpoint\n", t, func() {
		Convey("Status code should be 200", func() {
			So(w.Code, ShouldEqual, 200)
		})
	})
}

