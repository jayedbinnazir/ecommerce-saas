package domain

import "errors"

var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrRoleAlreadyExists = errors.New("role already exists")
	ErrInvalidRoleName   = errors.New("invalid role name")
	ErrRoleInUse         = errors.New("role is still assigned to one or more members")
	ErrNoUpdateFields    = errors.New("no fields to update")
)
