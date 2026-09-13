package domain

import "errors"

var (
	ErrIdentityNotFound  = errors.New("auth identity not found")
	ErrRefreshNotFound   = errors.New("refresh token not found")
	ErrRefreshInvalid    = errors.New("refresh token is invalid or expired")
	ErrSessionExpired    = errors.New("session expired, please sign in again")
	ErrInvalidProvider   = errors.New("unsupported auth provider")
	ErrProviderNotConfig = errors.New("this auth provider is not configured")
	ErrOAuthStateInvalid = errors.New("invalid or expired oauth state")
	ErrOAuthExchange     = errors.New("could not complete sign-in with the provider")
	ErrEmailNotVerified  = errors.New("the provider did not return a verified email")

	ErrCurrentPasswordInvalid = errors.New("current password is incorrect")
	ErrPasswordReused         = errors.New("new password must be different from the current password")
	// ErrResetTokenInvalid covers not-found, already-used and expired alike —
	// deliberately not distinguished so a caller can't use the error to probe
	// which case applies.
	ErrResetTokenInvalid = errors.New("invalid or expired reset token")
)
