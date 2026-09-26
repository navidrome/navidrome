package apiv1

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/navidrome/navidrome/core/apiauth"
	"github.com/navidrome/navidrome/core/auth"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const problemContentType = "application/problem+json"

type clientError struct {
	err    error
	detail string
}

func (e *clientError) Error() string { return e.detail }
func (e *clientError) Unwrap() error { return e.err }

// ClientError marks detail as safe to show clients; err still decides the status and code.
func ClientError(err error, detail string) error {
	return &clientError{err: err, detail: detail}
}

// scopeError names the scope an operation requires, for the insufficient_scope challenge.
type scopeError struct {
	scope string
}

func (e *scopeError) Error() string { return apiauth.ErrInsufficientScope.Error() }
func (e *scopeError) Unwrap() error { return apiauth.ErrInsufficientScope }

const tooLargeDetail = "request body too large"

func tooLarge(err error) bool {
	return errors.As(err, new(*http.MaxBytesError))
}

type fieldErrors struct {
	fields []ValidationError
}

func (e *fieldErrors) Error() string { return "validation failed" }
func (e *fieldErrors) Unwrap() error { return model.ErrValidation }

func validationFailed(fields ...ValidationError) error {
	return &fieldErrors{fields: fields}
}

func writeProblem(w http.ResponseWriter, r *http.Request, err error) {
	status, code := classifyError(err)
	if status == http.StatusInternalServerError {
		log.Error(r.Context(), "API v1: unexpected error", "path", r.URL.Path, err)
		writeProblemStatus(w, r, status, code, "")
		return
	}
	log.Debug(r.Context(), "API v1: request failed", "path", r.URL.Path, "status", status, "code", code, err)
	if code == ProblemCodeInsufficientScope {
		w.Header().Set("WWW-Authenticate", scopeChallenge(err))
	}
	var detail string
	var ce *clientError
	if errors.As(err, &ce) {
		detail = ce.detail
	}
	var fe *fieldErrors
	if errors.As(err, &fe) {
		writeProblemStatus(w, r, status, code, detail, fe.fields...)
		return
	}
	writeProblemStatus(w, r, status, code, detail)
}

func scopeChallenge(err error) string {
	challenge := `Bearer error="insufficient_scope"`
	var se *scopeError
	if errors.As(err, &se) && se.scope != "" {
		challenge += fmt.Sprintf(`, scope=%q`, se.scope)
	}
	return challenge
}

func classifyError(err error) (int, ProblemCode) {
	switch {
	case tooLarge(err):
		return http.StatusRequestEntityTooLarge, ProblemCodePayloadTooLarge
	case errors.Is(err, apiauth.ErrTokenExpired):
		return http.StatusUnauthorized, ProblemCodeTokenExpired
	case errors.Is(err, apiauth.ErrInsufficientScope):
		return http.StatusForbidden, ProblemCodeInsufficientScope
	case errors.Is(err, auth.ErrSetupComplete):
		return http.StatusConflict, ProblemCodeSetupComplete
	case errors.Is(err, apiauth.ErrPasswordManagedExternally):
		return http.StatusConflict, ProblemCodePasswordManagedExternally
	case errors.Is(err, model.ErrNotFound):
		return http.StatusNotFound, ProblemCodeNotFound
	case errors.Is(err, model.ErrNotAuthorized):
		return http.StatusForbidden, ProblemCodeForbidden
	case errors.Is(err, model.ErrInvalidAuth), errors.Is(err, model.ErrExpired):
		return http.StatusUnauthorized, ProblemCodeUnauthorized
	case errors.Is(err, model.ErrValidation):
		return http.StatusBadRequest, ProblemCodeValidation
	case errors.Is(err, model.ErrNotAvailable):
		return http.StatusServiceUnavailable, ProblemCodeUnavailable
	}
	return http.StatusInternalServerError, ProblemCodeInternal
}

func writeProblemStatus(w http.ResponseWriter, r *http.Request, status int, code ProblemCode, detail string, fieldErrors ...ValidationError) {
	p := Problem{Title: http.StatusText(status), Status: status, Code: code}
	if detail != "" {
		p.Detail = &detail
	}
	if len(fieldErrors) > 0 {
		p.Errors = &fieldErrors
	}
	if status == http.StatusInternalServerError {
		if ref := referenceIDFrom(r.Context()); ref != "" {
			p.ReferenceId = &ref
		}
	}
	// Every 401 carries a Bearer challenge; callers may set a more specific one first.
	if status == http.StatusUnauthorized && w.Header().Get("WWW-Authenticate") == "" {
		challenge := "Bearer"
		if _, sent := bearerToken(r); sent {
			challenge = `Bearer error="invalid_token"`
		}
		w.Header().Set("WWW-Authenticate", challenge)
	}
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Warn(r.Context(), "API v1: could not write problem response", err)
	}
}

func bindingErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var fieldErrs []ValidationError
	var required *RequiredParamError
	var invalid *InvalidParamFormatError
	var tooMany *TooManyValuesForParamError
	var unmarshal *UnmarshalingParamError
	switch {
	case errors.As(err, &required):
		fieldErrs = append(fieldErrs, ValidationError{Field: required.ParamName, Message: "is required"})
	case errors.As(err, &invalid):
		fieldErrs = append(fieldErrs, ValidationError{Field: invalid.ParamName, Message: "has an invalid value"})
	case errors.As(err, &tooMany):
		fieldErrs = append(fieldErrs, ValidationError{Field: tooMany.ParamName, Message: "expected a single value"})
	case errors.As(err, &unmarshal):
		fieldErrs = append(fieldErrs, ValidationError{Field: unmarshal.ParamName, Message: "has an invalid value"})
	}
	writeProblemStatus(w, r, http.StatusBadRequest, ProblemCodeValidation, "invalid request parameters", fieldErrs...)
}
