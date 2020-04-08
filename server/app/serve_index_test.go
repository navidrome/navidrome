package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"

	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/persistence"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ServeIndex", func() {
	var ds model.DataStore
	mockUser := &mockedUserRepo{}
	fs := http.Dir("tests/fixtures")

	BeforeEach(func() {
		ds = &persistence.MockDataStore{MockedUser: mockUser}
		conf.Server.UILoginBackgroundURL = ""
	})

	It("adds app_config to index.html", func() {
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		ServeIndex(ds, fs)(w, r)

		Expect(w.Code).To(Equal(200))
		config := extractAppConfig(w.Body.String())
		Expect(config).To(BeAssignableToTypeOf(map[string]interface{}{}))
	})

	It("sets firstTime = true when User table is empty", func() {
		mockUser.empty = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		ServeIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("firstTime", true))
	})

	It("sets firstTime = false when User table is not empty", func() {
		mockUser.empty = false
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		ServeIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("firstTime", false))
	})

	It("sets baseURL", func() {
		conf.Server.BaseURL = "base_url_test"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		ServeIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("baseURL", "base_url_test"))
	})

	It("sets the uiLoginBackgroundURL", func() {
		conf.Server.UILoginBackgroundURL = "my_background_url"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		ServeIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("loginBackgroundURL", "my_background_url"))
	})

	It("sets the version", func() {
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		ServeIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("version", consts.Version()))
	})
})

var appConfigRegex = regexp.MustCompile(`(?m)window.__APP_CONFIG__="([^"]*)`)

func extractAppConfig(body string) map[string]interface{} {
	config := make(map[string]interface{})
	match := appConfigRegex.FindStringSubmatch(body)
	if match == nil {
		return config
	}
	str, err := strconv.Unquote("\"" + match[1] + "\"")
	if err != nil {
		panic(fmt.Sprintf("%s: %s", match[1], err))
	}
	if err := json.Unmarshal([]byte(str), &config); err != nil {
		panic(err)
	}
	return config
}

type mockedUserRepo struct {
	model.UserRepository
	empty bool
}

func (u *mockedUserRepo) CountAll(...model.QueryOptions) (int64, error) {
	if u.empty {
		return 0, nil
	}
	return 1, nil
}
