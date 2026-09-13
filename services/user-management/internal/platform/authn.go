package platform

import (
	"context"

	"github.com/google/uuid"
)

// Principal is the authenticated caller, established by the auth middleware and
// consumed by every module's handlers/services for authorization decisions.
type Principal struct {
	UserID       uuid.UUID
	SessionID    uuid.UUID
	IsSuperAdmin bool
}

type principalKey struct{}

// WithPrincipal returns a copy of ctx carrying p.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// PrincipalFromContext returns the caller and whether one is present.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
