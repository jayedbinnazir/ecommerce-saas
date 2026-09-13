package domain

import "errors"

var (
	ErrMembershipNotFound = errors.New("membership not found")
	ErrAlreadyMember      = errors.New("user is already a member of this tenant")
	ErrRefInvalid         = errors.New("referenced tenant, user or role does not exist")
	ErrRoleNotAssignable  = errors.New("that role cannot be assigned to a tenant member")
	ErrCannotModifyOwner  = errors.New("the tenant owner's membership cannot be changed here")
	ErrForbidden          = errors.New("insufficient permissions for this tenant")
)
