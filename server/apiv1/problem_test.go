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
			func(err error, status int, code ProblemCode) {
				writeProblem(w, r, err)
				Expect(w.Code).To(Equal(status))
				Expect(w.Header().Get("Content-Type")).To(Equal(problemContentType))
				p := decodeProblem(w)
				Expect(p.Status).To(Equal(status))
				Expect(p.Code).To(Equal(code))
				Expect(p.Title).To(Equal(http.StatusText(status)))
				Expect(p.Type).To(BeNil())
				Expect(w.Body.String()).ToNot(ContainSubstring(`"type"`))
			},
			Entry("not found", model.ErrNotFound, http.StatusNotFound, ProblemCodeNotFound),
			Entry("not authorized", model.ErrNotAuthorized, http.StatusForbidden, ProblemCodeForbidden),
			Entry("invalid auth", model.ErrInvalidAuth, http.StatusUnauthorized, ProblemCodeUnauthorized),
			Entry("expired", model.ErrExpired, http.StatusUnauthorized, ProblemCodeUnauthorized),
			Entry("validation", model.ErrValidation, http.StatusBadRequest, ProblemCodeValidation),
			Entry("not available", model.ErrNotAvailable, http.StatusServiceUnavailable, ProblemCodeUnavailable),
			Entry("unknown", errors.New("boom"), http.StatusInternalServerError, ProblemCodeInternal),
		)

		DescribeTable("keeps the wrapping context as detail for client errors",
			func(err error) {
				writeProblem(w, r, err)
				p := decodeProblem(w)
				Expect(p.Status).To(Equal(http.StatusNotFound))
				Expect(p.Detail).ToNot(BeNil())
				Expect(*p.Detail).To(ContainSubstring("album 123"))
			},
			Entry("fmt.Errorf %w", fmt.Errorf("album 123: %w", model.ErrNotFound)),
			Entry("errors.Join", errors.Join(errors.New("album 123"), model.ErrNotFound)),
		)

		It("hides details for internal errors", func() {
			writeProblem(w, r, errors.New("db password is hunter2"))
			p := decodeProblem(w)
			Expect(p.Detail).To(BeNil())
		})
	})

	Describe("writeProblemStatus", func() {
		It("writes field errors only when provided", func() {
			writeProblemStatus(w, r, http.StatusBadRequest, ProblemCodeValidation, "bad input",
				ValidationError{Field: "limit", Message: "must be <= 2000"})
			p := decodeProblem(w)
			Expect(p.Errors).ToNot(BeNil())
			Expect(*p.Errors).To(HaveLen(1))
			Expect((*p.Errors)[0].Field).To(Equal("limit"))
		})

		It("omits detail when empty", func() {
			writeProblemStatus(w, r, http.StatusMethodNotAllowed, ProblemCodeMethodNotAllowed, "")
			Expect(w.Body.String()).ToNot(ContainSubstring(`"detail"`))
			Expect(w.Body.String()).ToNot(ContainSubstring(`"errors"`))
		})
	})

	Describe("bindingErrorHandler", func() {
		DescribeTable("maps parameter binding errors to a validation problem with the field",
			func(err error, field, message string) {
				bindingErrorHandler(w, r, err)
				Expect(w.Code).To(Equal(http.StatusBadRequest))
				p := decodeProblem(w)
				Expect(p.Code).To(Equal(ProblemCodeValidation))
				Expect(p.Errors).ToNot(BeNil())
				Expect(*p.Errors).To(HaveLen(1))
				Expect((*p.Errors)[0].Field).To(Equal(field))
				Expect((*p.Errors)[0].Message).To(ContainSubstring(message))
			},
			Entry("required", &RequiredParamError{ParamName: "limit"}, "limit", "is required"),
			Entry("invalid format", &InvalidParamFormatError{ParamName: "offset", Err: errors.New("not a number")}, "offset", "not a number"),
			Entry("too many values", &TooManyValuesForParamError{ParamName: "sort", Count: 2}, "sort", "single value"),
			Entry("unmarshaling", &UnmarshalingParamError{ParamName: "ids", Err: errors.New("bad json")}, "ids", "bad json"),
		)

		It("still returns a validation problem for unknown binding errors", func() {
			bindingErrorHandler(w, r, errors.New("weird"))
			p := decodeProblem(w)
			Expect(p.Status).To(Equal(http.StatusBadRequest))
			Expect(p.Code).To(Equal(ProblemCodeValidation))
			Expect(p.Errors).To(BeNil())
		})
	})
})
