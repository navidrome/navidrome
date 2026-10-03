package apiv1

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/core/auth"
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
	var ctx context.Context
	var w *httptest.ResponseRecorder
	var r *http.Request

	BeforeEach(func() {
		ctx = GinkgoT().Context()
		w = httptest.NewRecorder()
		r = httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/v1/server", nil)
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
			Entry("insufficient scope", &scopeError{scope: "read"}, http.StatusForbidden, ProblemCodeInsufficientScope),
			Entry("setup complete", auth.ErrSetupComplete, http.StatusConflict, ProblemCodeSetupComplete),
			Entry("password managed externally", apiauth.ErrPasswordManagedExternally, http.StatusConflict, ProblemCodePasswordManagedExternally),
			Entry("unknown", errors.New("boom"), http.StatusInternalServerError, ProblemCodeInternal),
		)

		It("shows detail only for errors marked as client-facing", func() {
			writeProblem(w, r, fmt.Errorf("album 123: %w", model.ErrNotFound))
			Expect(decodeProblem(w).Detail).To(BeNil())

			w = httptest.NewRecorder()
			writeProblem(w, r, ClientError(model.ErrNotFound, "album not found"))
			p := decodeProblem(w)
			Expect(p.Code).To(Equal(ProblemCodeNotFound))
			Expect(*p.Detail).To(Equal("album not found"))
		})

		It("writes field errors from validationFailed", func() {
			writeProblem(w, r, validationFailed(ValidationError{Field: "currentPassword", Message: "is incorrect"}))
			p := decodeProblem(w)
			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(p.Code).To(Equal(ProblemCodeValidation))
			Expect(*p.Errors).To(ConsistOf(ValidationError{Field: "currentPassword", Message: "is incorrect"}))
		})

		It("writes and logs a problem with debug logging on", func() {
			logs := captureLogs()
			writeProblem(w, r, model.ErrNotFound)
			Expect(w.Code).To(Equal(http.StatusNotFound))
			Expect(logs.String()).To(ContainSubstring("code=not_found"))
		})

		It("adds a Bearer challenge to every 401 unless one is already set", func() {
			writeProblem(w, r, model.ErrInvalidAuth)
			Expect(w.Header().Get("WWW-Authenticate")).To(Equal("Bearer"))

			w = httptest.NewRecorder()
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			writeProblem(w, r, model.ErrInvalidAuth)
			Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="invalid_token"`))
		})

		It("challenges a 401 with invalid_token when the request carried a bearer token", func() {
			r.Header.Set("Authorization", "Bearer tok")
			writeProblem(w, r, model.ErrInvalidAuth)
			Expect(w.Header().Get("WWW-Authenticate")).To(Equal(`Bearer error="invalid_token"`))

			w = httptest.NewRecorder()
			r.Header.Set("Authorization", "Basic dXNlcjpwdw==")
			writeProblem(w, r, model.ErrInvalidAuth)
			Expect(w.Header().Get("WWW-Authenticate")).To(Equal("Bearer"))
		})

		It("adds the request's referenceId to internal errors only", func() {
			r = r.WithContext(withReferenceID(r.Context(), "ref-123"))
			writeProblem(w, r, errors.New("boom"))
			Expect(*decodeProblem(w).ReferenceId).To(Equal("ref-123"))

			w = httptest.NewRecorder()
			writeProblem(w, r, model.ErrNotFound)
			Expect(decodeProblem(w).ReferenceId).To(BeNil())
		})

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
				Expect((*p.Errors)[0].Message).To(Equal(message))
			},
			Entry("required", &RequiredParamError{ParamName: "limit"}, "limit", "is required"),
			Entry("invalid format", &InvalidParamFormatError{ParamName: "offset", Err: errors.New(`parsing "abc": invalid syntax`)}, "offset", "has an invalid value"),
			Entry("too many values", &TooManyValuesForParamError{ParamName: "sort", Count: 2}, "sort", "expected a single value"),
			Entry("unmarshaling", &UnmarshalingParamError{ParamName: "ids", Err: errors.New("bad json")}, "ids", "has an invalid value"),
		)

		It("never echoes the submitted value", func() {
			bindingErrorHandler(w, r, &InvalidParamFormatError{ParamName: "offset", Err: errors.New(`parsing "hunter2": invalid syntax`)})
			Expect(w.Body.String()).ToNot(ContainSubstring("hunter2"))
		})

		It("still returns a validation problem for unknown binding errors", func() {
			bindingErrorHandler(w, r, errors.New("weird"))
			p := decodeProblem(w)
			Expect(p.Status).To(Equal(http.StatusBadRequest))
			Expect(p.Code).To(Equal(ProblemCodeValidation))
			Expect(p.Errors).To(BeNil())
		})
	})
})
