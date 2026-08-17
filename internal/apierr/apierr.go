package apierr

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Name       string `json:"name"`
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode"`
}

func (e *Error) Error() string { return e.Message }

func New(status int, name, message string) *Error {
	return &Error{Name: name, Message: message, StatusCode: status}
}

func Write(w http.ResponseWriter, err error) {
	ae, ok := err.(*Error)
	if !ok {
		ae = New(http.StatusInternalServerError, "internal_error", err.Error())
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(ae.StatusCode)
	_ = json.NewEncoder(w).Encode(ae)
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

var (
	Unauthorized     = New(http.StatusUnauthorized, "missing_api_key", "Missing API key. Pass it via the Authorization header as Bearer ra_…")
	Forbidden        = New(http.StatusForbidden, "invalid_api_key", "API key is invalid")
	NotFound         = New(http.StatusNotFound, "not_found", "Resource not found")
	RateLimited      = New(http.StatusTooManyRequests, "rate_limit_exceeded", "Too many requests. Please try again later.")
	Validation       = func(msg string) *Error { return New(http.StatusBadRequest, "validation_error", msg) }
	QuotaExceeded    = New(http.StatusTooManyRequests, "quota_exceeded", "Monthly sending quota exceeded. Sending is paused.")
	DomainNotVerified = New(http.StatusForbidden, "domain_not_verified", "The from domain is not verified")
)
