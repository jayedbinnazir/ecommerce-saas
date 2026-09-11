package platform

import (
	"context"

	"github.com/google/uuid"
)

// Principal is the authenticated caller, decoded from the access token issued by
// the user-management service. RawToken is kept so tenant-authorization checks
// can forward it to user-management.
type Principal struct {
	UserID       uuid.UUID
	SessionID    uuid.UUID
	IsSuperAdmin bool
	RawToken     string
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
