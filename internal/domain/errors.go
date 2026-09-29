package domain

import (
	"errors"
	"net/http"
)

type AppError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string { return e.Message }

func NewAppError(status int, code, msg string) *AppError {
	return &AppError{Status: status, Code: code, Message: msg}
}

var (
	ErrNotFound          = NewAppError(http.StatusNotFound, "not_found", "not found")
	ErrGone              = NewAppError(http.StatusGone, "gone", "withdrawn")
	ErrUnauthorized      = NewAppError(http.StatusUnauthorized, "unauthorized", "unauthorized")
	ErrForbidden         = NewAppError(http.StatusForbidden, "forbidden", "forbidden")
	ErrBadRequest        = NewAppError(http.StatusBadRequest, "validation", "bad request")
	ErrProfileIncomplete = NewAppError(http.StatusForbidden, "profile_incomplete", "profile incomplete")
	ErrUnavailable       = NewAppError(http.StatusServiceUnavailable, "internal", "identity service unavailable")
	ErrMediaUnavailable  = NewAppError(http.StatusServiceUnavailable, "media_unavailable", "object storage is not configured")
	ErrConflict          = NewAppError(http.StatusConflict, "conflict", "conflict")
	ErrConsentRequired   = NewAppError(http.StatusForbidden, "consent_required", "activity tracking not granted")
)

func IsAppError(err error) (*AppError, bool) {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
