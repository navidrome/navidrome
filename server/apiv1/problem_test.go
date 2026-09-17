package apiv1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func decodeProblem(w *httptest.ResponseRecorder) Problem {
	var p Problem
	ExpectWithOffset(1, json.Unmarshal(w.Body.Bytes(), &p)).To(Succeed())
	return p
}

var _ = Describe("problem", func() {
	var w *httptest.ResponseRecorder
	var r *http.Request

	BeforeEach(func() {
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, "/api/v1/server", nil)
	})

	Describe("writeProblem", func() {
		DescribeTable("maps domain errors to status and code",
			func(err error, status int, code string) {
				writeProblem(w, r, err)
				Expect(w.Code).To(Equal(status))
				Expect(w.Header().Get("Content-Type")).To(Equal(problemContentType))
				p := decodeProblem(w)
				Expect(p.Status).To(Equal(status))
				Expect(p.Code).To(Equal(code))
				Expect(p.Title).To(Equal(http.StatusText(status)))
				Expect(p.Type).To(Equal("about:blank"))
			},
			Entry("not found", model.ErrNotFound, http.StatusNotFound, "not_found"),
			Entry("not authorized", model.ErrNotAuthorized, http.StatusForbidden, "forbidden"),
			Entry("invalid auth", model.ErrInvalidAuth, http.StatusUnauthorized, "unauthorized"),
			Entry("expired", model.ErrExpired, http.StatusUnauthorized, "unauthorized"),
			Entry("validation", model.ErrValidation, http.StatusBadRequest, "validation"),
			Entry("not available", model.ErrNotAvailable, http.StatusServiceUnavailable, "unavailable"),
			Entry("unknown", errors.New("boom"), http.StatusInternalServerError, "internal"),
		)

		It("includes the error message as detail for client errors", func() {
			writeProblem(w, r, fmt.Errorf("album 123: %w", model.ErrNotFound))
			p := decodeProblem(w)
			Expect(p.Status).To(Equal(http.StatusNotFound))
			Expect(p.Detail).ToNot(BeNil())
			Expect(*p.Detail).To(ContainSubstring("album 123"))
		})

		It("unwraps wrapped domain errors", func() {
			writeProblem(w, r, errors.Join(errors.New("loading album"), model.ErrNotFound))
			p := decodeProblem(w)
			Expect(p.Status).To(Equal(http.StatusNotFound))
			Expect(*p.Detail).To(ContainSubstring("loading album"))
		})

		It("hides details for internal errors", func() {
			writeProblem(w, r, errors.New("db password is hunter2"))
			p := decodeProblem(w)
			Expect(p.Detail).To(BeNil())
			Expect(p.Errors).To(BeNil())
		})
	})

	Describe("writeProblemStatus", func() {
		It("writes field errors only when provided", func() {
			writeProblemStatus(w, r, http.StatusBadRequest, "validation", "bad input",
				ValidationError{Field: "limit", Message: "must be <= 500"})
			p := decodeProblem(w)
			Expect(p.Errors).ToNot(BeNil())
			Expect(*p.Errors).To(HaveLen(1))
			Expect((*p.Errors)[0].Field).To(Equal("limit"))
		})

		It("omits detail when empty", func() {
			writeProblemStatus(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "")
			Expect(w.Body.String()).ToNot(ContainSubstring(`"detail"`))
			Expect(w.Body.String()).ToNot(ContainSubstring(`"errors"`))
		})
	})

	Describe("bindingErrorHandler", func() {
		It("maps a required-param error to a validation problem with the field", func() {
			bindingErrorHandler(w, r, &RequiredParamError{ParamName: "limit"})
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			p := decodeProblem(w)
			Expect(p.Code).To(Equal("validation"))
			Expect((*p.Errors)[0].Field).To(Equal("limit"))
		})

		It("maps an invalid-format error to a validation problem with the field", func() {
			bindingErrorHandler(w, r, &InvalidParamFormatError{ParamName: "offset", Err: errors.New("not a number")})
			p := decodeProblem(w)
			Expect(p.Code).To(Equal("validation"))
			Expect((*p.Errors)[0].Field).To(Equal("offset"))
			Expect((*p.Errors)[0].Message).To(ContainSubstring("not a number"))
		})

		It("still returns a validation problem for unknown binding errors", func() {
			bindingErrorHandler(w, r, errors.New("weird"))
			p := decodeProblem(w)
			Expect(p.Status).To(Equal(http.StatusBadRequest))
			Expect(p.Code).To(Equal("validation"))
		})
	})
})
