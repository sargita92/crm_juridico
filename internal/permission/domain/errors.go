package domain

import "errors"

var (
	ErrGroupNotFound       = errors.New("permission group not found")
	ErrGroupNameRequired   = errors.New("group name is required")
	ErrGroupNameTooLong    = errors.New("group name exceeds 100 characters")
	ErrGroupDescTooLong    = errors.New("group description exceeds 500 characters")
	ErrTenantIDRequired    = errors.New("tenant ID is required")
	ErrUserIDRequired      = errors.New("user ID is required")
	ErrGroupIDRequired     = errors.New("group ID is required")
	ErrUserAlreadyInGroup  = errors.New("user is already in this group")
	ErrResourceRequired    = errors.New("resource is required")
	ErrActionRequired      = errors.New("action is required")
	ErrInvalidResource     = errors.New("invalid resource")
	ErrInvalidAction       = errors.New("invalid action for resource")
	ErrPermissionXOR       = errors.New("permission must have either group_id or user_id, not both")
	ErrPermissionNotFound  = errors.New("permission not found")
	ErrFunnelIDRequired    = errors.New("funnel ID is required")
	ErrViewProfileNotFound = errors.New("view profile not found")
	ErrGroupFunnelNotFound = errors.New("group-funnel association not found")
)
