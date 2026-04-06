package application

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type ReceiveMessageUseCase struct {
	contactRepo      domain.ContactRepository
	conversationRepo domain.ConversationRepository
	messageRepo      domain.MessageRepository
	eventBus         domain.EventBus
	leadCreator      domain.LeadCreator
}

func NewReceiveMessageUseCase(
	contactRepo domain.ContactRepository,
	conversationRepo domain.ConversationRepository,
	messageRepo domain.MessageRepository,
	eventBus domain.EventBus,
) *ReceiveMessageUseCase {
	return &ReceiveMessageUseCase{
		contactRepo:      contactRepo,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		eventBus:         eventBus,
	}
}

func (uc *ReceiveMessageUseCase) SetLeadCreator(lc domain.LeadCreator) {
	uc.leadCreator = lc
}

func (uc *ReceiveMessageUseCase) Execute(ctx context.Context, event domain.IncomingMessage) error {
	start := time.Now()
	defer func() {
		messageProcessingDuration.WithLabelValues("incoming").Observe(time.Since(start).Seconds())
	}()

	if event.Content == "" {
		return nil // discard empty messages
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
	newConversation := false
	conv, err := uc.conversationRepo.FindByContactID(ctx, event.TenantID, contact.ID)
	if errors.Is(err, domain.ErrConversationNotFound) {
		conv, err = domain.NewConversation(uuid.New().String(), event.TenantID, contact.ID)
		if err != nil {
			return err
		}
		if err := uc.conversationRepo.Create(ctx, conv); err != nil {
			return err
		}
		newConversation = true
	} else if err != nil {
		return err
	}

	// Create message
	msg, err := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, event.Content, domain.MessageTypeText, event.WhatsAppMsgID, event.Timestamp)
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

	// Publish SSE events
	uc.eventBus.Publish(domain.Event{
		Type:     domain.EventNewMessage,
		TenantID: event.TenantID,
		Payload:  msg,
	})
	uc.eventBus.Publish(domain.Event{
		Type:     domain.EventConversationUpdate,
		TenantID: event.TenantID,
		Payload:  conv,
	})

	// Create lead in funnel if this is a new conversation
	if newConversation && uc.leadCreator != nil {
		_ = uc.leadCreator.CreateFromConversation(ctx, event.TenantID, contact.ID, conv.ID)
	}

	return nil
}
