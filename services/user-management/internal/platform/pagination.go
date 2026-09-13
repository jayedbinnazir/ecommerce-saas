// Package platform holds cross-module primitives shared by every feature module
// (pagination, database helpers). It must not import any feature module.
package platform

// Page holds simple limit/offset pagination parameters.
type Page struct {
	Limit  int
	Offset int
}
