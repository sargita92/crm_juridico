package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/shared/events"
	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

func newReceiveMessageUC() (*ReceiveMessageUseCase, *mockContactRepo, *mockConversationRepo, *mockMessageRepo, *mockEventBus) {
	contactRepo := newMockContactRepo()
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	eventBus := newMockEventBus()
	uc := NewReceiveMessageUseCase(contactRepo, convRepo, msgRepo, eventBus, nil)
	return uc, contactRepo, convRepo, msgRepo, eventBus
}

type capturingAIHandler struct {
	called chan context.Context
}

func (h *capturingAIHandler) HandleIncomingMessage(ctx context.Context, _, _, _, _ string) {
	h.called <- ctx
}

type recordingLeadCreator struct {
	calls []struct {
		tenantID, contactID, conversationID, messageText string
	}
}

func (r *recordingLeadCreator) CreateFromConversation(_ context.Context, tenantID, contactID, conversationID, messageText string) error {
	r.calls = append(r.calls, struct {
		tenantID, contactID, conversationID, messageText string
	}{tenantID, contactID, conversationID, messageText})
	return nil
}

func TestReceiveMessage_AIHandler_GetsDetachedContext(t *testing.T) {
	uc, _, _, _, _ := newReceiveMessageUC()
	handler := &capturingAIHandler{called: make(chan context.Context, 1)}
	uc.SetAIHandler(handler)

	parent, cancel := context.WithCancel(context.Background())
	err := uc.Execute(parent, domain.IncomingMessage{
		TenantID: "t-1", SenderJID: "jid@s.whatsapp.net", SenderName: "U",
		SenderPhone: "+55", Content: "oi", WhatsAppMsgID: "wa-1", Timestamp: time.Now(),
	})
	require.NoError(t, err)

	// Simulate HTTP handler returning: parent context cancels.
	cancel()

	select {
	case gotCtx := <-handler.called:
		select {
		case <-gotCtx.Done():
			t.Fatal("AI handler received a cancelled context — goroutine is using request-scoped ctx")
		default:
			// OK: context is alive, AI pipeline can run.
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("AI handler was not called within timeout")
	}
}

func TestReceiveMessage_NewContact_CreatesContactConversationMessage(t *testing.T) {
	uc, contactRepo, convRepo, msgRepo, eventBus := newReceiveMessageUC()

	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID:      "tenant-1",
		SenderJID:     "5511999990001@s.whatsapp.net",
		SenderName:    "Maria",
		SenderPhone:   "+5511999990001",
		Content:       "Ola, preciso de ajuda",
		WhatsAppMsgID: "wa-msg-1",
		Timestamp:     time.Now(),
	})

	require.NoError(t, err)
	assert.Len(t, contactRepo.contacts, 1)
	assert.Len(t, convRepo.conversations, 1)
	assert.Len(t, msgRepo.messages, 1)
	assert.Len(t, eventBus.events, 2) // new-message + conversation-update
	assert.Equal(t, events.EventNewMessage, eventBus.events[0].Type)
	assert.Equal(t, events.EventConversationUpdate, eventBus.events[1].Type)
}

func TestReceiveMessage_ExistingContact_ReusesContact(t *testing.T) {
	uc, contactRepo, _, _, _ := newReceiveMessageUC()

	// First message creates contact
	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID:      "tenant-1",
		SenderJID:     "jid@s.whatsapp.net",
		SenderName:    "Maria",
		SenderPhone:   "+5511999990001",
		Content:       "Primeira",
		WhatsAppMsgID: "wa-1",
		Timestamp:     time.Now(),
	})
	require.NoError(t, err)

	// Second message reuses contact
	err = uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID:      "tenant-1",
		SenderJID:     "jid@s.whatsapp.net",
		SenderName:    "Maria",
		SenderPhone:   "+5511999990001",
		Content:       "Segunda",
		WhatsAppMsgID: "wa-2",
		Timestamp:     time.Now(),
	})
	require.NoError(t, err)
	assert.Len(t, contactRepo.contacts, 1)
}

func TestReceiveMessage_ExistingContact_UpdatesName(t *testing.T) {
	uc, contactRepo, _, _, _ := newReceiveMessageUC()

	_ = uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "tenant-1", SenderJID: "jid@wa", SenderName: "Maria",
		SenderPhone: "+55", Content: "Ola", WhatsAppMsgID: "wa-1", Timestamp: time.Now(),
	})

	_ = uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "tenant-1", SenderJID: "jid@wa", SenderName: "Maria Silva",
		SenderPhone: "+55", Content: "Oi", WhatsAppMsgID: "wa-2", Timestamp: time.Now(),
	})

	for _, c := range contactRepo.contacts {
		assert.Equal(t, "Maria Silva", c.Name)
	}
}

