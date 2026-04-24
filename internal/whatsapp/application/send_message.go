package application

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"

	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/shared/observability"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

type SendMessageInput struct {
	TenantID       string
	ConversationID string
	Content        string
}

type SendMessageOutput struct {
	MessageID string
	Content   string
	Timestamp time.Time
	Status    string
}

type SendMessageUseCase struct {
	conversationRepo domain.ConversationRepository
	messageRepo      domain.MessageRepository
	contactRepo      domain.ContactRepository
	provider         domain.WhatsAppProvider
	eventBus         events.EventBus
}

func NewSendMessageUseCase(
	conversationRepo domain.ConversationRepository,
	messageRepo domain.MessageRepository,
	contactRepo domain.ContactRepository,
	provider domain.WhatsAppProvider,
	eventBus events.EventBus,
) *SendMessageUseCase {
	return &SendMessageUseCase{
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
		contactRepo:      contactRepo,
		provider:         provider,
		eventBus:         eventBus,
	}
}

func (uc *SendMessageUseCase) Execute(ctx context.Context, input SendMessageInput) (*SendMessageOutput, error) {
	ctx, span := observability.StartSpan(ctx, "whatsapp.usecase.send_message",
		attribute.String("tenant.id", input.TenantID),
		attribute.String("conversation.id", input.ConversationID),
	)
	defer span.End()

	start := time.Now()
	defer func() {
		messageProcessingDuration.WithLabelValues("outgoing").Observe(time.Since(start).Seconds())
	}()

	// Find conversation and validate tenant
	conv, err := uc.conversationRepo.FindByID(ctx, input.ConversationID)
	if err != nil {
		return nil, err
	}
	if conv.TenantID != input.TenantID {
		return nil, domain.ErrConversationNotFound
	}

	// Find contact
	contact, err := uc.contactRepo.FindByID(ctx, conv.ContactID)
	if err != nil {
		return nil, err
	}

	// Create message (pending)
	now := time.Now()
	msg, err := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionOutgoing, input.Content, domain.MessageTypeText, "", now)
	if err != nil {
		return nil, err
	}
	if err := uc.messageRepo.Create(ctx, msg); err != nil {
		return nil, err
	}

	// Send via provider
	waMsgID, err := uc.provider.SendMessage(ctx, input.TenantID, contact.WhatsAppID, input.Content)
	if err != nil {
		msg.MarkFailed()
		_ = uc.messageRepo.Update(ctx, msg)
		messagesSentTotal.WithLabelValues(input.TenantID, "failed").Inc()
		return &SendMessageOutput{
			MessageID: msg.ID,
			Content:   msg.Content,
			Timestamp: msg.Timestamp,
			Status:    string(msg.Status),
		}, err
	}

	messagesSentTotal.WithLabelValues(input.TenantID, "sent").Inc()
	msg.MarkSent()
	msg.WhatsAppMsgID = waMsgID
	_ = uc.messageRepo.Update(ctx, msg)

	// Update conversation
	conv.RecordMessage(now, false)
	_ = uc.conversationRepo.Update(ctx, conv)

	// Publish events
	uc.eventBus.Publish(events.Event{
		Type:     events.EventNewMessage,
		TenantID: input.TenantID,
		Payload:  msg,
	})

	return &SendMessageOutput{
		MessageID: msg.ID,
		Content:   msg.Content,
		Timestamp: msg.Timestamp,
		Status:    string(msg.Status),
	}, nil
}
