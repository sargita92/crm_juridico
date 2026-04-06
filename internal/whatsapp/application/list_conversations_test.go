package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sasrgita/crm-juridico/internal/whatsapp/domain"
)

func TestListConversations_Success(t *testing.T) {
	convRepo := newMockConversationRepo()
	uc := NewListConversationsUseCase(convRepo)

	conv, _ := domain.NewConversation(uuid.New().String(), "tenant-1", "contact-1")
	_ = convRepo.Create(context.Background(), conv)

	output, err := uc.Execute(context.Background(), ListConversationsInput{
		TenantID: "tenant-1",
		Page:     1,
		Limit:    20,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), output.Total)
	assert.Len(t, output.Conversations, 1)
}

func TestListConversations_EmptyResult(t *testing.T) {
	convRepo := newMockConversationRepo()
	uc := NewListConversationsUseCase(convRepo)

	output, err := uc.Execute(context.Background(), ListConversationsInput{
		TenantID: "tenant-1",
		Page:     1,
		Limit:    20,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(0), output.Total)
	assert.Empty(t, output.Conversations)
}

func TestListConversations_IsolatedByTenant(t *testing.T) {
	convRepo := newMockConversationRepo()
	uc := NewListConversationsUseCase(convRepo)

	convA, _ := domain.NewConversation(uuid.New().String(), "tenant-a", "contact-1")
	_ = convRepo.Create(context.Background(), convA)
	convB, _ := domain.NewConversation(uuid.New().String(), "tenant-b", "contact-2")
	_ = convRepo.Create(context.Background(), convB)

	output, err := uc.Execute(context.Background(), ListConversationsInput{
		TenantID: "tenant-a",
		Page:     1,
		Limit:    20,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(1), output.Total)
}
