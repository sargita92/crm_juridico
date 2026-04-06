package application

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

func setupGetMessagesTest() (*GetConversationMessagesUseCase, *mockConversationRepo, *mockMessageRepo) {
	convRepo := newMockConversationRepo()
	msgRepo := newMockMessageRepo()
	uc := NewGetConversationMessagesUseCase(convRepo, msgRepo)
	return uc, convRepo, msgRepo
}

func TestGetConversationMessages_Success(t *testing.T) {
	uc, convRepo, msgRepo := setupGetMessagesTest()

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	_ = convRepo.Create(context.Background(), conv)

	msg, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Ola", domain.MessageTypeText, "wa-1", time.Now())
	_ = msgRepo.Create(context.Background(), msg)

	items, err := uc.Execute(context.Background(), GetConversationMessagesInput{
		TenantID:       "tenant-1",
		ConversationID: conv.ID,
		Limit:          50,
	})

	require.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, "Ola", items[0].Content)
	assert.Equal(t, "incoming", items[0].Direction)
}

func TestGetConversationMessages_MarksAsRead(t *testing.T) {
	uc, convRepo, _ := setupGetMessagesTest()

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	conv.RecordMessage(time.Now(), true)
	conv.RecordMessage(time.Now(), true)
	_ = convRepo.Create(context.Background(), conv)
	assert.Equal(t, 2, convRepo.conversations[conv.ID].UnreadCount)

	_, err := uc.Execute(context.Background(), GetConversationMessagesInput{
		TenantID:       "tenant-1",
		ConversationID: conv.ID,
		Limit:          50,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, convRepo.conversations[conv.ID].UnreadCount)
}

func TestGetConversationMessages_AfterID_DoesNotMarkRead(t *testing.T) {
	uc, convRepo, _ := setupGetMessagesTest()

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	conv.RecordMessage(time.Now(), true)
	_ = convRepo.Create(context.Background(), conv)

	_, err := uc.Execute(context.Background(), GetConversationMessagesInput{
		TenantID:       "tenant-1",
		ConversationID: conv.ID,
		AfterID:        "some-id",
		Limit:          50,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, convRepo.conversations[conv.ID].UnreadCount)
}

func TestGetConversationMessages_WrongTenant_ReturnsError(t *testing.T) {
	uc, convRepo, _ := setupGetMessagesTest()

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	_ = convRepo.Create(context.Background(), conv)

	_, err := uc.Execute(context.Background(), GetConversationMessagesInput{
		TenantID:       "other-tenant",
		ConversationID: conv.ID,
		Limit:          50,
	})

	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestGetConversationMessages_ConversationNotFound(t *testing.T) {
	uc, _, _ := setupGetMessagesTest()

	_, err := uc.Execute(context.Background(), GetConversationMessagesInput{
		TenantID:       "tenant-1",
		ConversationID: "nonexistent",
		Limit:          50,
	})

	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}
