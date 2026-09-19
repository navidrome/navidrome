package nativeapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/core/quickconnect"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Quick Connect endpoints", func() {
	var (
		qc      quickconnect.QuickConnect
		pending quickconnect.Request
		user    = model.User{ID: "u1", UserName: "alice"}
	)

	BeforeEach(func() {
		qc = quickconnect.New()
		var err error
		pending, err = qc.Initiate(quickconnect.Device{ID: "dev-1", Name: "Pixel 7", App: "Finamp", AppVersion: "1.0.0"})
		Expect(err).ToNot(HaveOccurred())
	})

	asUser := func(r *http.Request) *http.Request {
		return r.WithContext(request.WithUser(r.Context(), user))
	}
	decode := func(w *httptest.ResponseRecorder) quickConnectDevice {
		var res quickConnectDevice
		Expect(json.Unmarshal(w.Body.Bytes(), &res)).To(Succeed())
		return res
	}
	finamp := quickConnectDevice{AppName: "Finamp", AppVersion: "1.0.0", DeviceName: "Pixel 7"}

	Describe("GET /quickconnect", func() {
		lookup := func(code string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			lookupQuickConnect(qc)(w, asUser(httptest.NewRequest("GET", "/quickconnect?code="+code, nil)))
			return w
		}

		It("describes the device waiting for the code", func() {
			w := lookup(pending.Code)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(decode(w)).To(Equal(finamp))
		})

		It("does not approve the code", func() {
			lookup(pending.Code)
			got, _ := qc.Status(pending.Secret)
			Expect(got.Authorized()).To(BeFalse())
		})

		It("returns 404 for an unknown code", func() {
			Expect(lookup("000000").Code).To(Equal(http.StatusNotFound))
		})

		It("returns 409 for a code that is already approved", func() {
			_, _ = qc.Authorize(pending.Code, "someone")
			Expect(lookup(pending.Code).Code).To(Equal(http.StatusConflict))
		})
	})

	Describe("POST /quickconnect/authorize", func() {
		authorize := func(body string) *httptest.ResponseRecorder {
			w := httptest.NewRecorder()
			authorizeQuickConnect(qc)(w, asUser(httptest.NewRequest("POST", "/quickconnect/authorize", strings.NewReader(body))))
			return w
		}

		It("approves the code for the signed-in user", func() {
			w := authorize(`{"code":"` + pending.Code + `"}`)
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(decode(w)).To(Equal(finamp))
			got, _ := qc.Status(pending.Secret)
			Expect(got.UserID).To(Equal(user.ID))
		})

		It("returns 404 for an unknown code", func() {
			Expect(authorize(`{"code":"000000"}`).Code).To(Equal(http.StatusNotFound))
		})

		It("returns 409 for a code that is already approved", func() {
			_, _ = qc.Authorize(pending.Code, "someone")
			Expect(authorize(`{"code":"` + pending.Code + `"}`).Code).To(Equal(http.StatusConflict))
		})

		It("returns 400 for a bad body", func() {
			Expect(authorize(`nope`).Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("route registration", func() {
		serve := func() int {
			r := chi.NewRouter()
			(&Router{quickConnect: qc}).addQuickConnectRoute(r)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, asUser(httptest.NewRequest("GET", "/quickconnect?code="+pending.Code, nil)))
			return w.Code
		}

		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.Jellyfin.Enabled = true
			conf.Server.Jellyfin.QuickConnect = true
		})

		It("mounts the routes when Jellyfin Quick Connect is on", func() {
			Expect(serve()).To(Equal(http.StatusOK))
		})

		It("rate-limits code attempts when a login limit is configured", func() {
			conf.Server.AuthRequestLimit = 2
			conf.Server.AuthWindowLength = time.Minute
			r := chi.NewRouter()
			(&Router{quickConnect: qc}).addQuickConnectRoute(r)
			attempt := func() int {
				w := httptest.NewRecorder()
				req := httptest.NewRequest("GET", "/quickconnect?code=000000", nil)
				req.RemoteAddr = "10.0.0.1:1234"
				r.ServeHTTP(w, asUser(req))
				return w.Code
			}
			Expect(attempt()).To(Equal(http.StatusNotFound))
			Expect(attempt()).To(Equal(http.StatusNotFound))
			Expect(attempt()).To(Equal(http.StatusTooManyRequests))
		})

		It("skips the routes when the Jellyfin API is off", func() {
			conf.Server.Jellyfin.Enabled = false
			Expect(serve()).To(Equal(http.StatusNotFound))
		})

		It("skips the routes when Quick Connect is off", func() {
			conf.Server.Jellyfin.QuickConnect = false
			Expect(serve()).To(Equal(http.StatusNotFound))
		})
	})
})
