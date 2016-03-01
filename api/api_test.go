package api_test

import (
	"fmt"
	"github.com/astaxie/beego"
	"net/http"
	"net/http/httptest"
	_ "github.com/deluan/gosonic/conf"
)

const (
	testUser = "deluan"
	testPassword = "wordpass"
	testClient = "test"
	testVersion = "1.0.0"
)

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
