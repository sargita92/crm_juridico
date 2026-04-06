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

func createContact(t *testing.T, db *gorm.DB, tenantID string) *domain.Contact {
	t.Helper()
	repo := NewGormContactRepository(db)
	contact, err := domain.NewContact(uuid.New().String(), tenantID, "Contato Teste", "+5511999990001", uuid.New().String()+"@s.whatsapp.net")
	require.NoError(t, err)
	require.NoError(t, repo.Create(context.Background(), contact))
	return contact
}

func TestGormConversationRepository_Create_And_FindByID(t *testing.T) {
	db := setupDB(t)
	repo := NewGormConversationRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	ctx := context.Background()

	conv, err := domain.NewConversation(uuid.New().String(), tenant.ID, contact.ID)
	require.NoError(t, err)

	err = repo.Create(ctx, conv)
	require.NoError(t, err)

	found, err := repo.FindByID(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, found.ID)
	assert.Equal(t, tenant.ID, found.TenantID)
	assert.Equal(t, contact.ID, found.ContactID)
	assert.Equal(t, domain.ConversationStatusOpen, found.Status)
	assert.Equal(t, 0, found.UnreadCount)
}

func TestGormConversationRepository_FindByID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := NewGormConversationRepository(db)

	_, err := repo.FindByID(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestGormConversationRepository_FindByContactID(t *testing.T) {
	db := setupDB(t)
	repo := NewGormConversationRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	ctx := context.Background()

	conv, _ := domain.NewConversation(uuid.New().String(), tenant.ID, contact.ID)
	require.NoError(t, repo.Create(ctx, conv))

	found, err := repo.FindByContactID(ctx, tenant.ID, contact.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, found.ID)
}

func TestGormConversationRepository_FindByContactID_ClosedNotReturned(t *testing.T) {
	db := setupDB(t)
	repo := NewGormConversationRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	ctx := context.Background()

	conv, _ := domain.NewConversation(uuid.New().String(), tenant.ID, contact.ID)
	conv.Close()
	require.NoError(t, repo.Create(ctx, conv))

	_, err := repo.FindByContactID(ctx, tenant.ID, contact.ID)
	assert.ErrorIs(t, err, domain.ErrConversationNotFound)
}

func TestGormConversationRepository_Update(t *testing.T) {
	db := setupDB(t)
	repo := NewGormConversationRepository(db)
	tenant := createTenant(t, db)
	contact := createContact(t, db, tenant.ID)
	ctx := context.Background()

	conv, _ := domain.NewConversation(uuid.New().String(), tenant.ID, contact.ID)
	require.NoError(t, repo.Create(ctx, conv))

	conv.RecordMessage(time.Now(), true)
	conv.RecordMessage(time.Now(), true)
	require.NoError(t, repo.Update(ctx, conv))

	found, _ := repo.FindByID(ctx, conv.ID)
	assert.Equal(t, 2, found.UnreadCount)
}

func TestGormConversationRepository_FindByTenantID_OrderedByLastMessage(t *testing.T) {
	db := setupDB(t)
	convRepo := NewGormConversationRepository(db)
	tenant := createTenant(t, db)
	ctx := context.Background()

	c1 := createContact(t, db, tenant.ID)
	c2 := createContact(t, db, tenant.ID)

	conv1, _ := domain.NewConversation(uuid.New().String(), tenant.ID, c1.ID)
	require.NoError(t, convRepo.Create(ctx, conv1))
	// Update to set older timestamp
	conv1.RecordMessage(time.Now().Add(-1*time.Hour), true)
	require.NoError(t, convRepo.Update(ctx, conv1))

	conv2, _ := domain.NewConversation(uuid.New().String(), tenant.ID, c2.ID)
	require.NoError(t, convRepo.Create(ctx, conv2))
	// Update to set newer timestamp
	conv2.RecordMessage(time.Now().Add(1*time.Hour), true)
	require.NoError(t, convRepo.Update(ctx, conv2))

	result, err := convRepo.FindByTenantID(ctx, tenant.ID, domain.ConversationFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Total)
	assert.Len(t, result.Conversations, 2)
	// Most recent first
	assert.Equal(t, conv2.ID, result.Conversations[0].Conversation.ID)
	assert.Equal(t, conv1.ID, result.Conversations[1].Conversation.ID)
}

func TestGormConversationRepository_FindByTenantID_IsolatedByTenant(t *testing.T) {
	db := setupDB(t)
	convRepo := NewGormConversationRepository(db)
	tenantA := createTenant(t, db)
	tenantB := createTenant(t, db)
	ctx := context.Background()

	c1 := createContact(t, db, tenantA.ID)
	conv1, _ := domain.NewConversation(uuid.New().String(), tenantA.ID, c1.ID)
	require.NoError(t, convRepo.Create(ctx, conv1))

	c2 := createContact(t, db, tenantB.ID)
	conv2, _ := domain.NewConversation(uuid.New().String(), tenantB.ID, c2.ID)
	require.NoError(t, convRepo.Create(ctx, conv2))

	resultA, err := convRepo.FindByTenantID(ctx, tenantA.ID, domain.ConversationFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resultA.Total)
	require.Len(t, resultA.Conversations, 1)
	assert.Equal(t, tenantA.ID, resultA.Conversations[0].Conversation.TenantID)

	resultB, err := convRepo.FindByTenantID(ctx, tenantB.ID, domain.ConversationFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), resultB.Total)
	require.Len(t, resultB.Conversations, 1)
	assert.Equal(t, tenantB.ID, resultB.Conversations[0].Conversation.TenantID)
}

