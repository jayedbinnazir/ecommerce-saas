// Package services holds the membership-module application logic: adding/removing
// tenant members, changing their role, and per-member permission grants.
package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	membershipdomain "github.com/jayedbinnazir/golang-saas.git/internal/membership/domain"
	membershiprepo "github.com/jayedbinnazir/golang-saas.git/internal/membership/repository"
	permservices "github.com/jayedbinnazir/golang-saas.git/internal/permission/services"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
	roledomain "github.com/jayedbinnazir/golang-saas.git/internal/role/domain"
	rolerepo "github.com/jayedbinnazir/golang-saas.git/internal/role/repository"
	tenantdomain "github.com/jayedbinnazir/golang-saas.git/internal/tenant/domain"
)

type Service struct {
	db      *sql.DB
	tenants tenantdomain.Repository
	grants  *permservices.GrantService
}

func New(db *sql.DB, tenants tenantdomain.Repository, grants *permservices.GrantService) *Service {
	return &Service{db: db, tenants: tenants, grants: grants}
}

// compile-time check that Service satisfies the tenant module's enroller port.
var _ tenantdomain.OwnerEnroller = (*Service)(nil)

// EnrollOwnerAsAdmin is called from inside the tenant-creation transaction.
func (s *Service) EnrollOwnerAsAdmin(ctx context.Context, exec platform.DBTX, tenantID, userID uuid.UUID) error {
	adminRole, err := rolerepo.NewRoleRepository(exec).GetByName(ctx, roledomain.Admin)
	if err != nil {
		return err
	}
	return membershiprepo.New(exec).Create(ctx, &membershipdomain.Membership{
		TenantID: tenantID,
		UserID:   userID,
		RoleID:   adminRole.ID,
	})
}

// Resolve returns the caller's membership view for a tenant (used by the guard).
func (s *Service) Resolve(ctx context.Context, tenantID, userID uuid.UUID) (*membershipdomain.View, error) {
	return membershiprepo.New(s.db).ResolveView(ctx, tenantID, userID)
}

func (s *Service) ListMembers(ctx context.Context, tenantID uuid.UUID) ([]membershipdomain.View, error) {
	return membershiprepo.New(s.db).ListByTenant(ctx, tenantID)
}

func (s *Service) ListForUser(ctx context.Context, userID uuid.UUID) ([]membershipdomain.View, error) {
	return membershiprepo.New(s.db).ListByUser(ctx, userID)
}

// AddMember lets a tenant ADMIN attach a user as MANAGER or CUSTOMER.
func (s *Service) AddMember(ctx context.Context, tenantID, userID uuid.UUID, roleName roledomain.RoleName) (*membershipdomain.Membership, error) {
	if roleName != roledomain.Manager && roleName != roledomain.Customer {
		return nil, membershipdomain.ErrRoleNotAssignable
	}

	role, err := rolerepo.NewRoleRepository(s.db).GetByName(ctx, roleName)
	if err != nil {
		return nil, err
	}

	m := &membershipdomain.Membership{TenantID: tenantID, UserID: userID, RoleID: role.ID}
	if err := membershiprepo.New(s.db).Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// ChangeRole updates a member's role. The tenant owner's membership is locked.
func (s *Service) ChangeRole(ctx context.Context, tenantID, userID uuid.UUID, roleName roledomain.RoleName) (*membershipdomain.Membership, error) {
	if roleName != roledomain.Manager && roleName != roledomain.Customer {
		return nil, membershipdomain.ErrRoleNotAssignable
	}
	if err := s.guardNotOwner(ctx, tenantID, userID); err != nil {
		return nil, err
	}

	repo := membershiprepo.New(s.db)
	current, err := repo.GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	role, err := rolerepo.NewRoleRepository(s.db).GetByName(ctx, roleName)
	if err != nil {
		return nil, err
	}
	return repo.UpdateRole(ctx, tenantID, current.ID, role.ID)
}

// RemoveMember detaches a member (never the owner).
func (s *Service) RemoveMember(ctx context.Context, tenantID, userID uuid.UUID) error {
	if err := s.guardNotOwner(ctx, tenantID, userID); err != nil {
		return err
	}
	repo := membershiprepo.New(s.db)
	m, err := repo.GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	return repo.Delete(ctx, tenantID, m.ID)
}

// ---- per-member permission grants ----

// HasMembershipPermission reports whether the membership's effective permission
// set contains perm.
func (s *Service) HasMembershipPermission(ctx context.Context, membershipID uuid.UUID, perm string) (bool, error) {
	return s.grants.HasPermission(ctx, membershipID, perm)
}

func (s *Service) ListMemberPermissions(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	m, err := membershiprepo.New(s.db).GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	return s.grants.EffectiveForMembership(ctx, m.ID)
}

func (s *Service) GrantMemberPermission(ctx context.Context, tenantID, userID, permissionID uuid.UUID) error {
	m, err := membershiprepo.New(s.db).GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	return s.grants.AssignToMembership(ctx, m.ID, permissionID)
}

func (s *Service) RevokeMemberPermission(ctx context.Context, tenantID, userID, permissionID uuid.UUID) error {
	m, err := membershiprepo.New(s.db).GetByTenantAndUser(ctx, tenantID, userID)
	if err != nil {
		return err
	}
	return s.grants.RevokeFromMembership(ctx, m.ID, permissionID)
}

func (s *Service) guardNotOwner(ctx context.Context, tenantID, userID uuid.UUID) error {
	tenant, err := s.tenants.GetByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant.OwnerUserID == userID {
		return membershipdomain.ErrCannotModifyOwner
	}
	return nil
}
