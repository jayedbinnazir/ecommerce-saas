package domain

import "errors"

var (
	ErrPermissionNotFound      = errors.New("permission not found")
	ErrPermissionAlreadyExists = errors.New("permission already exists")
	ErrInvalidPermissionName   = errors.New("invalid permission name")
	ErrGrantNotFound           = errors.New("grant not found")
	ErrGrantAlreadyExists      = errors.New("permission already granted")
	ErrGrantReferenceInvalid   = errors.New("referenced role, membership or permission does not exist")
	ErrNoUpdateFields          = errors.New("no fields to update")
)
