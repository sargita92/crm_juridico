package domain

import "errors"

var (
	ErrAutomationNotFound    = errors.New("automation not found")
	ErrTenantIDRequired      = errors.New("tenant ID is required")
	ErrFunnelIDRequired      = errors.New("funnel ID is required")
	ErrInvalidType           = errors.New("invalid automation type")
	ErrInvalidConfig         = errors.New("invalid automation config")
	ErrRateLimitNotFound     = errors.New("rate limit counter not found")
	ErrExecutionLogNotFound  = errors.New("execution log not found")
)
