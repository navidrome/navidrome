package test

import (
	"fmt"
	"github.com/astaxie/beego"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
)

const (
	testUser     = "deluan"
	testPassword = "wordpass"
	testClient   = "test"
	testVersion  = "1.0.0"
)

func init() {
	_, file, _, _ := runtime.Caller(1)
	appPath, _ := filepath.Abs(filepath.Dir(filepath.Join(file, ".."+string(filepath.Separator))))
	beego.TestBeegoInit(appPath)

	noLog := os.Getenv("NOLOG")
	if noLog != "" {
		beego.SetLevel(beego.LevelError)
	}
}

func AddParams(url string) string {
	return fmt.Sprintf("%s?u=%s&p=%s&c=%s&v=%s", url, testUser, testPassword, testClient, testVersion)
}

func Get(url string, testCase string) (*http.Request, *httptest.ResponseRecorder) {
	r, _ := http.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Debug("testing", testCase, fmt.Sprintf("\nUrl: %s\nStatus Code: [%d]\n%s", r.URL, w.Code, w.Body.String()))

	return r, w
}
