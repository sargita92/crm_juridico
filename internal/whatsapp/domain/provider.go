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
	Content       string
	WhatsAppMsgID string
	Timestamp     time.Time
}
