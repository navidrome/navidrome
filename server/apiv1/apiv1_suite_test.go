package apiv1

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers"
	"github.com/getkin/kin-openapi/routers/gorillamux"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/api"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAPIv1(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "API v1 Suite")
}

var specRouter routers.Router

var _ = BeforeSuite(func() {
	doc, err := openapi3.NewLoader().LoadFromData(api.SpecJSON())
	Expect(err).ToNot(HaveOccurred())
	specRouter, err = gorillamux.NewRouter(doc)
	Expect(err).ToNot(HaveOccurred())
})

// serve routes req through h mounted at /api/v1 and asserts the response conforms to the spec.
func serve(h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	root := chi.NewRouter()
	root.Mount("/api/v1", h)
	w := httptest.NewRecorder()
	root.ServeHTTP(w, req)
	validateAgainstSpec(req, w)
	return w
}

func validateAgainstSpec(req *http.Request, w *httptest.ResponseRecorder) {
	route, pathParams, err := specRouter.FindRoute(req)
	if errors.Is(err, routers.ErrPathNotFound) || errors.Is(err, routers.ErrMethodNotAllowed) {
		return
	}
	ExpectWithOffset(2, err).ToNot(HaveOccurred())
	input := &openapi3filter.ResponseValidationInput{
		RequestValidationInput: &openapi3filter.RequestValidationInput{
			Request: req, PathParams: pathParams, Route: route,
		},
		Status:  w.Code,
		Header:  w.Header(),
		Body:    io.NopCloser(bytes.NewReader(w.Body.Bytes())),
		Options: &openapi3filter.Options{IncludeResponseStatus: true},
	}
	ExpectWithOffset(2, openapi3filter.ValidateResponse(req.Context(), input)).To(Succeed(),
		"response for %s %s does not conform to the spec", req.Method, req.URL.Path)
}
