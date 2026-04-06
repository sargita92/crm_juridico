package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

func createConversation(t *testing.T, db *gorm.DB, tenantID, contactID string) *domain.Conversation {
	t.Helper()
	repo := NewGormConversationRepository(db)
	conv, err := domain.NewConversation(uuid.New().String(), tenantID, contactID)
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), conv))
	return conv
}

func TestGormMessageRepository_Create_And_FindByConversationID(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	conv := createConversation(t, db, tenant.ID, contact.ID)
	ctx := context.Background()

	msg, err := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Ola, preciso de ajuda", domain.MessageTypeText, "wa-msg-1", time.Now())
	require.NoError(t, err)

	err = repo.Create(ctx, msg)
	require.NoError(t, err)

	messages, err := repo.FindByConversationID(ctx, conv.ID, domain.MessageFilter{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, msg.ID, messages[0].ID)
	assert.Equal(t, "Ola, preciso de ajuda", messages[0].Content)
	assert.Equal(t, domain.MessageDirectionIncoming, messages[0].Direction)
	assert.Equal(t, domain.MessageStatusSent, messages[0].Status)
	assert.Equal(t, "wa-msg-1", messages[0].WhatsAppMsgID)
}

func TestGormMessageRepository_FindByConversationID_OrderedByTimestamp(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	conv := createConversation(t, db, tenant.ID, contact.ID)
	ctx := context.Background()

	now := time.Now()
	msg1, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Primeira", domain.MessageTypeText, "wa-1", now.Add(-2*time.Minute))
	msg2, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionOutgoing, "Segunda", domain.MessageTypeText, "", now.Add(-1*time.Minute))
	msg3, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Terceira", domain.MessageTypeText, "wa-3", now)

	require.NoError(t, repo.Create(ctx, msg1))
	require.NoError(t, repo.Create(ctx, msg2))
	require.NoError(t, repo.Create(ctx, msg3))

	messages, err := repo.FindByConversationID(ctx, conv.ID, domain.MessageFilter{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, messages, 3)
	assert.Equal(t, "Primeira", messages[0].Content)
	assert.Equal(t, "Segunda", messages[1].Content)
	assert.Equal(t, "Terceira", messages[2].Content)
}

func TestGormMessageRepository_FindByConversationID_AfterID(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	conv := createConversation(t, db, tenant.ID, contact.ID)
	ctx := context.Background()

	now := time.Now()
	msg1, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Primeira", domain.MessageTypeText, "wa-1", now.Add(-2*time.Minute))
	msg2, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Segunda", domain.MessageTypeText, "wa-2", now.Add(-1*time.Minute))
	msg3, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Terceira", domain.MessageTypeText, "wa-3", now)

	require.NoError(t, repo.Create(ctx, msg1))
	require.NoError(t, repo.Create(ctx, msg2))
	require.NoError(t, repo.Create(ctx, msg3))

	// Get messages after msg1
	messages, err := repo.FindByConversationID(ctx, conv.ID, domain.MessageFilter{AfterID: msg1.ID, Limit: 50})
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	assert.Equal(t, "Segunda", messages[0].Content)
	assert.Equal(t, "Terceira", messages[1].Content)
}

func TestGormMessageRepository_FindByConversationID_IsolatedByConversation(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)
	tenant := createTenant(t, db)
	c1 := createContact(t, db, tenant.ID)
	c2 := createContact(t, db, tenant.ID)
	conv1 := createConversation(t, db, tenant.ID, c1.ID)
	conv2 := createConversation(t, db, tenant.ID, c2.ID)
	ctx := context.Background()

	msg1, _ := domain.NewMessage(uuid.New().String(), conv1.ID, domain.MessageDirectionIncoming, "Conv 1", domain.MessageTypeText, "wa-1", time.Now())
	msg2, _ := domain.NewMessage(uuid.New().String(), conv2.ID, domain.MessageDirectionIncoming, "Conv 2", domain.MessageTypeText, "wa-2", time.Now())
	require.NoError(t, repo.Create(ctx, msg1))
	require.NoError(t, repo.Create(ctx, msg2))

	messages, err := repo.FindByConversationID(ctx, conv1.ID, domain.MessageFilter{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "Conv 1", messages[0].Content)
}

func TestGormMessageRepository_FindByWhatsAppMsgID(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	conv := createConversation(t, db, tenant.ID, contact.ID)
	ctx := context.Background()

	msg, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Ola", domain.MessageTypeText, "wa-unique-123", time.Now())
	require.NoError(t, repo.Create(ctx, msg))

	found, err := repo.FindByWhatsAppMsgID(ctx, "wa-unique-123")
	require.NoError(t, err)
	assert.Equal(t, msg.ID, found.ID)
}

func TestGormMessageRepository_FindByWhatsAppMsgID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)

	_, err := repo.FindByWhatsAppMsgID(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrMessageNotFound)
}

func TestGormMessageRepository_Update(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	conv := createConversation(t, db, tenant.ID, contact.ID)
	ctx := context.Background()

	msg, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionOutgoing, "Ola", domain.MessageTypeText, "", time.Now())
	require.NoError(t, repo.Create(ctx, msg))
	assert.Equal(t, domain.MessageStatusPending, msg.Status)

	msg.MarkSent()
	require.NoError(t, repo.Update(ctx, msg))

	found, _ := repo.FindByWhatsAppMsgID(ctx, "")
	// Can't find by empty WhatsAppMsgID, so find by conversation
	messages, err := repo.FindByConversationID(ctx, conv.ID, domain.MessageFilter{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, domain.MessageStatusSent, messages[0].Status)
	_ = found
}

func TestGormMessageRepository_DuplicateWhatsAppMsgID_ReturnsError(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	conv := createConversation(t, db, tenant.ID, contact.ID)
	ctx := context.Background()

	msg1, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Msg 1", domain.MessageTypeText, "wa-dup-123", time.Now())
	require.NoError(t, repo.Create(ctx, msg1))

	msg2, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionIncoming, "Msg 2", domain.MessageTypeText, "wa-dup-123", time.Now())
	err := repo.Create(ctx, msg2)
	assert.Error(t, err)
}

func TestGormMessageRepository_NullWhatsAppMsgID_MultipleAllowed(t *testing.T) {
	db := setupDB(t)
	repo := NewGormMessageRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	conv := createConversation(t, db, tenant.ID, contact.ID)
	ctx := context.Background()

	msg1, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionOutgoing, "Msg 1", domain.MessageTypeText, "", time.Now())
	msg2, _ := domain.NewMessage(uuid.New().String(), conv.ID, domain.MessageDirectionOutgoing, "Msg 2", domain.MessageTypeText, "", time.Now().Add(time.Second))
	require.NoError(t, repo.Create(ctx, msg1))
	require.NoError(t, repo.Create(ctx, msg2))

	messages, err := repo.FindByConversationID(ctx, conv.ID, domain.MessageFilter{Limit: 50})
	require.NoError(t, err)
	assert.Len(t, messages, 2)
}
