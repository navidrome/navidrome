package api_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/astaxie/beego"
)

const (
	testUser     = "deluan"
	testPassword = "wordpass"
	testClient   = "test"
	testVersion  = "1.0.0"
)

func AddParams(endpoint string, params ...string) string {
	url := fmt.Sprintf("%s?u=%s&p=%s&c=%s&v=%s&f=json", endpoint, testUser, testPassword, testClient, testVersion)
	if len(params) > 0 {
		url = url + "&" + strings.Join(params, "&")
	}
	return url
}

func Get(url string, testCase string) (*http.Request, *httptest.ResponseRecorder) {
	r, _ := http.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Debug("testing", testCase, fmt.Sprintf("\nUrl: %s\nStatus Code: [%d]\n%s", r.URL, w.Code, w.Body.String()))

	return r, w
}

func GetWithHeader(url string, header, value, testCase string) (*http.Request, *httptest.ResponseRecorder) {
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Add(header, value)
	w := httptest.NewRecorder()
	beego.BeeApp.Handlers.ServeHTTP(w, r)

	beego.Debug("testing", testCase, fmt.Sprintf("\nUrl: %s\nStatus Code: [%d]\n%s", r.URL, w.Code, w.Body.String()))

	return r, w
}
