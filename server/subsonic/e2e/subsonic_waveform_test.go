package e2e

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/conf/configtest"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Waveform endpoint", Ordered, func() {
	var trackID string

	BeforeAll(func() {
		setupTestDB()
		songs, err := ds.MediaFile().GetAll(ctx, model.QueryOptions{Max: 1, Sort: "title"})
		Expect(err).ToNot(HaveOccurred())
		Expect(songs).ToNot(BeEmpty())
		trackID = songs[0].ID
	})

	// Clients that call getWaveform without checking for the songWaveform extension must get a
	// regular Subsonic error, in the format they asked for, not a non-Subsonic page.
	Describe("when waveforms are disabled", func() {
		BeforeEach(func() {
			DeferCleanup(configtest.SetupConfig())
			conf.Server.EnableWaveform = false
		})

		It("is not advertised as an extension", func() {
			resp := doReq("getOpenSubsonicExtensions")
			Expect(resp.OpenSubsonicExtensions).ToNot(BeNil())
			for _, ext := range *resp.OpenSubsonicExtensions {
				Expect(ext.Name).ToNot(Equal("songWaveform"))
			}
		})

		It("returns a failed JSON response", func() {
			w := doRawReq("getWaveform", "id", trackID)
			Expect(w.Code).To(Equal(http.StatusOK))
			resp := parseJSONResponse(w)
			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
			Expect(resp.Error.Code).To(Equal(responses.ErrorGeneric))
			Expect(resp.Waveform).To(BeNil())
		})

		It("returns a failed XML response", func() {
			r := buildReq(adminUser, "getWaveform", "id", trackID)
			q := r.URL.Query()
			q.Set("f", "xml")
			r.URL.RawQuery = q.Encode()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r)
			Expect(w.Code).To(Equal(http.StatusOK))
			var resp responses.Subsonic
			Expect(xml.Unmarshal(w.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Status).To(Equal(responses.StatusFailed))
			Expect(resp.Error).ToNot(BeNil())
		})
	})

	It("returns a failed response when enabled but no waveform service is available", func() {
		DeferCleanup(configtest.SetupConfig())
		conf.Server.EnableWaveform = true
		resp := doReq("getWaveform", "id", trackID)
		Expect(resp.Status).To(Equal(responses.StatusFailed))
	})
})