func TestGormConversationRepository_FindByTenantID_SearchByContactName(t *testing.T) {
	db := setupDB(t)
	convRepo := NewGormConversationRepository(db)
	contactRepo := NewGormContactRepository(db)
	tenant := createTenant(t, db)
	ctx := context.Background()

	maria, _ := domain.NewContact(uuid.New().String(), tenant.ID, "Maria Silva", "+5511111111111", uuid.New().String()+"@s.whatsapp.net")
	require.NoError(t, contactRepo.Create(ctx, maria))
	joao, _ := domain.NewContact(uuid.New().String(), tenant.ID, "Joao Pereira", "+5522222222222", uuid.New().String()+"@s.whatsapp.net")
	require.NoError(t, contactRepo.Create(ctx, joao))

	conv1, _ := domain.NewConversation(uuid.New().String(), tenant.ID, maria.ID)
	require.NoError(t, convRepo.Create(ctx, conv1))
	conv2, _ := domain.NewConversation(uuid.New().String(), tenant.ID, joao.ID)
	require.NoError(t, convRepo.Create(ctx, conv2))

	result, err := convRepo.FindByTenantID(ctx, tenant.ID, domain.ConversationFilter{Search: "Maria", Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.Total)
	assert.Equal(t, "Maria Silva", result.Conversations[0].Contact.Name)
}

func TestGormConversationRepository_FindByTenantID_EmptyResult(t *testing.T) {
	db := setupDB(t)
	convRepo := NewGormConversationRepository(db)
	tenant := createTenant(t, db)

	result, err := convRepo.FindByTenantID(context.Background(), tenant.ID, domain.ConversationFilter{Page: 1, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, int64(0), result.Total)
	assert.Empty(t, result.Conversations)
}

func TestGormConversationRepository_FindByTenantID_Pagination(t *testing.T) {
	db := setupDB(t)
	convRepo := NewGormConversationRepository(db)
	tenant := createTenant(t, db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		c := createContact(t, db, tenant.ID)
		conv, _ := domain.NewConversation(uuid.New().String(), tenant.ID, c.ID)
		require.NoError(t, convRepo.Create(ctx, conv))
	}

	result, err := convRepo.FindByTenantID(ctx, tenant.ID, domain.ConversationFilter{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, int64(5), result.Total)
	assert.Len(t, result.Conversations, 2)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 2, result.Limit)
}
