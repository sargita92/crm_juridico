package domain

import (
	"context"
	"time"
)

// WhatsAppProvider abstrai a conexao com WhatsApp.
// Implementacoes: whatsmeow (dev), Meta Business API (prod futuro).
type WhatsAppProvider interface {
	Connect(ctx context.Context, tenantID string) (qrChan <-chan string, err error)
	Disconnect(ctx context.Context, tenantID string) error
	IsConnected(tenantID string) bool
	SendMessage(ctx context.Context, tenantID, recipientWhatsAppID, content string) (whatsappMsgID string, err error)
	SetMessageHandler(handler IncomingMessageHandler)
}

type IncomingMessageHandler func(ctx context.Context, event IncomingMessage)

type IncomingMessage struct {
	TenantID      string
	SenderJID     string
	SenderName    string
	SenderPhone   string
	Content       string // caption for media, body for text
	WhatsAppMsgID string
	Timestamp     time.Time
	Media         *MediaPayload // non-nil for image/document/audio/video/sticker
}

// MediaPayload carries the bytes and metadata of a downloaded media message.
// It is populated by the provider before invoking the IncomingMessageHandler.
type MediaPayload struct {
	Type     MessageType // image/document/audio/video/sticker/other
	Name     string      // original or derived filename
	MimeType string
	Content  []byte
}
