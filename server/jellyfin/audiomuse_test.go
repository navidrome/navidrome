package jellyfin

import (
	"encoding/json"
	"net/http/httptest"

	"github.com/navidrome/navidrome/consts"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AudioMuse info", func() {
	It("returns the version and available endpoints, excluding info itself", func() {
		api := &Router{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/AudioMuseAI/info", nil)

		api.audioMuseInfo(w, r)

		Expect(w.Code).To(Equal(200))
		var body audioMuseInfoResponse
		Expect(json.Unmarshal(w.Body.Bytes(), &body)).To(Succeed())
		Expect(body.Version).To(Equal(consts.Version))
		Expect(body.AvailableEndpoints).To(ConsistOf(
			"GET /AudioMuseAI/similar_tracks",
			"GET /AudioMuseAI/find_path",
		))
	})
})
