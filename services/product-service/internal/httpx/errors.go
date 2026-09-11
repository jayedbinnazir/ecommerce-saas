package httpx

import (
	"database/sql"
	"errors"
	"net/http"

	attrdomain "github.com/jayedbinnazir/product-service/internal/attributes/domain"
	categorydomain "github.com/jayedbinnazir/product-service/internal/category/domain"
	productdomain "github.com/jayedbinnazir/product-service/internal/product/domain"
	reviewdomain "github.com/jayedbinnazir/product-service/internal/review/domain"
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

// Error is the canonical application error. Handlers push it (or any error) onto
// gin's error stack with c.Error(err); the ErrorHandler middleware renders it.
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

// Upstream means a dependency (e.g. user-management) failed or was unreachable.
func Upstream(msg string) *Error { return newError(CodeUpstream, http.StatusBadGateway, msg) }
func Internal(msg string) *Error { return newError(CodeInternal, http.StatusInternalServerError, msg) }

// domain sentinels grouped by the HTTP status they map to.
var (
	notFoundErrors = []error{
		sql.ErrNoRows,
		productdomain.ErrProductNotFound,
		productdomain.ErrVariantNotFound,
		productdomain.ErrImageNotFound,
		categorydomain.ErrCategoryNotFound,
		reviewdomain.ErrReviewNotFound,
		attrdomain.ErrAttributeNotFound,
		attrdomain.ErrValueNotFound,
		attrdomain.ErrCategoryNotFound,
	}

	forbiddenErrors = []error{
		reviewdomain.ErrNotAuthor,
	}

	conflictErrors = []error{
		productdomain.ErrSlugTaken,
		productdomain.ErrSkuTaken,
		productdomain.ErrImageAlreadyExists,
		categorydomain.ErrSlugTaken,
		attrdomain.ErrCodeTaken,
		attrdomain.ErrValueTaken,
	}

	validationErrors = []error{
		productdomain.ErrInvalidSlug,
		productdomain.ErrInvalidStatus,
		productdomain.ErrNoUpdateFields,
		productdomain.ErrActivateNeedsVariant,
		productdomain.ErrVariantNotInProduct,
		productdomain.ErrCategoryNotInTenant,
		productdomain.ErrAttributeNotForProduct,
		productdomain.ErrOptionValueInvalid,
		productdomain.ErrImageNotUploaded,
		productdomain.ErrImageKeyMismatch,
		productdomain.ErrUnsupportedImageType,
		categorydomain.ErrInvalidSlug,
		categorydomain.ErrNoUpdateFields,
		categorydomain.ErrParentCycle,
		reviewdomain.ErrInvalidRating,
		attrdomain.ErrInvalidRole,
		attrdomain.ErrInvalidCode,
		attrdomain.ErrNoUpdateFields,
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
	case matchesAny(err, notFoundErrors):
		return NotFound(cleanMessage(err)).Wrap(err)
	case matchesAny(err, conflictErrors):
		return Conflict(cleanMessage(err)).Wrap(err)
	case matchesAny(err, forbiddenErrors):
		return Forbidden(cleanMessage(err)).Wrap(err)
	case matchesAny(err, validationErrors):
		return Validation(cleanMessage(err)).Wrap(err)
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

func cleanMessage(err error) string {
	if errors.Is(err, sql.ErrNoRows) {
		return "resource not found"
	}
	return err.Error()
}
