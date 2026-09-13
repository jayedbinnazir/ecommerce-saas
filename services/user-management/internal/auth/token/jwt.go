// Package token issues and verifies the short-lived access JWT.
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims is what we put inside the access token.
type Claims struct {
	jwt.RegisteredClaims
	SessionID  string `json:"sid"`
	SuperAdmin bool   `json:"sadm"`
}

// Manager signs and verifies access tokens with a single HMAC secret.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

// TTL is how long an issued access token stays valid.
func (m *Manager) TTL() time.Duration { return m.ttl }

// Issue returns a signed access token for the given user + session.
func (m *Manager) Issue(userID, sessionID uuid.UUID, superAdmin bool) (string, error) {
	now := time.Now().UTC()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
		SessionID:  sessionID.String(),
		SuperAdmin: superAdmin,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}

// Verify parses and validates a token string and returns its claims.
func (m *Manager) Verify(tokenString string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// UserID / SessionID parse the claim strings back to UUIDs.
func (c *Claims) UserID() (uuid.UUID, error)  { return uuid.Parse(c.Subject) }
func (c *Claims) Session() (uuid.UUID, error) { return uuid.Parse(c.SessionID) }
