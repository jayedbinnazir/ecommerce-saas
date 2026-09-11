// Package platform holds cross-module primitives shared by every feature module.
package platform

// ServiceName identifies this microservice in logs and error responses.
const ServiceName = "notification-service"

// Page holds simple limit/offset pagination parameters.
type Page struct {
	Limit  int
	Offset int
}
