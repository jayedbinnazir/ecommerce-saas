package httpx

import "github.com/gin-gonic/gin"

// Guards bundles the app-wide auth middleware built by the composition root and
// handed to each module's route registrar, so feature modules never import the
// auth package directly.
type Guards struct {
	// Authenticated rejects the request (401) unless a valid session cookie is
	// present, and puts the platform.Principal on the request context.
	Authenticated gin.HandlerFunc
	// SuperAdmin runs after Authenticated and rejects (403) non-platform-admins.
	SuperAdmin gin.HandlerFunc
}