func TestReceiveMessage_DuplicateWhatsAppMsgID_Ignored(t *testing.T) {
	uc, _, _, msgRepo, _ := newReceiveMessageUC()

	event := domain.IncomingMessage{
		TenantID: "tenant-1", SenderJID: "jid@wa", SenderName: "Maria",
		SenderPhone: "+55", Content: "Ola", WhatsAppMsgID: "wa-dup", Timestamp: time.Now(),
	}

	require.NoError(t, uc.Execute(context.Background(), event))
	require.NoError(t, uc.Execute(context.Background(), event))

	assert.Len(t, msgRepo.messages, 1)
}

func TestReceiveMessage_EmptyContent_Ignored(t *testing.T) {
	uc, contactRepo, _, _, _ := newReceiveMessageUC()

	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "tenant-1", SenderJID: "jid@wa", SenderName: "Maria",
		SenderPhone: "+55", Content: "", WhatsAppMsgID: "wa-1", Timestamp: time.Now(),
	})

	require.NoError(t, err)
	assert.Len(t, contactRepo.contacts, 0)
}

func TestReceiveMessage_ExistingConversationWithoutLead_CreatesLead(t *testing.T) {
	// Reproduces the playground bug: fixture/scripts pre-create a conversation
	// (so the contact appears in the playground sidebar) but no lead exists.
	// When the first message arrives, the lead must still be created — otherwise
	// the lead never shows up in the kanban.
	uc, contactRepo, convRepo, _, _ := newReceiveMessageUC()
	leadCreator := &recordingLeadCreator{}
	uc.SetLeadCreator(leadCreator)

	// Pre-create contact + open conversation (simulating playground fixture).
	contact, err := domain.NewContact("c-1", "tenant-1", "Teste Playground", "+5511988880000", "5511988880000@s.whatsapp.net")
	require.NoError(t, err)
	require.NoError(t, contactRepo.Create(context.Background(), contact))

	conv, err := domain.NewConversation("cv-1", "tenant-1", contact.ID)
	require.NoError(t, err)
	require.NoError(t, convRepo.Create(context.Background(), conv))

	err = uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID:      "tenant-1",
		SenderJID:     contact.WhatsAppID,
		SenderName:    contact.Name,
		SenderPhone:   contact.Phone,
		Content:       "Preciso de ajuda com aposentadoria",
		WhatsAppMsgID: "wa-pg-1",
		Timestamp:     time.Now(),
	})
	require.NoError(t, err)

	require.Len(t, leadCreator.calls, 1, "LeadCreator must be invoked even when conversation already exists")
	assert.Equal(t, "tenant-1", leadCreator.calls[0].tenantID)
	assert.Equal(t, contact.ID, leadCreator.calls[0].contactID)
	assert.Equal(t, conv.ID, leadCreator.calls[0].conversationID)
	assert.Equal(t, "Preciso de ajuda com aposentadoria", leadCreator.calls[0].messageText)
}

func TestReceiveMessage_NewConversation_CreatesLead(t *testing.T) {
	uc, _, _, _, _ := newReceiveMessageUC()
	leadCreator := &recordingLeadCreator{}
	uc.SetLeadCreator(leadCreator)

	err := uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID:      "tenant-1",
		SenderJID:     "5511977770000@s.whatsapp.net",
		SenderName:    "Novo Contato",
		SenderPhone:   "+5511977770000",
		Content:       "Ola",
		WhatsAppMsgID: "wa-new-1",
		Timestamp:     time.Now(),
	})
	require.NoError(t, err)

	require.Len(t, leadCreator.calls, 1)
	assert.Equal(t, "Ola", leadCreator.calls[0].messageText)
}

func TestReceiveMessage_IncrementsUnreadCount(t *testing.T) {
	uc, _, convRepo, _, _ := newReceiveMessageUC()

	_ = uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "tenant-1", SenderJID: "jid@wa", SenderName: "Maria",
		SenderPhone: "+55", Content: "Msg 1", WhatsAppMsgID: "wa-1", Timestamp: time.Now(),
	})
	_ = uc.Execute(context.Background(), domain.IncomingMessage{
		TenantID: "tenant-1", SenderJID: "jid@wa", SenderName: "Maria",
		SenderPhone: "+55", Content: "Msg 2", WhatsAppMsgID: "wa-2", Timestamp: time.Now(),
	})

	for _, c := range convRepo.conversations {
		assert.Equal(t, 2, c.UnreadCount)
	}
}
