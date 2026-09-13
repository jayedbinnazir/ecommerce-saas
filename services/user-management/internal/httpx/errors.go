package httpx

import (
	"database/sql"
	"errors"
	"net/http"

	authdomain "github.com/jayedbinnazir/golang-saas.git/internal/auth/domain"
	membershipdomain "github.com/jayedbinnazir/golang-saas.git/internal/membership/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/paymentclient"
	permissiondomain "github.com/jayedbinnazir/golang-saas.git/internal/permission/domain"
	roledomain "github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
	tenantdomain "github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
	userdomain "github.com/jayedbinnazir/golang-saas.git/internal/user/domain"
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
	cause   error        // wrapped cause, never serialized
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.Message + ": " + e.cause.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

// Wrap attaches an underlying cause for logging without exposing it to clients.
func (e *Error) Wrap(cause error) *Error {
	copyErr := *e
	copyErr.cause = cause
	return &copyErr
}

// WithFields attaches per-field validation messages.
func (e *Error) WithFields(fields []FieldError) *Error {
	copyErr := *e
	copyErr.Fields = fields
	return &copyErr
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

// notFoundErrors, conflictErrors, etc. list the domain sentinels that map to a
// given HTTP status. Adding a new domain error is a one-line change here.
var (
	notFoundErrors = []error{
		sql.ErrNoRows,
		userdomain.ErrUserNotFound, userdomain.ErrAddressNotFound,
		roledomain.ErrRoleNotFound,
		permissiondomain.ErrPermissionNotFound, permissiondomain.ErrGrantNotFound,
		tenantdomain.ErrTenantNotFound,
		membershipdomain.ErrMembershipNotFound,
		authdomain.ErrIdentityNotFound, authdomain.ErrRefreshNotFound,
	}

	conflictErrors = []error{
		userdomain.ErrUserAlreadyExists,
		roledomain.ErrRoleAlreadyExists,
		permissiondomain.ErrPermissionAlreadyExists, permissiondomain.ErrGrantAlreadyExists,
		tenantdomain.ErrSlugTaken,
		membershipdomain.ErrAlreadyMember,
	}

	validationErrors = []error{
		userdomain.ErrInvalidEmail, userdomain.ErrEmptyUserName,
		userdomain.ErrNoUpdateFields, userdomain.ErrAddressNotOwned,
		roledomain.ErrInvalidRoleName, roledomain.ErrRoleInUse, roledomain.ErrNoUpdateFields,
		permissiondomain.ErrInvalidPermissionName, permissiondomain.ErrGrantReferenceInvalid,
		permissiondomain.ErrNoUpdateFields,
		tenantdomain.ErrInvalidSlug, tenantdomain.ErrInvalidStatus,
		tenantdomain.ErrNoUpdateFields, tenantdomain.ErrOwnerRequired,
		membershipdomain.ErrRefInvalid, membershipdomain.ErrRoleNotAssignable,
		membershipdomain.ErrCannotModifyOwner,
		paymentclient.ErrSubscriptionRequired,
		authdomain.ErrPasswordReused,
	}

	forbiddenErrors = []error{
		tenantdomain.ErrForbidden,
		membershipdomain.ErrForbidden,
	}

	upstreamErrors = []error{
		paymentclient.ErrUpstream,
	}

	unauthorizedErrors = []error{
		userdomain.ErrInvalidCredentials,
		authdomain.ErrRefreshInvalid, authdomain.ErrSessionExpired,
		authdomain.ErrOAuthStateInvalid, authdomain.ErrEmailNotVerified,
		authdomain.ErrCurrentPasswordInvalid,
	}

	badRequestErrors = []error{
		authdomain.ErrInvalidProvider, authdomain.ErrProviderNotConfig,
		authdomain.ErrOAuthExchange, authdomain.ErrResetTokenInvalid,
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
	case matchesAny(err, validationErrors):
		return Validation(cleanMessage(err)).Wrap(err)
	case matchesAny(err, forbiddenErrors):
		return Forbidden(cleanMessage(err)).Wrap(err)
	case matchesAny(err, unauthorizedErrors):
		return Unauthorized(cleanMessage(err)).Wrap(err)
	case matchesAny(err, badRequestErrors):
		return BadRequest(cleanMessage(err)).Wrap(err)
	case matchesAny(err, upstreamErrors):
		return Upstream(cleanMessage(err)).Wrap(err)
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
