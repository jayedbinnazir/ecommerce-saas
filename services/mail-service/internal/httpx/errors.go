package httpx

import (
	"database/sql"
	"errors"
	"net/http"

	maildomain "github.com/jayedbinnazir/mail-service/internal/mail/domain"
)

// Code is a stable, machine-readable error identifier returned to clients.
type Code string

const (
	CodeBadRequest   Code = "BAD_REQUEST"
	CodeValidation   Code = "VALIDATION_ERROR"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeForbidden    Code = "FORBIDDEN"
	CodeNotFound     Code = "NOT_FOUND"
	CodeConflict     Code = "CONFLICT"
	CodeRateLimited  Code = "RATE_LIMITED"
	CodeUpstream     Code = "UPSTREAM_ERROR"
	CodeInternal     Code = "INTERNAL_ERROR"
)

// FieldError explains why one request field failed validation.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error is the canonical application error.
type Error struct {
	Code    Code         `json:"code"`
	Message string       `json:"message"`
	Status  int          `json:"-"`
	Fields  []FieldError `json:"fields,omitempty"`
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

func (e *Error) Wrap(cause error) *Error {
	cp := *e
	cp.cause = cause
	return &cp
}

func (e *Error) WithFields(fields []FieldError) *Error {
	cp := *e
	cp.Fields = fields
	return &cp
}

func newError(code Code, status int, msg string) *Error {
	return &Error{Code: code, Status: status, Message: msg}
}

func BadRequest(msg string) *Error { return newError(CodeBadRequest, http.StatusBadRequest, msg) }
func Validation(msg string) *Error {
	return newError(CodeValidation, http.StatusUnprocessableEntity, msg)
}
func Unauthorized(msg string) *Error { return newError(CodeUnauthorized, http.StatusUnauthorized, msg) }
func Forbidden(msg string) *Error    { return newError(CodeForbidden, http.StatusForbidden, msg) }
func NotFound(msg string) *Error     { return newError(CodeNotFound, http.StatusNotFound, msg) }
func Conflict(msg string) *Error     { return newError(CodeConflict, http.StatusConflict, msg) }
func TooManyRequests(msg string) *Error {
	return newError(CodeRateLimited, http.StatusTooManyRequests, msg)
}
func Upstream(msg string) *Error { return newError(CodeUpstream, http.StatusBadGateway, msg) }
func Internal(msg string) *Error { return newError(CodeInternal, http.StatusInternalServerError, msg) }

var (
	validationErrors = []error{
		maildomain.ErrUnknownTemplate,
		maildomain.ErrInvalidRecipient,
	}
	upstreamErrors = []error{
		maildomain.ErrSendFailed,
	}
)

// Classify turns any error into a client-facing *Error with the right status.
func Classify(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	switch {
	case errors.Is(err, sql.ErrNoRows):
		return NotFound("resource not found").Wrap(err)
	case matchesAny(err, validationErrors):
		return Validation(err.Error()).Wrap(err)
	case matchesAny(err, upstreamErrors):
		return Upstream(err.Error()).Wrap(err)
	default:
		return Internal("internal server error").Wrap(err)
	}
}

func matchesAny(err error, list []error) bool {
	for _, target := range list {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
