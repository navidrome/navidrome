package apiv1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

const problemContentType = "application/problem+json"

func writeProblem(w http.ResponseWriter, r *http.Request, err error) {
	status, code := classifyError(err)
	detail := err.Error()
	if status == http.StatusInternalServerError {
		log.Error(r.Context(), "API v1: unexpected error", "path", r.URL.Path, err)
		detail = ""
	}
	writeProblemStatus(w, r, status, code, detail)
}

func classifyError(err error) (int, ProblemCode) {
	switch {
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
	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(p); err != nil {
		log.Warn(r.Context(), "API v1: could not write problem response", err)
	}
}

func bindingErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var fieldErrors []ValidationError
	var required *RequiredParamError
	var invalid *InvalidParamFormatError
	var tooMany *TooManyValuesForParamError
	var unmarshal *UnmarshalingParamError
	switch {
	case errors.As(err, &required):
		fieldErrors = append(fieldErrors, ValidationError{Field: required.ParamName, Message: "is required"})
	case errors.As(err, &invalid):
		fieldErrors = append(fieldErrors, ValidationError{Field: invalid.ParamName, Message: invalid.Err.Error()})
	case errors.As(err, &tooMany):
		fieldErrors = append(fieldErrors, ValidationError{Field: tooMany.ParamName, Message: "expected a single value"})
	case errors.As(err, &unmarshal):
		fieldErrors = append(fieldErrors, ValidationError{Field: unmarshal.ParamName, Message: unmarshal.Err.Error()})
	}
	writeProblemStatus(w, r, http.StatusBadRequest, ProblemCodeValidation, err.Error(), fieldErrors...)
}
