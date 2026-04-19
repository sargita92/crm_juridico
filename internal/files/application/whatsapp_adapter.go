package application

import (
	"context"

	"github.com/sasrgita/crm-juridico/internal/files/domain"
	waDomain "github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// WhatsAppFileAdapter bridges the whatsapp module's FileStorer port to the
// files module's StoreFileUseCase, translating inbound/outbound callbacks to
// the internal domain direction. This keeps the whatsapp module decoupled
// from the files domain types.
type WhatsAppFileAdapter struct {
	uc *StoreFileUseCase
}

func NewWhatsAppFileAdapter(uc *StoreFileUseCase) *WhatsAppFileAdapter {
	return &WhatsAppFileAdapter{uc: uc}
}

func (a *WhatsAppFileAdapter) StoreInbound(ctx context.Context, in waDomain.InboundMediaInput) (string, error) {
	return a.store(ctx, storeArgs{
		tenantID:       in.TenantID,
		conversationID: in.ConversationID,
		contactID:      in.ContactID,
		messageID:      in.MessageID,
		name:           in.Name,
		mime:           in.MimeType,
		content:        in.Content,
		direction:      domain.DirectionInbound,
	})
}

func (a *WhatsAppFileAdapter) StoreOutbound(ctx context.Context, in waDomain.OutboundMediaInput) (string, error) {
	return a.store(ctx, storeArgs{
		tenantID:       in.TenantID,
		conversationID: in.ConversationID,
		contactID:      in.ContactID,
		messageID:      in.MessageID,
		name:           in.Name,
		mime:           in.MimeType,
		content:        in.Content,
		direction:      domain.DirectionOutbound,
	})
}

type storeArgs struct {
	tenantID, conversationID, contactID string
	messageID                           string
	name, mime                          string
	content                             []byte
	direction                           domain.Direction
}

func (a *WhatsAppFileAdapter) store(ctx context.Context, args storeArgs) (string, error) {
	var msgID *string
	if args.messageID != "" {
		msgID = &args.messageID
	}
	f, err := a.uc.Execute(ctx, StoreFileInput{
		TenantID:       args.tenantID,
		ConversationID: args.conversationID,
		ContactID:      args.contactID,
		MessageID:      msgID,
		Name:           args.name,
		MimeType:       args.mime,
		Direction:      args.direction,
		Content:        args.content,
	})
	if err != nil {
		return "", err
	}
	return f.ID, nil
}
