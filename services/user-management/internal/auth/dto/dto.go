// Package dto holds the request/response payloads for the auth module.
package dto

import "time"

type RegisterRequest struct {
	Name     string  `json:"name" binding:"required,min=1,max=255"`
	Email    string  `json:"email" binding:"required,email"`
	Phone    *string `json:"phone" binding:"omitempty,max=32"`
	Password string  `json:"password" binding:"required,min=8,max=128"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=128"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=128"`
}

// Tokens is the pair issued on every successful sign-in. The handler puts these
// into httpOnly cookies; they are also returned in the body for API clients.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token lifetime, seconds
	accessTTL    time.Duration
	refreshTTL   time.Duration
}

func NewTokens(access, refresh string, accessTTL, refreshTTL time.Duration) *Tokens {
	return &Tokens{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(accessTTL.Seconds()),
		accessTTL:    accessTTL,
		refreshTTL:   refreshTTL,
	}
}

func (t *Tokens) AccessMaxAge() int  { return int(t.accessTTL.Seconds()) }
func (t *Tokens) RefreshMaxAge() int { return int(t.refreshTTL.Seconds()) }
