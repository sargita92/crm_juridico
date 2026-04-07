package domain

import "errors"

var (
	ErrTenantIDRequired     = errors.New("tenant ID is required")
	ErrUserIDRequired       = errors.New("user ID is required")
	ErrTitleRequired        = errors.New("notification title is required")
	ErrTitleTooLong         = errors.New("notification title exceeds 200 characters")
	ErrBodyTooLong          = errors.New("notification body exceeds 1000 characters")
	ErrNotificationNotFound = errors.New("notification not found")
	ErrPreferenceNotFound   = errors.New("notification preference not found")
	ErrInvalidChannel       = errors.New("invalid notification channel")
)
