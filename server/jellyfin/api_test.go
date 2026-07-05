package jellyfin

import (
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	It("serves the public handshake through the mounted handler", func() {
		ds := &tests.MockDataStore{}
		api := New(ds, nil, nil, nil, nil, nil, nil)
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/System/Info/Public", nil)
		api.ServeHTTP(w, r)
		Expect(w.Code).To(Equal(http.StatusOK))
	})
})
