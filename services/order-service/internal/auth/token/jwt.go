// Package token verifies the access JWT issued by the user-management service.
// It shares only the HMAC secret; it never mints tokens.
package token

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims mirrors the subset of user-management's access-token claims we need.
type Claims struct {
	jwt.RegisteredClaims
	SessionID  string `json:"sid"`
	SuperAdmin bool   `json:"sadm"`
}

func (c *Claims) UserID() (uuid.UUID, error)  { return uuid.Parse(c.Subject) }
func (c *Claims) Session() (uuid.UUID, error) { return uuid.Parse(c.SessionID) }

type Verifier struct {
	secret []byte
}

func NewVerifier(secret string) *Verifier {
	return &Verifier{secret: []byte(secret)}
}

// Verify parses and validates a token string (signature + expiry).
func (v *Verifier) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return v.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}
