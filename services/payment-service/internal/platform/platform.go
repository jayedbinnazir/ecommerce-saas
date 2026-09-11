// Package platform holds cross-module primitives shared by every feature module
// (service identity, pagination, database helpers, the authenticated caller).
// It must not import any feature module.
package platform

// ServiceName identifies this microservice in logs and error responses.
const ServiceName = "payment-service"

// Page holds simple limit/offset pagination parameters.
type Page struct {
	Limit  int
	Offset int
}
