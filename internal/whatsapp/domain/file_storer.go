package domain

import "context"

// FileStorer is an outbound port implemented by the files module. The
// whatsapp module calls it on media events (incoming media from whatsmeow
// or outgoing media sent via the platform) so the files module can persist
// the bytes and metadata. The returned fileID links back to the message.
type FileStorer interface {
	StoreInbound(ctx context.Context, in InboundMediaInput) (fileID string, err error)
	StoreOutbound(ctx context.Context, in OutboundMediaInput) (fileID string, err error)
}

type InboundMediaInput struct {
	TenantID       string
	ConversationID string
	ContactID      string
	MessageID      string
	Name           string
	MimeType       string
	Content        []byte
}

type OutboundMediaInput struct {
	TenantID       string
	ConversationID string
	ContactID      string
	MessageID      string
	Name           string
	MimeType       string
	Content        []byte
}
