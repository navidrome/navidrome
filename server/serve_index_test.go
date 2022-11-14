package server

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("serveIndex", func() {
	var ds model.DataStore
	mockUser := &mockedUserRepo{}
	fs := os.DirFS("tests/fixtures")

	BeforeEach(func() {
		ds = &tests.MockDataStore{MockedUser: mockUser}
		DeferCleanup(configtest.SetupConfig())
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

	It("includes the VariousArtistsID", func() {
		mockUser.empty = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("variousArtistsId", consts.VariousArtistsID))
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

	It("sets the enableLoved", func() {
		conf.Server.EnableFavourites = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("enableFavourites", true))
	})

	It("sets the enableStarRating", func() {
		conf.Server.EnableStarRating = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("enableStarRating", true))
	})

	It("sets the defaultTheme", func() {
		conf.Server.DefaultTheme = "Light"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("defaultTheme", "Light"))
	})

	It("sets the defaultLanguage", func() {
		conf.Server.DefaultLanguage = "pt"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("defaultLanguage", "pt"))
	})

	It("sets the defaultUIVolume", func() {
		conf.Server.DefaultUIVolume = 45
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("defaultUIVolume", float64(45)))
	})

	It("sets the enableCoverAnimation", func() {
		conf.Server.EnableCoverAnimation = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("enableCoverAnimation", true))
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
		Expect(config).To(HaveKeyWithValue("version", consts.Version))
	})

	It("sets the losslessFormats", func() {
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		expected := strings.ToUpper(strings.Join(consts.LosslessFormats, ","))
		Expect(config).To(HaveKeyWithValue("losslessFormats", expected))
	})

	It("sets the enableUserEditing", func() {
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("enableUserEditing", true))
	})

	It("sets the devEnableShare", func() {
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("devEnableShare", false))
	})

	It("sets the devSidebarPlaylists", func() {
		conf.Server.DevSidebarPlaylists = true

		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("devSidebarPlaylists", true))
	})

	It("sets the lastFMEnabled", func() {
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("lastFMEnabled", true))
	})

	It("sets the lastFMApiKey", func() {
		conf.Server.LastFM.ApiKey = "APIKEY-123"
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("lastFMApiKey", "APIKEY-123"))
	})

	It("sets the devShowArtistPage", func() {
		conf.Server.DevShowArtistPage = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("devShowArtistPage", true))
	})

	It("sets the listenBrainzEnabled", func() {
		conf.Server.ListenBrainz.Enabled = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("listenBrainzEnabled", true))
	})

	It("sets the enableReplayGain", func() {
		conf.Server.EnableReplayGain = true
		r := httptest.NewRequest("GET", "/index.html", nil)
		w := httptest.NewRecorder()

		serveIndex(ds, fs)(w, r)

		config := extractAppConfig(w.Body.String())
		Expect(config).To(HaveKeyWithValue("enableReplayGain", true))
	})

	Describe("loginBackgroundURL", func() {
		Context("empty BaseURL", func() {
			BeforeEach(func() {
				conf.Server.BaseURL = "/"
			})
			When("it is the default URL", func() {
				It("points to the default URL", func() {
					conf.Server.UILoginBackgroundURL = consts.DefaultUILoginBackgroundURL
					r := httptest.NewRequest("GET", "/index.html", nil)
					w := httptest.NewRecorder()

					serveIndex(ds, fs)(w, r)

					config := extractAppConfig(w.Body.String())
					Expect(config).To(HaveKeyWithValue("loginBackgroundURL", consts.DefaultUILoginBackgroundURL))
				})
			})
			When("it is the default offline URL", func() {
				It("points to the offline URL", func() {
					conf.Server.UILoginBackgroundURL = consts.DefaultUILoginBackgroundURLOffline
					r := httptest.NewRequest("GET", "/index.html", nil)
					w := httptest.NewRecorder()

					serveIndex(ds, fs)(w, r)

					config := extractAppConfig(w.Body.String())
					Expect(config).To(HaveKeyWithValue("loginBackgroundURL", consts.DefaultUILoginBackgroundURLOffline))
				})
			})
			When("it is a custom URL", func() {
				It("points to the offline URL", func() {
					conf.Server.UILoginBackgroundURL = "https://example.com/images/1.jpg"
					r := httptest.NewRequest("GET", "/index.html", nil)
					w := httptest.NewRecorder()

					serveIndex(ds, fs)(w, r)

					config := extractAppConfig(w.Body.String())
					Expect(config).To(HaveKeyWithValue("loginBackgroundURL", "https://example.com/images/1.jpg"))
				})
			})
		})
		Context("with a BaseURL", func() {
			BeforeEach(func() {
				conf.Server.BaseURL = "/music"
			})
			When("it is the default URL", func() {
				It("points to the default URL with BaseURL prefix", func() {
					conf.Server.UILoginBackgroundURL = consts.DefaultUILoginBackgroundURL
					r := httptest.NewRequest("GET", "/index.html", nil)
					w := httptest.NewRecorder()

					serveIndex(ds, fs)(w, r)

					config := extractAppConfig(w.Body.String())
					Expect(config).To(HaveKeyWithValue("loginBackgroundURL", "/music"+consts.DefaultUILoginBackgroundURL))
				})
			})
			When("it is the default offline URL", func() {
				It("points to the offline URL", func() {
					conf.Server.UILoginBackgroundURL = consts.DefaultUILoginBackgroundURLOffline
					r := httptest.NewRequest("GET", "/index.html", nil)
					w := httptest.NewRecorder()

					serveIndex(ds, fs)(w, r)

					config := extractAppConfig(w.Body.String())
					Expect(config).To(HaveKeyWithValue("loginBackgroundURL", consts.DefaultUILoginBackgroundURLOffline))
				})
			})
			When("it is a custom URL", func() {
				It("points to the offline URL", func() {
					conf.Server.UILoginBackgroundURL = "https://example.com/images/1.jpg"
					r := httptest.NewRequest("GET", "/index.html", nil)
					w := httptest.NewRecorder()

					serveIndex(ds, fs)(w, r)

					config := extractAppConfig(w.Body.String())
					Expect(config).To(HaveKeyWithValue("loginBackgroundURL", "https://example.com/images/1.jpg"))
				})
			})
		})
	})
})

var appConfigRegex = regexp.MustCompile(`(?m)window.__APP_CONFIG__=(.*);</script>`)

func extractAppConfig(body string) map[string]interface{} {
	config := make(map[string]interface{})
	match := appConfigRegex.FindStringSubmatch(body)
	if match == nil {
		return config
	}
	str, err := strconv.Unquote(match[1])
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
