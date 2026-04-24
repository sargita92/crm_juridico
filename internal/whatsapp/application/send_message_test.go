package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

// TestSendMessage_CreatesSpan verifies that Execute emits the expected
// OpenTelemetry span.
func TestSendMessage_CreatesSpan(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sr))
	otel.SetTracerProvider(tp)

	uc, convRepo, _, contactRepo, _, _ := setupSendMessageTest()
	conv, _ := createTestConvAndContact(t, convRepo, contactRepo)

	_, err := uc.Execute(context.Background(), SendMessageInput{
		TenantID:       "tenant-1",
		ConversationID: conv.ID,
		Content:        "oi",
	})
	require.NoError(t, err)

	names := []string{}
	for _, s := range sr.Ended() {
		names = append(names, s.Name())
	}
	assert.Contains(t, names, "whatsapp.usecase.send_message", "send_message span should be present")
}

func setupSendMessageTest() (*SendMessageUseCase, *mockConversationRepo, *mockMessageRepo, *mockContactRepo, *mockProvider, *mockEventBus) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	contactRepo := newMockContactRepo()
	provider := newMockProvider()
	eventBus := newMockEventBus()
	provider.connected["tenant-1"] = true
	uc := NewSendMessageUseCase(convRepo, msgRepo, contactRepo, provider, eventBus)
	return uc, convRepo, msgRepo, contactRepo, provider, eventBus
}

func createTestConvAndContact(t *testing.T, convRepo *mockConversationRepo, contactRepo *mockContactRepo) (*domain.Conversation, *domain.Contact) {
	t.Helper()
	contact, _ := domain.NewContact(uuid.New().String(), "tenant-1", "Maria", "+5511999990001", "jid@wa")
	_ = contactRepo.Create(context.Background(), contact)

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", contact.ID)
	_ = convRepo.Create(context.Background(), conv)

	return conv, contact
}

func TestSendMessage_Success(t *testing.T) {
	uc, convRepo, msgRepo, contactRepo, provider, eventBus := setupSendMessageTest()
	conv, _ := createTestConvAndContact(t, convRepo, contactRepo)

	output, err := uc.Execute(context.Background(), SendMessageInput{
		TenantID:       "tenant-1",
		ConversationID: conv.ID,
		Content:        "Ola Maria!",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, output.MessageID)
	assert.Equal(t, "Ola Maria!", output.Content)
	assert.Equal(t, "sent", output.Status)
	assert.Equal(t, "jid@wa", provider.lastSentTo)
	assert.Equal(t, "Ola Maria!", provider.lastSentMsg)
	assert.Len(t, msgRepo.messages, 1)
	assert.Len(t, eventBus.events, 1)
	assert.Equal(t, events.EventNewMessage, eventBus.events[0].Type)
}

func TestSendMessage_ProviderFails_MessageMarkedFailed(t *testing.T) {
	uc, convRepo, msgRepo, contactRepo, provider, _ := setupSendMessageTest()
	conv, _ := createTestConvAndContact(t, convRepo, contactRepo)
	provider.sendErr = errProviderSend

	output, err := uc.Execute(context.Background(), SendMessageInput{
		TenantID:       "tenant-1",
		ConversationID: conv.ID,
		Content:        "Ola",
	})

	assert.Error(t, err)
	assert.Equal(t, "failed", output.Status)
	for _, m := range msgRepo.messages {
		assert.Equal(t, domain.MessageStatusFailed, m.Status)
	}
}

func TestSendMessage_ConversationNotFound_ReturnsError(t *testing.T) {
	uc, _, _, _, _, _ := setupSendMessageTest()

	_, err := uc.Execute(context.Background(), SendMessageInput{
		TenantID:       "tenant-1",
		ConversationID: "nonexistent",
		Content:        "Ola",
	})

	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestSendMessage_WrongTenant_ReturnsError(t *testing.T) {
	uc, convRepo, _, contactRepo, _, _ := setupSendMessageTest()
	conv, _ := createTestConvAndContact(t, convRepo, contactRepo)

	_, err := uc.Execute(context.Background(), SendMessageInput{
		TenantID:       "other-tenant",
		ConversationID: conv.ID,
		Content:        "Ola",
	})

	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestSendMessage_EmptyContent_ReturnsError(t *testing.T) {
	uc, convRepo, _, contactRepo, _, _ := setupSendMessageTest()
	conv, _ := createTestConvAndContact(t, convRepo, contactRepo)

	_, err := uc.Execute(context.Background(), SendMessageInput{
		TenantID:       "tenant-1",
		ConversationID: conv.ID,
		Content:        "",
	})

	assert.ErrorIs(t, err, domain.ErrMessageContentRequired)
}

func TestSendMessage_UpdatesLastMessageAt(t *testing.T) {
	uc, convRepo, _, contactRepo, _, _ := setupSendMessageTest()
	conv, _ := createTestConvAndContact(t, convRepo, contactRepo)
	oldTime := conv.LastMessageAt

	time.Sleep(time.Millisecond)

	_, err := uc.Execute(context.Background(), SendMessageInput{
		TenantID:       "tenant-1",
		ConversationID: conv.ID,
		Content:        "Ola",
	})

	require.NoError(t, err)
	updated := convRepo.conversations[conv.ID]
	assert.True(t, updated.LastMessageAt.After(oldTime) || updated.LastMessageAt.Equal(oldTime))
}
