package domain

import "errors"

var (
	// Contact
	ErrTenantIDRequired          = errors.New("tenant ID is required")
	ErrContactPhoneRequired      = errors.New("contact phone is required")
	ErrContactWhatsAppIDRequired = errors.New("contact WhatsApp ID is required")
	ErrContactNameTooLong        = errors.New("contact name exceeds maximum length")
	ErrContactNotFound           = errors.New("contact not found")

	// Conversation
	ErrContactIDRequired      = errors.New("contact ID is required")
	ErrConversationNotFound   = errors.New("conversation not found")
	ErrConversationIDRequired = errors.New("conversation ID is required")

	// Message
	ErrMessageContentRequired = errors.New("message content is required")
	ErrMessageContentTooLong  = errors.New("message content exceeds maximum length")
	ErrMessageNotFound        = errors.New("message not found")

	// Session
	ErrSessionNotFound      = errors.New("whatsapp session not found")
	ErrSessionAlreadyExists = errors.New("whatsapp session already exists for this tenant")
	ErrNotConnected         = errors.New("whatsapp is not connected")
)
