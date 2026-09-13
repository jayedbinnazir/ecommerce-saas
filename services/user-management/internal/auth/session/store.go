// Package session stores login sessions and short-lived OAuth state in Redis.
package session

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	authdomain "github.com/jayedbinnazir/golang-saas.git/internal/auth/domain"
)

const (
	sessionPrefix = "session:"
	statePrefix   = "oauth_state:"
	stateTTL      = 10 * time.Minute
)

// Session is the small record the auth middleware needs on every request.
type Session struct {
	UserID     uuid.UUID `json:"user_id"`
	SuperAdmin bool      `json:"super_admin"`
}

type Store struct {
	rdb *redis.Client
	ttl time.Duration // matches the refresh-token lifetime
}

func NewStore(rdb *redis.Client, ttl time.Duration) *Store {
	return &Store{rdb: rdb, ttl: ttl}
}

// Save writes (or refreshes) a session.
func (s *Store) Save(ctx context.Context, sessionID uuid.UUID, sess Session) error {
	blob, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, sessionPrefix+sessionID.String(), blob, s.ttl).Err()
}

// Get returns the session, or ErrSessionExpired if it is gone (logged out / TTL).
func (s *Store) Get(ctx context.Context, sessionID uuid.UUID) (Session, error) {
	blob, err := s.rdb.Get(ctx, sessionPrefix+sessionID.String()).Bytes()
	if errors.Is(err, redis.Nil) {
		return Session{}, authdomain.ErrSessionExpired
	}
	if err != nil {
		return Session{}, err
	}
	var sess Session
	if err := json.Unmarshal(blob, &sess); err != nil {
		return Session{}, err
	}
	return sess, nil
}

// Delete removes a session (logout).
func (s *Store) Delete(ctx context.Context, sessionID uuid.UUID) error {
	return s.rdb.Del(ctx, sessionPrefix+sessionID.String()).Err()
}

// SaveState stores the OAuth "state" value with the post-login redirect target.
func (s *Store) SaveState(ctx context.Context, state, redirectTo string) error {
	return s.rdb.Set(ctx, statePrefix+state, redirectTo, stateTTL).Err()
}

// ConsumeState atomically reads and deletes a state value. It returns
// ErrOAuthStateInvalid when the state is unknown or already used.
func (s *Store) ConsumeState(ctx context.Context, state string) (string, error) {
	redirectTo, err := s.rdb.GetDel(ctx, statePrefix+state).Result()
	if errors.Is(err, redis.Nil) {
		return "", authdomain.ErrOAuthStateInvalid
	}
	if err != nil {
		return "", err
	}
	return redirectTo, nil
}
