package domain

import "errors"

var (
	ErrTenantIDRequired       = errors.New("tenant ID is required")
	ErrConversationIDRequired = errors.New("conversation ID is required")
	ErrContactIDRequired      = errors.New("contact ID is required")
	ErrFileNameRequired       = errors.New("file name is required")
	ErrFileNameTooLong        = errors.New("file name exceeds maximum length")
	ErrStorageKeyRequired     = errors.New("storage key is required")
	ErrMimeTypeTooLong        = errors.New("mime type exceeds maximum length")
	ErrInvalidMediaType       = errors.New("invalid media type")
	ErrInvalidDirection       = errors.New("invalid direction")
	ErrNegativeSize           = errors.New("size must be non-negative")
	ErrFileNotFound           = errors.New("file not found")
	ErrFileContentUnavailable = errors.New("file content unavailable")
	ErrFileTooLarge           = errors.New("file exceeds maximum allowed size")
	ErrInvalidDateRange       = errors.New("invalid date range: from is after to")
)
