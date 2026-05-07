package domain

import "errors"

var (
	ErrTenantIDRequired   = errors.New("tenant ID is required")
	ErrFunnelNotFound     = errors.New("funnel not found")
	ErrFunnelNameRequired = errors.New("funnel name is required")
	ErrFunnelNameTooLong  = errors.New("funnel name exceeds maximum length")
	ErrFunnelInactive     = errors.New("funnel is inactive")
	ErrFunnelIDRequired   = errors.New("funnel ID is required")

	ErrColumnNotFound     = errors.New("column not found")
	ErrColumnNameRequired = errors.New("column name is required")
	ErrColumnNameTooLong  = errors.New("column name exceeds maximum length")
	ErrColumnTypeInvalid  = errors.New("column type is invalid")
	ErrColumnHasLeads     = errors.New("column has leads and cannot be removed")
	ErrColumnLimitReached = errors.New("maximum number of columns reached")
	ErrColumnIDRequired   = errors.New("column ID is required")

	ErrLeadNotFound           = errors.New("lead not found")
	ErrLeadAlreadyExists      = errors.New("lead already exists for this contact")
	ErrContactIDRequired      = errors.New("contact ID is required")
	ErrConversationIDRequired = errors.New("conversation ID is required")

	// Lead notes
	ErrNoteContentRequired   = errors.New("note content is required")
	ErrNoteContentTooLong    = errors.New("note content exceeds 2000 characters")
	ErrNoteCreatedByRequired = errors.New("note created_by is required")
)
