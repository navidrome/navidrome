package subsonic

import (
	"encoding/json"
	"encoding/xml"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("sendResponse", func() {
	var (
		w       *httptest.ResponseRecorder
		r       *http.Request
		payload *responses.Subsonic
	)

	BeforeEach(func() {
		w = httptest.NewRecorder()
		r = httptest.NewRequest("GET", "/somepath", nil)
		payload = &responses.Subsonic{
			Status:  responses.StatusOK,
			Version: "1.16.1",
		}
	})

	When("format is JSON", func() {
		It("should set Content-Type to application/json and return the correct body", func() {
			q := r.URL.Query()
			q.Add("f", "json")
			r.URL.RawQuery = q.Encode()

			sendResponse(w, r, payload)

			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))
			Expect(w.Body.String()).NotTo(BeEmpty())

			var wrapper responses.JsonWrapper
			err := json.Unmarshal(w.Body.Bytes(), &wrapper)
			Expect(err).NotTo(HaveOccurred())
			Expect(wrapper.Subsonic.Status).To(Equal(payload.Status))
			Expect(wrapper.Subsonic.Version).To(Equal(payload.Version))
		})
	})

	When("format is JSONP", func() {
		It("should set Content-Type to application/javascript and return the correct callback body", func() {
			q := r.URL.Query()
			q.Add("f", "jsonp")
			q.Add("callback", "testCallback")
			r.URL.RawQuery = q.Encode()

			sendResponse(w, r, payload)

			Expect(w.Header().Get("Content-Type")).To(Equal("application/javascript"))
			body := w.Body.String()
			Expect(body).To(SatisfyAll(
				HavePrefix("testCallback("),
				HaveSuffix(")"),
			))

			// Extract JSON from the JSONP response
			jsonBody := body[strings.Index(body, "(")+1 : strings.LastIndex(body, ")")]
			var wrapper responses.JsonWrapper
			err := json.Unmarshal([]byte(jsonBody), &wrapper)
			Expect(err).NotTo(HaveOccurred())
			Expect(wrapper.Subsonic.Status).To(Equal(payload.Status))
		})
	})

	When("format is XML or unspecified", func() {
		It("should set Content-Type to application/xml and return the correct body", func() {
			// No format specified, expecting XML by default
			sendResponse(w, r, payload)

			Expect(w.Header().Get("Content-Type")).To(Equal("application/xml"))
			var subsonicResponse responses.Subsonic
			err := xml.Unmarshal(w.Body.Bytes(), &subsonicResponse)
			Expect(err).NotTo(HaveOccurred())
			Expect(subsonicResponse.Status).To(Equal(payload.Status))
			Expect(subsonicResponse.Version).To(Equal(payload.Version))
		})
	})

	When("an error occurs during marshalling", func() {
		It("should return a fail response", func() {
			payload.Song = &responses.Child{OpenSubsonicChild: &responses.OpenSubsonicChild{}}
			// An +Inf value will cause an error when marshalling to JSON
			payload.Song.ReplayGain = responses.ReplayGain{TrackGain: math.Inf(1)}
			q := r.URL.Query()
			q.Add("f", "json")
			r.URL.RawQuery = q.Encode()

			sendResponse(w, r, payload)

			Expect(w.Code).To(Equal(http.StatusOK))
			var wrapper responses.JsonWrapper
			err := json.Unmarshal(w.Body.Bytes(), &wrapper)
			Expect(err).NotTo(HaveOccurred())
			Expect(wrapper.Subsonic.Status).To(Equal(responses.StatusFailed))
			Expect(wrapper.Subsonic.Version).To(Equal(payload.Version))
			Expect(wrapper.Subsonic.Error.Message).To(ContainSubstring("json: unsupported value: +Inf"))
		})
	})

})
