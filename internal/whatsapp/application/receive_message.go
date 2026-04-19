package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type ReceiveMessageUseCase struct {
	contactRepo      domain.ContactRepository
	conversationRepo domain.ConversationRepository
	messageRepo      domain.MessageRepository
	eventBus         events.EventBus
	leadCreator      domain.LeadCreator
	aiHandler        domain.AIHandler
	fileStorer       domain.FileStorer
	log              *zap.Logger
}

func NewReceiveMessageUseCase(
	contactRepo domain.ContactRepository,
	conversationRepo domain.ConversationRepository,
	messageRepo domain.MessageRepository,
	eventBus events.EventBus,
	log *zap.Logger,
) *ReceiveMessageUseCase {
	if log == nil {
		log = zap.NewNop()
	}
	return &ReceiveMessageUseCase{
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		eventBus:         eventBus,
		log:              log,
	}
}

func (uc *ReceiveMessageUseCase) SetLeadCreator(lc domain.LeadCreator) {
	uc.leadCreator = lc
}

func (uc *ReceiveMessageUseCase) SetAIHandler(handler domain.AIHandler) {
	uc.aiHandler = handler
}

func (uc *ReceiveMessageUseCase) SetFileStorer(fs domain.FileStorer) {
	uc.fileStorer = fs
}

// resolveMessageType maps the incoming media type (or text absence) to the
// MessageType persisted on the Message row.
func resolveMessageType(event domain.IncomingMessage) domain.MessageType {
	if event.Media != nil {
		if t := event.Media.Type; t != "" {
			return t
		}
		return domain.MessageTypeOther
	}
	return domain.MessageTypeText
}

func (uc *ReceiveMessageUseCase) Execute(ctx context.Context, event domain.IncomingMessage) error {
	start := time.Now()
	defer func() {
		messageProcessingDuration.WithLabelValues("incoming").Observe(time.Since(start).Seconds())
	}()

	if event.Content == "" && event.Media == nil {
		return nil // discard empty messages with no media
	}

	// Dedup by WhatsAppMsgID
	if event.WhatsAppMsgID != "" {
		_, err := uc.messageRepo.FindByWhatsAppMsgID(ctx, event.WhatsAppMsgID)
		if err == nil {
			return nil // already processed
		}
		if !errors.Is(err, domain.ErrMessageNotFound) {
			return err
		}
	}

	// Find or create contact
	contact, err := uc.contactRepo.FindByWhatsAppID(ctx, event.TenantID, event.SenderJID)
	if errors.Is(err, domain.ErrContactNotFound) {
		contact, err = domain.NewContact(uuid.New().String(), event.TenantID, event.SenderName, event.SenderPhone, event.SenderJID)
		if err != nil {
			return err
		}
		if err := uc.contactRepo.Create(ctx, contact); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		contact.UpdateName(event.SenderName)
		if err := uc.contactRepo.Update(ctx, contact); err != nil {
			return err
		}
	}

	// Find or create conversation
	conv, err := uc.conversationRepo.FindByContactID(ctx, event.TenantID, contact.ID)
	if errors.Is(err, domain.ErrConversationNotFound) {
		conv, err = domain.NewConversation(uuid.New().String(), event.TenantID, contact.ID)
		if err != nil {
			return err
		}
		if err := uc.conversationRepo.Create(ctx, conv); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	// Create message (caption for media; body for text). Media messages may
	// have empty content — domain.NewMessage allows that for media types.
	msg, err := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, event.Content, resolveMessageType(event), event.WhatsAppMsgID, event.Timestamp)
	if err != nil {
		return err
	}
	if err := uc.messageRepo.Create(ctx, msg); err != nil {
		// UNIQUE constraint on whatsapp_msg_id handles race condition dedup at DB level
		if event.WhatsAppMsgID != "" {
			if _, findErr := uc.messageRepo.FindByWhatsAppMsgID(ctx, event.WhatsAppMsgID); findErr == nil {
				return nil // concurrent duplicate, already persisted
			}
		}
		return err
	}

	// Update conversation
	conv.RecordMessage(event.Timestamp, true)
	if err := uc.conversationRepo.Update(ctx, conv); err != nil {
		return err
	}

	messagesReceivedTotal.WithLabelValues(event.TenantID).Inc()

	// Persist media bytes separately (best-effort). A storage failure must not
	// fail the message ingestion — the text/metadata has already been saved.
	if event.Media != nil && uc.fileStorer != nil {
		_, fsErr := uc.fileStorer.StoreInbound(ctx, domain.InboundMediaInput{
			TenantID:       event.TenantID,
			ConversationID: conv.ID,
			ContactID:      contact.ID,
			MessageID:      msg.ID,
			Name:           event.Media.Name,
			MimeType:       event.Media.MimeType,
			Content:        event.Media.Content,
		})
		if fsErr != nil {
			uc.log.Error("receive_message: media storage failed",
				zap.String("tenant_id", event.TenantID),
				zap.String("conversation_id", conv.ID),
				zap.String("message_id", msg.ID),
				zap.Int("size_bytes", len(event.Media.Content)),
				zap.String("mime_type", event.Media.MimeType),
				zap.Error(fsErr))
		}
	}

	// Publish SSE events
	uc.eventBus.Publish(events.Event{
		Type:     events.EventNewMessage,
		TenantID: event.TenantID,
		Payload:  msg,
	})
	uc.eventBus.Publish(events.Event{
		Type:     events.EventConversationUpdate,
		TenantID: event.TenantID,
		Payload:  conv,
	})

	// Idempotent: no-ops when the lead already exists. Covers contacts whose
	// conversation was created out-of-band (fixtures, playground, manual seed).
	if uc.leadCreator != nil {
		if err := uc.leadCreator.CreateFromConversation(ctx, event.TenantID, contact.ID, conv.ID, event.Content); err != nil {
			uc.log.Error("receive_message: lead creation failed",
				zap.String("tenant_id", event.TenantID),
				zap.String("contact_id", contact.ID),
				zap.String("conversation_id", conv.ID),
				zap.Error(err))
		}
	} else {
		uc.log.Warn("receive_message: leadCreator is nil — lead will not be created",
			zap.String("tenant_id", event.TenantID),
			zap.String("contact_id", contact.ID))
	}

	// Trigger AI handler for all incoming messages.
	// Use a detached context so the AI pipeline survives after the HTTP handler returns
	// (which cancels the request context). Values like trace/request IDs are preserved.
	if uc.aiHandler != nil {
		aiCtx := context.WithoutCancel(ctx)
		go uc.aiHandler.HandleIncomingMessage(aiCtx, event.TenantID, conv.ID, event.SenderPhone, event.Content)
	}

	return nil
}
