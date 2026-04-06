package application

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

func newReceiveMessageUC() (*ReceiveMessageUseCase, *mockContactRepo, *mockConversationRepo, *mockMessageRepo, *mockEventBus) {
	contactRepo := newMockContactRepo()
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	eventBus := newMockEventBus()
	uc := NewReceiveMessageUseCase(contactRepo, convRepo, msgRepo, eventBus)
	return uc, contactRepo, convRepo, msgRepo, eventBus
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
	assert.Equal(t, domain.EventNewMessage, eventBus.events[0].Type)
	assert.Equal(t, domain.EventConversationUpdate, eventBus.events[1].Type)
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
