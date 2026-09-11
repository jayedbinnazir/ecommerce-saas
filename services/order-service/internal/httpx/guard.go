package httpx

import "github.com/gin-gonic/gin"

// Guards bundles the auth middleware built by the composition root and handed to
// each feature module's route registrar, so modules never import the auth or
// authz packages directly.
type Guards struct {
	// Authenticated rejects the request (401) unless a valid access token is
	// present, and puts the platform.Principal on the request context.
	Authenticated gin.HandlerFunc
	// SuperAdmin runs after Authenticated and rejects (403) non-platform-admins.
	SuperAdmin gin.HandlerFunc
	// TenantRole returns middleware that (after Authenticated) allows super
	// admins, or callers whose membership in the tenant named by tenantParam has
	// one of the given roles ("ADMIN", "MANAGER", ...). It resolves membership by
	// calling the user-management service.
	TenantRole func(tenantParam string, roles ...string) gin.HandlerFunc
}
