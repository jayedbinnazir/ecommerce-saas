package server

import (
	"github.com/gin-gonic/gin"

	authroutes "github.com/jayedbinnazir/golang-saas.git/internal/auth/routes"
	membershiproutes "github.com/jayedbinnazir/golang-saas.git/internal/membership/routes"
	"github.com/jayedbinnazir/golang-saas.git/internal/paymentclient"
	permissionroutes "github.com/jayedbinnazir/golang-saas.git/internal/permission/routes"
	roleroutes "github.com/jayedbinnazir/golang-saas.git/internal/role/routes"
	tenantdomain "github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
	tenantrepo "github.com/jayedbinnazir/golang-saas.git/internal/tenant/repository"
	tenantroutes "github.com/jayedbinnazir/golang-saas.git/internal/tenant/routes"
	userroutes "github.com/jayedbinnazir/golang-saas.git/internal/user/routes"
)

// registerModules builds every feature module and mounts its routes on api.
// The build order follows the dependency arrows between modules.
func (s *Server) registerModules(api *gin.RouterGroup) error {
	// 1. auth — provides the Authenticated / SuperAdmin guards everyone else needs.
	auth, err := authroutes.Build(s.db, s.rdb, s.cfg)
	if err != nil {
		return err
	}
	guards := auth.Guard.Guards()

	// 2. permission — grant checks are reused by the membership guard.
	permissionSvc, grantSvc := permissionroutes.Build(s.db)

	// 3. subscription gate — payment-service owns subscriptions; this client
	//    implements the tenant module's SubscriptionGuard.
	var subscriptionGuard tenantdomain.SubscriptionGuard = paymentclient.NewClient(
		s.cfg.Services.PaymentServiceURL, s.cfg.Services.PaymentInternalKey)

	// 4. membership — implements the tenant module's OwnerEnroller.
	tenantRepo := tenantrepo.New(s.db)
	membershipSvc, membershipGuard := membershiproutes.Build(s.db, tenantRepo, grantSvc)

	// 5. tenant — needs the subscription gate + membership.
	tenantSvc := tenantroutes.Build(s.db, subscriptionGuard, membershipSvc)

	// ---- mount routes ----
	authroutes.Register(api, auth, guards)
	userroutes.Register(api, s.db, guards, s.cfg.InternalKey)
	roleroutes.Register(api, s.db, guards)
	permissionroutes.Register(api, permissionSvc, grantSvc, guards)
	tenantroutes.Register(api, tenantSvc, guards)
	membershiproutes.Register(api, membershipSvc, membershipGuard, guards)

	return nil
}
