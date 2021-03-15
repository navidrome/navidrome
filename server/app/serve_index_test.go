package app

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("serveIndex", func() {
	var ds model.DataStore
	mockUser := &mockedUserRepo{}
	fs := os.DirFS("tests/fixtures")

	BeforeEach(func() {
		ds = &tests.MockDataStore{MockedUser: mockUser}
		conf.Server.UILoginBackgroundURL = ""
	})

	It("adds app_config to index.html", func() {
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		Expect(w.Code).To(Equal(200))
		config := extractAppConfig(w.Body.String())
		Expect(config).To(BeAssignableToTypeOf(map[string]interface{}{}))
	})

	It("sets firstTime = true when User table is empty", func() {
		mockUser.empty = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("firstTime", true))
	})

	It("sets firstTime = false when User table is not empty", func() {
		mockUser.empty = false
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("firstTime", false))
	})

	It("sets baseURL", func() {
		conf.Server.BaseURL = "base_url_test"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("baseURL", "base_url_test"))
	})

	It("sets the uiLoginBackgroundURL", func() {
		conf.Server.UILoginBackgroundURL = "my_background_url"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("loginBackgroundURL", "my_background_url"))
	})

	It("sets the welcomeMessage", func() {
		conf.Server.UIWelcomeMessage = "Hello"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("welcomeMessage", "Hello"))
	})

	It("sets the enableTranscodingConfig", func() {
		conf.Server.EnableTranscodingConfig = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("enableTranscodingConfig", true))
	})

	It("sets the enableDownloads", func() {
		conf.Server.EnableDownloads = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("enableDownloads", true))
	})

	It("sets the gaTrackingId", func() {
		conf.Server.GATrackingID = "UA-12345"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("gaTrackingId", "UA-12345"))
	})

	It("sets the version", func() {
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

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
