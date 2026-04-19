package domain

import (
	"path/filepath"
	"strings"
	"time"
)

type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeDocument MediaType = "document"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeVideo    MediaType = "video"
	MediaTypeOther    MediaType = "other"
)

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

const (
	MaxFileNameLength   = 255
	MaxMimeTypeLength   = 100
	MaxStorageKeyLength = 512
)

type File struct {
	ID             string
	TenantID       string
	LeadID         *string
	ConversationID string
	ContactID      string
	MessageID      *string
	Name           string
	MediaType      MediaType
	MimeType       string
	SizeBytes      int64
	StorageKey     string
	Direction      Direction
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewFile(
	id, tenantID, conversationID, contactID, name, mimeType, storageKey string,
	sizeBytes int64,
	mediaType MediaType,
	direction Direction,
	leadID, messageID *string,
) (*File, error) {
	if tenantID == "" {
		return nil, ErrTenantIDRequired
	}
	if conversationID == "" {
		return nil, ErrConversationIDRequired
	}
	if contactID == "" {
		return nil, ErrContactIDRequired
	}
	if name == "" {
		return nil, ErrFileNameRequired
	}
	if len(name) > MaxFileNameLength {
		return nil, ErrFileNameTooLong
	}
	if len(mimeType) > MaxMimeTypeLength {
		return nil, ErrMimeTypeTooLong
	}
	if storageKey == "" {
		return nil, ErrStorageKeyRequired
	}
	if sizeBytes < 0 {
		return nil, ErrNegativeSize
	}
	if !IsValidMediaType(mediaType) {
		return nil, ErrInvalidMediaType
	}
	if !IsValidDirection(direction) {
		return nil, ErrInvalidDirection
	}
	now := time.Now()
	return &File{
		ID:             id,
		TenantID:       tenantID,
		LeadID:         leadID,
		ConversationID: conversationID,
		ContactID:      contactID,
		MessageID:      messageID,
		Name:           name,
		MediaType:      mediaType,
		MimeType:       mimeType,
		SizeBytes:      sizeBytes,
		StorageKey:     storageKey,
		Direction:      direction,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func IsValidMediaType(m MediaType) bool {
	switch m {
	case MediaTypeImage, MediaTypeDocument, MediaTypeAudio, MediaTypeVideo, MediaTypeOther:
		return true
	}
	return false
}

func IsValidDirection(d Direction) bool {
	switch d {
	case DirectionInbound, DirectionOutbound:
		return true
	}
	return false
}

// DetectMediaType maps a MIME type (as seen on WhatsApp media) to the
// project-level MediaType enum used for filtering and display. Unknown or
// empty MIME types fall back to MediaTypeOther.
func DetectMediaType(mime string) MediaType {
	mime = strings.ToLower(strings.TrimSpace(mime))
	switch {
	case mime == "":
		return MediaTypeOther
	case strings.HasPrefix(mime, "image/"):
		return MediaTypeImage
	case strings.HasPrefix(mime, "audio/"):
		return MediaTypeAudio
	case strings.HasPrefix(mime, "video/"):
		return MediaTypeVideo
	case mime == "application/pdf",
		mime == "application/msword",
		strings.HasPrefix(mime, "application/vnd."),
		strings.HasPrefix(mime, "text/"):
		return MediaTypeDocument
	}
	return MediaTypeOther
}

// SanitizeFileName returns a safe display name for persistence. It strips
// path separators, null bytes, trims whitespace and rejects traversal
// sentinels (".", ".."). Empty input becomes "arquivo".
func SanitizeFileName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "\x00", "")
	// normalize path separators and keep only the final component
	name = strings.ReplaceAll(name, "\\", "/")
	base := filepath.Base(name)
	base = strings.TrimSpace(base)
	if base == "" || base == "." || base == ".." {
		return "arquivo"
	}
	return base
}
