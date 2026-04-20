package domain

import "errors"

var (
	ErrForbidden      = errors.New("access forbidden")
	ErrTenantRequired = errors.New("tenant id is required")
	ErrUserRequired   = errors.New("user id is required")
)
